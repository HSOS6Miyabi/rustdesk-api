package service

import (
	"errors"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// claimAuditNonce records the incoming event ID in the same transaction as the audit.
func claimAuditNonce(tx *gorm.DB, kind, nonce string) (bool,error) {
	nonce=strings.TrimSpace(nonce)
	if nonce=="" { return true,nil }
	if len(nonce)>128 { return false,errors.New("audit nonce exceeds 128 characters") }
	now:=time.Now().Unix()
	key:=kind+":"+nonce
	row:=model.AuditEventNonce{Nonce:key,CreatedAt:now}
	result:=tx.Clauses(clause.OnConflict{DoNothing:true}).Create(&row)
	if result.Error!=nil { return false,result.Error }
	return result.RowsAffected>0,nil
}

// PruneAuditNonces can be called from the scheduled retention worker.
func PruneAuditNonces(now time.Time) error {
	for {
		var nonces []string
		if err:=DB.Model(&model.AuditEventNonce{}).
			Where("created_at < ?",now.Add(-24*time.Hour).Unix()).
			Limit(500).Pluck("nonce",&nonces).Error;err!=nil{return err}
		if len(nonces)==0{return nil}
		if err:=DB.Where("nonce IN ?",nonces).
			Delete(&model.AuditEventNonce{}).Error;err!=nil{return err}
		if len(nonces)<500{return nil}
	}
}

func (as *AuditService) SaveAuditAlarm(event *model.AuditAlarm,nonce,sourceIP string) error {
	if event==nil || strings.TrimSpace(event.PeerId)=="" || event.Typ<0 || event.Typ>127 {
		return errors.New("invalid security alarm")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		fresh,err:=claimAuditNonce(tx,"alarm",nonce)
		if err!=nil {return err}
		if !fresh{return nil}
		event.SourceIp=sourceIP
		_,target,err:=auditParticipants(tx,"",event.PeerId)
		if err!=nil{return err}
		event.ToUserId=target.UserId
		if event.ConnId!=0 {
			var conn model.AuditConn
			err=tx.Where("peer_id = ? AND conn_id = ?",event.PeerId,event.ConnId).
				Order("id DESC").First(&conn).Error
			if err!=nil && !errors.Is(err,gorm.ErrRecordNotFound){return err}
			if conn.Id!=0 {event.FromUserId=conn.FromUserId}
		}
		return tx.Create(event).Error
	})
}

func (as *AuditService) OwnAlarms(userId uint,page,pageSize uint,sort ...string) ([]model.AuditAlarm,int64,error) {
	if userId==0{return nil,0,errors.New("invalid account ID")}
	if page==0{page=1}
	if pageSize==0{pageSize=20}
	if pageSize>100{pageSize=100}
	tx:=DB.Model(&model.AuditAlarm{}).Where("from_user_id = ? OR to_user_id = ?",userId,userId)
	var count int64
	if err:=tx.Count(&count).Error;err!=nil{return nil,0,err}
	var items []model.AuditAlarm
	err:=tx.Order("id DESC").Scopes(Paginate(page,pageSize),SortScope(sort...)).Find(&items).Error
	return items,count,err
}
