package service

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func normalIDWhitelist(value string) (string,error) {
	items:=strings.FieldsFunc(value,func(r rune)bool{return r==','||r=='\n'||r=='\r'})
	if len(items)>256{return "",errors.New("a device ID whitelist supports at most 256 entries")}
	seen:=make(map[string]bool)
	out:=make([]string,0,len(items))
	for _,raw:=range items{
		item:=strings.TrimSpace(raw)
		if item==""{continue}
		if len(item)>128{return "",errors.New("device ID is too long")}
		for _,ch:=range item{
			if !((ch>='0'&&ch<='9')||(ch>='a'&&ch<='z')||
				(ch>='A'&&ch<='Z')||strings.ContainsRune("-_.*?@:",ch)){
				return "",fmt.Errorf("invalid device ID pattern: %q",item)
			}
		}
		if !seen[item]{out=append(out,item);seen[item]=true}
	}
	return strings.Join(out,","),nil
}

func normalIPWhitelist(value string) (string,error) {
	items:=strings.FieldsFunc(value,func(r rune)bool{return r==','||r=='\n'||r=='\r'})
	if len(items)>256{return "",errors.New("an IP whitelist supports at most 256 entries")}
	seen:=make(map[string]bool)
	out:=make([]string,0,len(items))
	for _,raw:=range items{
		item:=strings.TrimSpace(raw)
		if item==""{continue}
		if net.ParseIP(item)==nil{
			_,cidr,err:=net.ParseCIDR(item)
			if err!=nil{return "",fmt.Errorf("invalid IP or CIDR: %q",item)}
			item=cidr.String()
		}
		if !seen[item]{out=append(out,item);seen[item]=true}
	}
	return strings.Join(out,","),nil
}

func ValidateDevicePolicy(policy *model.DevicePolicy) error {
	if policy==nil{return errors.New("policy is required")}
	policy.Name=strings.TrimSpace(policy.Name)
	if len([]rune(policy.Name))<1||len([]rune(policy.Name))>100 {
		return errors.New("policy name must be between 1 and 100 characters")
	}
	ids,err:=normalIDWhitelist(policy.IdWhitelist)
	if err!=nil{return err}
	ips,err:=normalIPWhitelist(policy.IpWhitelist)
	if err!=nil{return err}
	policy.IdWhitelist,policy.IpWhitelist=ids,ips
	if policy.TrustDurationDays!=0 && policy.TrustDurationDays!=1 &&
		policy.TrustDurationDays!=7 && policy.TrustDurationDays!=30 && policy.TrustDurationDays!=90 {
		return errors.New("invalid trusted-device duration")
	}
	return nil
}

func SaveDevicePolicy(policy *model.DevicePolicy) error {
	if err:=ValidateDevicePolicy(policy);err!=nil{return err}
	policy.Revision=time.Now().UnixNano()
	if policy.Id==0{return DB.Create(policy).Error}
	result:=DB.Model(&model.DevicePolicy{}).Where("id = ?",policy.Id).Updates(map[string]interface{}{
		"name":policy.Name,"id_whitelist":policy.IdWhitelist,
		"ip_whitelist":policy.IpWhitelist,"enable_web_rtc":policy.EnableWebRTC,
		"trust_duration_days":policy.TrustDurationDays,"revision":policy.Revision,
	})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return errors.New("device policy was not found")}
	return nil
}

func DeleteDevicePolicy(id uint) error {
	if id==0{return errors.New("invalid policy ID")}
	return DB.Transaction(func(tx *gorm.DB)error{
		result:=tx.Where("id = ?",id).Delete(&model.DevicePolicy{})
		if result.Error!=nil{return result.Error}
		if result.RowsAffected==0{return gorm.ErrRecordNotFound}
		return tx.Where("policy_id = ?",id).Delete(&model.DevicePolicyAssignment{}).Error
	})
}

func SaveDevicePolicyAssignment(item *model.DevicePolicyAssignment) error {
	if item==nil||item.PolicyId==0 || strings.TrimSpace(item.DeviceId)=="" ||
		len(strings.TrimSpace(item.DeviceId))>128{return errors.New("invalid policy assignment")}
	item.DeviceId=strings.TrimSpace(item.DeviceId)
	item.Revision=time.Now().UnixNano()
	var count int64
	if err:=DB.Model(&model.DevicePolicy{}).Where("id = ?",item.PolicyId).Count(&count).Error;err!=nil{return err}
	if count==0{return errors.New("device policy was not found")}
	count=0
	if err:=DB.Model(&model.Peer{}).Where("id = ?",item.DeviceId).Count(&count).Error;err!=nil{return err}
	if count==0{return errors.New("registered device was not found")}
	if item.Id==0{
		return DB.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"device_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"policy_id","revision","updated_at"}),
		}).Create(item).Error
	}
	result:=DB.Model(&model.DevicePolicyAssignment{}).Where("id = ?",item.Id).
		Updates(map[string]interface{}{"policy_id":item.PolicyId,"device_id":item.DeviceId,"revision":item.Revision})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return errors.New("policy assignment was not found")}
	return nil
}

func DeleteDevicePolicyAssignment(id uint) error {
	if id==0{return errors.New("invalid assignment ID")}
	return DB.Where("id = ?",id).Delete(&model.DevicePolicyAssignment{}).Error
}

func EffectiveDevicePolicy(deviceId string,userId uint,admin bool) (map[string]string,bool,error) {
	if len(deviceId)==0||len(deviceId)>128||userId==0{return nil,false,errors.New("invalid device or account")}
	var owner model.Peer
	if err:=DB.Where("id = ?",deviceId).First(&owner).Error;err!=nil{
		return nil,false,errors.New("device was not found")
	}
	if owner.UserId!=userId&&!admin{
		return nil,false,errors.New("device was not found")
	}
	var assignment model.DevicePolicyAssignment
	err:=DB.Where("device_id = ?",deviceId).First(&assignment).Error
	if errors.Is(err,gorm.ErrRecordNotFound){return map[string]string{},false,nil}
	if err!=nil{return nil,false,err}
	var policy model.DevicePolicy
	if err:=DB.First(&policy,assignment.PolicyId).Error;err!=nil{return nil,false,err}
	options:=map[string]string{
		"id-whitelist":policy.IdWhitelist,
		"whitelist":policy.IpWhitelist,
		"enable-webrtc":map[bool]string{true:"Y",false:"N"}[policy.EnableWebRTC],
		"trusted-device-expiration-days":strconv.Itoa(policy.TrustDurationDays),
	}
	return options,true,nil
}


// HeartbeatDeviceStrategy returns RustDesk's native configuration strategy response.
// The client requests it during its normal 15-second heartbeat and handles updates itself.
func HeartbeatDeviceStrategy(peer *model.Peer, modifiedAt int64) (map[string]interface{},error) {
	if peer==nil||peer.RowId==0{return map[string]interface{}{},nil}
	var assignment model.DevicePolicyAssignment
	err:=DB.Where("device_id = ?",peer.Id).First(&assignment).Error
	if errors.Is(err,gorm.ErrRecordNotFound) {
		if modifiedAt==0{return map[string]interface{}{},nil}
		return map[string]interface{}{
			"modified_at":int64(0),
			"strategy":map[string]interface{}{"config_options":map[string]string{
				"id-whitelist":"","whitelist":"",
				"trusted-device-expiration-days":"",
			}},
		},nil
	}
	if err!=nil{return nil,err}
	var policy model.DevicePolicy
	if err:=DB.First(&policy,assignment.PolicyId).Error;err!=nil{
		if errors.Is(err,gorm.ErrRecordNotFound){return map[string]interface{}{},nil}
		return nil,err
	}
	revision:=policy.Revision
	if assignment.Revision>revision{revision=assignment.Revision}
	if revision==0 {
		revision=time.Time(policy.UpdatedAt).UnixNano()
		if a:=time.Time(assignment.UpdatedAt).UnixNano();a>revision{revision=a}
	}
	if revision<=0{revision=1}
	if revision==modifiedAt{return map[string]interface{}{},nil}
	opts:=map[string]string{
		"id-whitelist":policy.IdWhitelist,
		"whitelist":policy.IpWhitelist,
		"trusted-device-expiration-days":strconv.Itoa(policy.TrustDurationDays),
	}
	return map[string]interface{}{
		"modified_at":revision,
		"strategy":map[string]interface{}{"config_options":opts},
	},nil
}
