package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func normalizeAuditPage(page,size uint) (uint,uint) {
	if page==0{page=1}
	if size==0{size=20}
	if size>100{size=100}
	return page,size
}

func (as *AuditService) AllAlarms(page,size uint,peer string,alarmType *int,sort ...string) ([]model.AuditAlarm,int64,error){
	page,size=normalizeAuditPage(page,size)
	tx:=DB.Model(&model.AuditAlarm{})
	if peer!=""{tx=tx.Where("peer_id = ?",peer)}
	if alarmType!=nil{tx=tx.Where("typ = ?",*alarmType)}
	var total int64
	if err:=tx.Count(&total).Error;err!=nil{return nil,0,err}
	var rows []model.AuditAlarm
	err:=tx.Order("id DESC").Scopes(Paginate(page,size),SortScope(sort...)).Find(&rows).Error
	return rows,total,err
}

func (as *AuditService) AllOperations(page,size uint,userId uint,method string,sort ...string) ([]model.AuditOperation,int64,error){
	page,size=normalizeAuditPage(page,size)
	tx:=DB.Model(&model.AuditOperation{})
	if userId!=0{tx=tx.Where("user_id = ?",userId)}
	if method!=""{tx=tx.Where("method = ?",method)}
	var total int64
	if err:=tx.Count(&total).Error;err!=nil{return nil,0,err}
	var rows []model.AuditOperation
	err:=tx.Order("id DESC").Scopes(Paginate(page,size),SortScope(sort...)).Find(&rows).Error
	return rows,total,err
}

// GetAuditRetention returns default unlimited retention when no policy has been saved.
func GetAuditRetention() (*model.AuditRetentionPolicy,error) {
	var policy model.AuditRetentionPolicy
	err:=DB.First(&policy,1).Error
	if errors.Is(err,gorm.ErrRecordNotFound){
		policy.Id=1
		policy.PushSentDays=30
		policy.PushFailedDays=90
		policy.NotificationSentDays=30
		policy.NotificationFailedDays=90
		return &policy,nil
	}
	return &policy,err
}

func SaveAuditRetention(policy *model.AuditRetentionPolicy) error {
	if policy==nil{return errors.New("retention policy is required")}
	for _,days:=range []int{policy.ConnectionDays,policy.FileDays,policy.AlarmDays,policy.LoginDays,policy.OperationDays,
		policy.PushSentDays,policy.PushFailedDays,policy.NotificationSentDays,policy.NotificationFailedDays}{
		if days<0||days>3650{return fmt.Errorf("retention days must be between 0 and 3650")}
	}
	policy.Id=1
	now := time.Now()
	// A map preserves an explicit zero-day value, which disables cleanup.
	values := map[string]interface{}{
		"id": uint(1), "connection_days": policy.ConnectionDays,
		"file_days": policy.FileDays, "alarm_days": policy.AlarmDays,
		"login_days": policy.LoginDays, "operation_days": policy.OperationDays,
		"push_sent_days": policy.PushSentDays, "push_failed_days": policy.PushFailedDays,
		"notification_sent_days": policy.NotificationSentDays,
		"notification_failed_days": policy.NotificationFailedDays,
		"created_at": now, "updated_at": now,
	}
	return DB.Model(&model.AuditRetentionPolicy{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name:"id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"connection_days","file_days","alarm_days","login_days","operation_days",
			"push_sent_days","push_failed_days","notification_sent_days",
			"notification_failed_days","updated_at",
		}),
	}).Create(values).Error
}

func PruneOldAuditRecords(now time.Time) error {
	policy,err:=GetAuditRetention()
	if err!=nil{return err}
	for _,category:=range []struct{model interface{};days int}{
			{&model.AuditConn{},policy.ConnectionDays},
			{&model.AuditFile{},policy.FileDays},
			{&model.AuditAlarm{},policy.AlarmDays},
			{&model.LoginLog{},policy.LoginDays},
			{&model.AuditOperation{},policy.OperationDays},
		}{
			if category.days==0{continue}
			cutoff:=now.Add(-time.Duration(category.days)*24*time.Hour)
			for {
				var ids []uint
				if err:=DB.Model(category.model).Where("created_at < ?",cutoff).
					Order("id ASC").Limit(500).Pluck("id",&ids).Error;err!=nil{return err}
				if len(ids)==0{break}
				if err:=DB.Where("id IN ?",ids).Delete(category.model).Error;err!=nil{return err}
				if len(ids)<500{break}
			}
		}
	return nil
}

// Finished deliveries are removed in short batches. Pending, retrying,
// paused and currently sending tasks are never included in this cleanup.
func pruneDeliveryHistory(modelType interface{}, status string, days int, now time.Time) error {
	if days == 0 || !DB.Migrator().HasTable(modelType) { return nil }
	var deadline interface{}
	column := "sent_at"
	if status == "sent" {
		deadline = now.Add(-time.Duration(days)*24*time.Hour).Unix()
	} else {
		column = "updated_at"
		deadline = now.Add(-time.Duration(days)*24*time.Hour)
	}
	for {
		var ids []uint
		if err := DB.Model(modelType).
			Where("status = ? AND "+column+" < ?", status, deadline).
			Order("id ASC").Limit(500).Pluck("id", &ids).Error; err != nil { return err }
		if len(ids) == 0 { return nil }
		if err := DB.Where("id IN ? AND status = ? AND "+column+" < ?", ids, status, deadline).
			Delete(modelType).Error; err != nil { return err }
		if len(ids) < 500 { return nil }
	}
}

func PruneOldDeliveryRecords(now time.Time) error {
	policy, err := GetAuditRetention()
	if err != nil { return err }
	for _, category := range []struct{ table interface{}; status string; days int }{
		{&model.PushDelivery{}, "sent", policy.PushSentDays},
		{&model.PushDelivery{}, "failed", policy.PushFailedDays},
		{&model.NotificationDelivery{}, "sent", policy.NotificationSentDays},
		{&model.NotificationDelivery{}, "failed", policy.NotificationFailedDays},
	} {
		if err := pruneDeliveryHistory(category.table, category.status, category.days, now); err != nil {
			return err
		}
	}
	return nil
}

func StartAuditMaintenance() {
	launchServiceWorker(func(ctx context.Context) {
		run:=func(){
			if err:=PruneOldAuditRecords(time.Now());err!=nil&&Logger!=nil{
				Logger.WithError(err).Warn("audit record cleanup failed")
			}
			if err:=PruneOldDeliveryRecords(time.Now());err!=nil&&Logger!=nil{
				Logger.WithError(err).Warn("delivery history cleanup failed")
			}
			if err:=PruneAuditNonces(time.Now());err!=nil&&Logger!=nil{
				Logger.WithError(err).Warn("audit nonce cleanup failed")
			}
		}
		run()
		ticker:=time.NewTicker(24*time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-ticker.C: run()
			}
		}
	})
}
