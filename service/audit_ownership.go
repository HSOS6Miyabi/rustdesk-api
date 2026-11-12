package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type auditParticipant struct {
	UserId uint
	Account string
	DeviceName string
}

func auditParticipants(tx *gorm.DB, fromId, toId string) (auditParticipant, auditParticipant, error) {
	from, to := auditParticipant{}, auditParticipant{}
	ids := make([]string, 0, 2)
	if fromId != "" { ids = append(ids, fromId) }
	if toId != "" && toId != fromId { ids = append(ids, toId) }
	if len(ids) == 0 { return from, to, nil }
	var peers []model.Peer
	if err := tx.Where("id IN ?", ids).Order("last_online_time DESC, row_id DESC").Find(&peers).Error; err != nil {
		return from, to, err
	}
	byId := make(map[string]model.Peer)
	userIds := make([]uint, 0)
	for _, peer := range peers {
		if _, exists := byId[peer.Id]; exists { continue }
		byId[peer.Id] = peer
		if peer.UserId != 0 { userIds = append(userIds, peer.UserId) }
	}
	users := make(map[uint]string)
	if len(userIds) > 0 {
		var names []model.User
		if err := tx.Select("id,username").Where("id IN ?", userIds).Find(&names).Error; err != nil {
			return from, to, err
		}
		for _, u := range names { users[u.Id] = u.Username }
	}
	if peer, ok := byId[fromId]; ok {
		from = auditParticipant{peer.UserId, users[peer.UserId], peer.Hostname}
	}
	if peer, ok := byId[toId]; ok {
		to = auditParticipant{peer.UserId, users[peer.UserId], peer.Hostname}
	}
	return from, to, nil
}

func applyAuditConnParticipants(row *model.AuditConn, from, to auditParticipant) {
	if row.FromUserId == 0 { row.FromUserId = from.UserId }
	if row.ToUserId == 0 { row.ToUserId = to.UserId }
	if row.FromAccount == "" { row.FromAccount = from.Account }
	if row.ToAccount == "" { row.ToAccount = to.Account }
	if row.FromDeviceName == "" { row.FromDeviceName = from.DeviceName }
	if row.ToDeviceName == "" { row.ToDeviceName = to.DeviceName }
}

func applyAuditFileParticipants(row *model.AuditFile, from, to auditParticipant) {
	if row.FromUserId == 0 { row.FromUserId = from.UserId }
	if row.ToUserId == 0 { row.ToUserId = to.UserId }
	if row.FromAccount == "" { row.FromAccount = from.Account }
	if row.ToAccount == "" { row.ToAccount = to.Account }
	if row.FromDeviceName == "" { row.FromDeviceName = from.DeviceName }
	if row.ToDeviceName == "" { row.ToDeviceName = to.DeviceName }
}

// SaveAuditConnection combines start, detail and close events in one record.
// Empty fields in a later event never erase previously received participant information.
func (as *AuditService) SaveAuditConnection(event *model.AuditConn) error {
	if event == nil || strings.TrimSpace(event.PeerId) == "" || event.ConnId == 0 {
		return errors.New("connection ID and target device ID are required")
	}
	if event.Action != "" && event.Action != model.AuditActionNew && event.Action != model.AuditActionClose {
		return errors.New("unsupported connection event")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		fresh, err := claimAuditNonce(tx, "conn", event.EventNonce)
		if err != nil { return err }
		if !fresh { return nil }
		previous := &model.AuditConn{}
		err = tx.Where("peer_id = ? AND conn_id = ?", event.PeerId, event.ConnId).Order("id DESC").First(previous).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) { return err }

		// Resolve both participants only after the source device becomes known.
		lookupSource := event.FromPeer
		if lookupSource == "" && previous.Id > 0 { lookupSource = previous.FromPeer }
		from, to, err := auditParticipants(tx, lookupSource, event.PeerId)
		if err != nil { return err }

		if previous.Id == 0 {
            row := *event
            applyAuditConnParticipants(&row, from, to)
            if row.Action == "" { row.Action = model.AuditActionNew }
            if row.Action == model.AuditActionClose { row.CloseTime = time.Now().Unix() }
            if err := tx.Create(&row).Error; err != nil { return err }
            event.Id = row.Id
            event.AuthenticatedNew = event.Action == "" && event.FromPeer != "" && row.CloseTime == 0
            return nil
        }

		changes := make(map[string]interface{})
		if event.Action == model.AuditActionClose {
			changes["close_time"] = time.Now().Unix()
		}
		if event.FromPeer != "" { changes["from_peer"] = event.FromPeer }
		if event.FromName != "" { changes["from_name"] = event.FromName }
		if event.SessionId != "" && event.SessionId != "0" { changes["session_id"] = event.SessionId }
		if event.Ip != "" { changes["ip"] = event.Ip }
		if event.Uuid != "" { changes["uuid"] = event.Uuid }
		if event.Type != 0 { changes["type"] = event.Type }
		if event.ConnAuditRef != "" { changes["conn_audit_ref"] = event.ConnAuditRef }
		if event.PrimaryAuth != 0 { changes["primary_auth"] = event.PrimaryAuth }
		if event.TwoFactor != 0 { changes["two_factor"] = event.TwoFactor }
		if previous.FromUserId == 0 && from.UserId != 0 { changes["from_user_id"] = from.UserId }
		if previous.ToUserId == 0 && to.UserId != 0 { changes["to_user_id"] = to.UserId }
		if previous.FromAccount == "" && from.Account != "" { changes["from_account"] = from.Account }
		if previous.ToAccount == "" && to.Account != "" { changes["to_account"] = to.Account }
		if previous.FromDeviceName == "" && from.DeviceName != "" { changes["from_device_name"] = from.DeviceName }
		if previous.ToDeviceName == "" && to.DeviceName != "" { changes["to_device_name"] = to.DeviceName }
		if len(changes) == 0 { return nil }
        authenticated := event.Action == "" && event.FromPeer != "" &&
            previous.FromPeer == "" && previous.CloseTime == 0
        query := tx.Model(&model.AuditConn{}).Where("id = ?", previous.Id)
        if authenticated {
            query = query.Where("from_peer = ? AND close_time = 0", "")
        }
        result := query.Updates(changes)
        if result.Error != nil { return result.Error }
        if authenticated && result.RowsAffected == 1 {
            event.AuthenticatedNew = true
            event.Id = previous.Id
        } else if event.Action != "" {
            event.Id = previous.Id
        }
        return nil
	})
}

func (as *AuditService) SaveAuditFile(event *model.AuditFile) error {
	if event == nil || strings.TrimSpace(event.PeerId) == "" {
		return errors.New("target device ID is required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		fresh, err := claimAuditNonce(tx, "file", event.EventNonce)
		if err != nil { return err }
		if !fresh { return nil }
		from, to, err := auditParticipants(tx, event.FromPeer, event.PeerId)
		if err != nil { return err }
		applyAuditFileParticipants(event, from, to)
		return tx.Create(event).Error
	})
}

// fill missing display details from current device information. Historical
// rows retain their stored identifiers and are never assigned a new owner in DB.
func (as *AuditService) EnrichConnections(rows []*model.AuditConn, viewerId uint) error {
	ids := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		if row.FromPeer != "" { ids = append(ids, row.FromPeer) }
		if row.PeerId != "" { ids = append(ids, row.PeerId) }
	}
	peers, users, err := auditDisplayIndex(ids)
	if err != nil { return err }
	for _, row := range rows {
		from, to := participantFromIndex(peers, users, row.FromPeer), participantFromIndex(peers, users, row.PeerId)
		applyAuditConnParticipants(row, from, to)
		row.Direction = auditDirection(viewerId, row.FromUserId, row.ToUserId)
	}
	return nil
}

func (as *AuditService) EnrichFiles(rows []*model.AuditFile, viewerId uint) error {
	ids := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		if row.FromPeer != "" { ids = append(ids, row.FromPeer) }
		if row.PeerId != "" { ids = append(ids, row.PeerId) }
	}
	peers, users, err := auditDisplayIndex(ids)
	if err != nil { return err }
	for _, row := range rows {
		from, to := participantFromIndex(peers, users, row.FromPeer), participantFromIndex(peers, users, row.PeerId)
		applyAuditFileParticipants(row, from, to)
		row.Direction = auditDirection(viewerId, row.FromUserId, row.ToUserId)
	}
	return nil
}

func auditDirection(viewerId, fromId, toId uint) string {
	if viewerId == 0 { return "" }
	if fromId == viewerId && toId == viewerId { return "both" }
	if fromId == viewerId { return "outgoing" }
	if toId == viewerId { return "incoming" }
	return ""
}

func auditDisplayIndex(ids []string) (map[string]model.Peer, map[uint]string, error) {
	peersById := make(map[string]model.Peer)
	userNames := make(map[uint]string)
	if len(ids) == 0 { return peersById, userNames, nil }
	var peers []model.Peer
	if err := DB.Where("id IN ?", ids).Order("last_online_time DESC, row_id DESC").Find(&peers).Error; err != nil {
		return nil, nil, err
	}
	owners := make([]uint, 0)
	for _, peer := range peers {
		if _, exists := peersById[peer.Id]; exists { continue }
		peersById[peer.Id] = peer
		if peer.UserId != 0 { owners = append(owners, peer.UserId) }
	}
	if len(owners) > 0 {
		var users []model.User
		if err := DB.Select("id,username").Where("id IN ?", owners).Find(&users).Error; err != nil {
			return nil, nil, err
		}
		for _, u := range users { userNames[u.Id] = u.Username }
	}
	return peersById, userNames, nil
}

func participantFromIndex(peers map[string]model.Peer, users map[uint]string, id string) auditParticipant {
	p, ok := peers[id]
	if !ok { return auditParticipant{} }
	return auditParticipant{p.UserId, users[p.UserId], p.Hostname}
}

func auditPersonalScope(tx *gorm.DB, userId uint) *gorm.DB {
	return tx.Where(
		"(from_user_id = ? OR to_user_id = ? OR (from_user_id = 0 AND from_peer IN (SELECT id FROM peers WHERE user_id = ?)) OR (to_user_id = 0 AND peer_id IN (SELECT id FROM peers WHERE user_id = ?)))",
		userId, userId, userId, userId,
	)
}

// personalAuditDirectionExpression matches the personal audit direction shown
// after enrichment, including records whose original owner IDs were absent.
func personalAuditDirectionExpression(table string) string {
 from:=fmt.Sprintf("(CASE WHEN %s.from_user_id != 0 THEN %s.from_user_id ELSE COALESCE((SELECT user_id FROM peers WHERE peers.id = %s.from_peer ORDER BY last_online_time DESC, row_id DESC LIMIT 1),0) END)",table,table,table)
 to:=fmt.Sprintf("(CASE WHEN %s.to_user_id != 0 THEN %s.to_user_id ELSE COALESCE((SELECT user_id FROM peers WHERE peers.id = %s.peer_id ORDER BY last_online_time DESC, row_id DESC LIMIT 1),0) END)",table,table,table)
 return "CASE WHEN "+from+" = ? AND "+to+" = ? THEN 'both' WHEN "+from+" = ? THEN 'outgoing' WHEN "+to+" = ? THEN 'incoming' ELSE '' END"
}

func (as *AuditService) OwnConnections(userId uint, page, pageSize uint,sort ...string) (*model.AuditConnList, error) {
	res := &model.AuditConnList{}
	if userId == 0 { return res, errors.New("user not found") }
	if page == 0 { page = 1 }
	if pageSize == 0 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	res.Page, res.PageSize = int64(page), int64(pageSize)
	tx := auditPersonalScope(DB.Model(&model.AuditConn{}), userId)
	if err := tx.Count(&res.Total).Error; err != nil { return nil, err }
	ordered:=tx.Order("id DESC")
	if len(sort)>=2 && sort[0]=="direction" {
		ApplyComputedSort(ordered,personalAuditDirectionExpression("audit_conns"),[]interface{}{userId,userId,userId,userId},sort[1],"audit_conns.id")
	}
	if err := ordered.Scopes(Paginate(page, pageSize),SortScope(sort...)).Find(&res.AuditConns).Error; err != nil { return nil, err }
	if err := as.EnrichConnections(res.AuditConns, userId); err != nil { return nil, err }
	return res, nil
}

func (as *AuditService) OwnFiles(userId uint, page, pageSize uint,sort ...string) (*model.AuditFileList, error) {
	res := &model.AuditFileList{}
	if userId == 0 { return res, errors.New("user not found") }
	if page == 0 { page = 1 }
	if pageSize == 0 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	res.Page, res.PageSize = int64(page), int64(pageSize)
	tx := auditPersonalScope(DB.Model(&model.AuditFile{}), userId)
	if err := tx.Count(&res.Total).Error; err != nil { return nil, err }
	ordered:=tx.Order("id DESC")
	if len(sort)>=2 && sort[0]=="direction" {
		ApplyComputedSort(ordered,personalAuditDirectionExpression("audit_files"),[]interface{}{userId,userId,userId,userId},sort[1],"audit_files.id")
	}
	if err := ordered.Scopes(Paginate(page, pageSize),SortScope(sort...)).Find(&res.AuditFiles).Error; err != nil { return nil, err }
	if err := as.EnrichFiles(res.AuditFiles, userId); err != nil { return nil, err }
	return res, nil
}
