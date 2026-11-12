package service

import (
    "strings"
    "strconv"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

// ActiveConnectionDetails adds the participants available from recorded connection
// audits and the latest device ownership records.
type ActiveConnectionDetails struct {
    model.DeviceActiveConnection
    FromPeer string `json:"from_peer"`
    FromDeviceName string `json:"from_device_name"`
    FromName string `json:"from_name"`
    FromUserID uint `json:"from_user_id"`
    FromAccount string `json:"from_account"`
    ToDeviceName string `json:"to_device_name"`
    ToUserID uint `json:"to_user_id"`
    ToAccount string `json:"to_account"`
    IP string `json:"ip"`
    ParticipantRecorded bool `json:"participant_recorded"`
}

// EnrichActiveConnections uses two bounded queries to identify the participants
// without performing one database lookup for each active connection.
func EnrichActiveConnections(rows []model.DeviceActiveConnection) ([]ActiveConnectionDetails, error) {
    out := make([]ActiveConnectionDetails, len(rows))
    if len(rows) == 0 { return out, nil }
    conditions := make([]string, 0, len(rows))
    args := make([]interface{}, 0, len(rows)*2)
    deviceIDs := make(map[string]bool, len(rows)*2)
    for i, row := range rows {
        out[i].DeviceActiveConnection = row
        deviceIDs[row.DeviceId] = true
        conditions = append(conditions, "(peer_id = ? AND conn_id = ?)")
        args = append(args, row.DeviceId, row.ConnId)
    }
    var audits []model.AuditConn
    err := DB.Model(&model.AuditConn{}).
        Select("id, peer_id, conn_id, from_peer, from_name, from_user_id, to_user_id, from_account, to_account, from_device_name, to_device_name, ip, close_time").
        Where(strings.Join(conditions, " OR "), args...).
        Order("id DESC").Limit(len(rows)*4).Find(&audits).Error
    if err != nil { return nil, err }
    byConnection := make(map[string]model.AuditConn, len(rows))
    connectionKey := func(device string, conn int64) string { return device + "\x00" + strconv.FormatInt(conn, 10) }
    for _, record := range audits {
        key := connectionKey(record.PeerId, record.ConnId)
        if _, exists := byConnection[key]; exists { continue }
        byConnection[key] = record
        if record.CloseTime == 0 && record.FromPeer != "" { deviceIDs[record.FromPeer] = true }
    }
    for i := range out {
        record, found := byConnection[connectionKey(out[i].DeviceId, int64(out[i].ConnId))]
        if !found || record.CloseTime != 0 { continue }
        out[i].ParticipantRecorded = true
        out[i].FromPeer, out[i].FromName = record.FromPeer, record.FromName
        out[i].FromAccount, out[i].ToAccount = record.FromAccount, record.ToAccount
        out[i].FromUserID, out[i].ToUserID = record.FromUserId, record.ToUserId
        out[i].FromDeviceName, out[i].ToDeviceName = record.FromDeviceName, record.ToDeviceName
        out[i].IP = record.Ip
    }
    ids := make([]string, 0, len(deviceIDs))
    for id := range deviceIDs { if id != "" { ids = append(ids, id) } }
    var peers []model.Peer
    if err := DB.Model(&model.Peer{}).Select("id, user_id, alias, hostname, last_online_time, row_id").
        Where("id IN ?", ids).Order("last_online_time DESC, row_id DESC").Find(&peers).Error; err != nil {
        return nil, err
    }
    byDevice := make(map[string]model.Peer, len(peers))
    ownerIDs := make(map[uint]bool)
    for _, peer := range peers {
        if _, ok := byDevice[peer.Id]; ok { continue }
        byDevice[peer.Id] = peer
        if peer.UserId != 0 { ownerIDs[peer.UserId] = true }
    }
    users := make(map[uint]string, len(ownerIDs))
    if len(ownerIDs) > 0 {
        ids := make([]uint, 0, len(ownerIDs))
        for id := range ownerIDs { ids = append(ids, id) }
        var accounts []model.User
        if err := DB.Model(&model.User{}).Select("id,username").Where("id IN ?", ids).Find(&accounts).Error; err != nil {
            return nil, err
        }
        for _, account := range accounts { users[account.Id] = account.Username }
    }
    displayName := func(p model.Peer) string {
        if p.Alias != "" { return p.Alias }
        return p.Hostname
    }
    for i := range out {
        row := &out[i]
        if peer, ok := byDevice[row.DeviceId]; ok {
            if row.ToDeviceName == "" { row.ToDeviceName = displayName(peer) }
            if row.ToUserID == 0 { row.ToUserID = peer.UserId }
            if row.ToAccount == "" { row.ToAccount = users[peer.UserId] }
        }
        if peer, ok := byDevice[row.FromPeer]; ok {
            if row.FromDeviceName == "" { row.FromDeviceName = displayName(peer) }
            if row.FromUserID == 0 { row.FromUserID = peer.UserId }
            if row.FromAccount == "" { row.FromAccount = users[peer.UserId] }
        }
    }
    return out, nil
}
