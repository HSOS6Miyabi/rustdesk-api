package service

import (
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

func (as *AuditService) AllAlarms(page,size uint,peer string,alarmType *int) ([]model.AuditAlarm,int64,error){
	page,size=normalizeAuditPage(page,size)
	tx:=DB.Model(&model.AuditAlarm{})
	if peer!=""{tx=tx.Where("peer_id = ?",peer)}
	if alarmType!=nil{tx=tx.Where("typ = ?",*alarmType)}
	var total int64
	if err:=tx.Count(&total).Error;err!=nil{return nil,0,err}
	var rows []model.AuditAlarm
	err:=tx.Order("id DESC").Scopes(Paginate(page,size)).Find(&rows).Error
	return rows,total,err
}

func (as *AuditService) AllOperations(page,size uint,userId uint,method string) ([]model.AuditOperation,int64,error){
	page,size=normalizeAuditPage(page,size)
	tx:=DB.Model(&model.AuditOperation{})
	if userId!=0{tx=tx.Where("user_id = ?",userId)}
	if method!=""{tx=tx.Where("method = ?",method)}
	var total int64
	if err:=tx.Count(&total).Error;err!=nil{return nil,0,err}
	var rows []model.AuditOperation
	err:=tx.Order("id DESC").Scopes(Paginate(page,size)).Find(&rows).Error
	return rows,total,err
}

// GetAuditRetention returns default unlimited retention when no policy has been saved.
func GetAuditRetention() (*model.AuditRetentionPolicy,error) {
	var policy model.AuditRetentionPolicy
	err:=DB.First(&policy,1).Error
	if errors.Is(err,gorm.ErrRecordNotFound){policy.Id=1;return &policy,nil}
	return &policy,err
}

func SaveAuditRetention(policy *model.AuditRetentionPolicy) error {
	if policy==nil{return errors.New("retention policy is required")}
	for _,days:=range []int{policy.ConnectionDays,policy.FileDays,policy.AlarmDays,policy.LoginDays,policy.OperationDays}{
		if days<0||days>3650{return fmt.Errorf("retention days must be between 0 and 3650")}
	}
	policy.Id=1
	return DB.Clauses(clause.OnConflict{
		Columns:[]clause.Column{{Name:"id"}},
		DoUpdates:clause.AssignmentColumns([]string{
			"connection_days","file_days","alarm_days","login_days","operation_days","updated_at",
		}),
	}).Create(policy).Error
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

func StartAuditMaintenance() {
	go func(){
		run:=func(){
			if err:=PruneOldAuditRecords(time.Now());err!=nil&&Logger!=nil{
				Logger.WithError(err).Warn("audit record cleanup failed")
			}
			if err:=PruneAuditNonces(time.Now());err!=nil&&Logger!=nil{
				Logger.WithError(err).Warn("audit nonce cleanup failed")
			}
		}
		run()
		ticker:=time.NewTicker(24*time.Hour)
		defer ticker.Stop()
		for range ticker.C{run()}
	}()
}
