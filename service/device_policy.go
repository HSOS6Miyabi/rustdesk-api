package service

import (
	"errors"
	"sort"
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

func validPolicyValue(value string, allowed ...string) bool {
	if value == "" { return true }
	for _, candidate := range allowed { if value == candidate { return true } }
	return false
}

func extraClientPolicyOptions(policy *model.DevicePolicy) map[string]string {
	values:=map[string]string{
		"access-mode":policy.AccessMode,
		"enable-keyboard":policy.EnableKeyboard,
		"enable-clipboard":policy.EnableClipboard,
		"enable-file-transfer":policy.EnableFileTransfer,
		"approve-mode":policy.ApprovalMode,
	}
	result:=make(map[string]string)
	for _,name:=range strings.Split(policy.ManagedClientKeys,","){
		if value,ok:=values[name];ok{result[name]=value}
	}
	return result
}

func mergeManagedClientKeys(policy *model.DevicePolicy, previous string) {
	seen:=make(map[string]bool)
	for _,name:=range strings.Split(previous,","){
		switch name {
		case "access-mode","enable-keyboard","enable-clipboard","enable-file-transfer","approve-mode":
			seen[name]=true
		}
	}
	for name,value:=range map[string]string{
		"access-mode":policy.AccessMode,
		"enable-keyboard":policy.EnableKeyboard,
		"enable-clipboard":policy.EnableClipboard,
		"enable-file-transfer":policy.EnableFileTransfer,
		"approve-mode":policy.ApprovalMode,
	}{
		if value!=""{seen[name]=true}
	}
	keys:=make([]string,0,len(seen))
	for name:=range seen{keys=append(keys,name)}
	sort.Strings(keys)
	policy.ManagedClientKeys=strings.Join(keys,",")
}

func ValidateDevicePolicy(policy *model.DevicePolicy) error {
	if policy==nil{return errors.New("policy is required")}
	policy.Name=strings.TrimSpace(policy.Name)
	if policy.Status==0{policy.Status=1}
	if policy.Status!=1 && policy.Status!=2{return errors.New("invalid device policy status")}
	if len([]rune(policy.Name))<1||len([]rune(policy.Name))>100 {
		return errors.New("policy name must be between 1 and 100 characters")
	}
	ids,err:=normalIDWhitelist(policy.IdWhitelist)
	if err!=nil{return err}
	ips,err:=normalIPWhitelist(policy.IpWhitelist)
	if err!=nil{return err}
	policy.IdWhitelist,policy.IpWhitelist=ids,ips
	if !validPolicyValue(policy.AccessMode,"custom","full","view") ||
		!validPolicyValue(policy.EnableKeyboard,"Y","N") ||
		!validPolicyValue(policy.EnableClipboard,"Y","N") ||
		!validPolicyValue(policy.EnableFileTransfer,"Y","N") ||
		!validPolicyValue(policy.ApprovalMode,"password","click","password-click") {
		return errors.New("invalid client security policy option")
	}
	if policy.TrustDurationDays!=0 && policy.TrustDurationDays!=1 &&
		policy.TrustDurationDays!=7 && policy.TrustDurationDays!=30 && policy.TrustDurationDays!=90 {
		return errors.New("invalid trusted-device duration")
	}
	return nil
}

func SaveDevicePolicy(policy *model.DevicePolicy) error {
	if err:=ValidateDevicePolicy(policy);err!=nil{return err}
	previous:=""
	if policy.Id>0 {
		var saved model.DevicePolicy
		if err:=DB.Where("id = ?",policy.Id).First(&saved).Error;err!=nil{return err}
		previous=saved.ManagedClientKeys
	}
	mergeManagedClientKeys(policy,previous)
	policy.Revision=time.Now().UnixNano()
	if policy.Id==0{
		err:=DB.Create(policy).Error
		if err==nil {invalidatePolicyActivity()}
		return err
	}
	result:=DB.Model(&model.DevicePolicy{}).Where("id = ?",policy.Id).Updates(map[string]interface{}{
		"name":policy.Name,"status":policy.Status,"id_whitelist":policy.IdWhitelist,
		"ip_whitelist":policy.IpWhitelist,"enable_web_rtc":policy.EnableWebRTC,
		"trust_duration_days":policy.TrustDurationDays,"revision":policy.Revision,
		"access_mode":policy.AccessMode,
		"enable_keyboard":policy.EnableKeyboard,
		"enable_clipboard":policy.EnableClipboard,
		"enable_file_transfer":policy.EnableFileTransfer,
		"approval_mode":policy.ApprovalMode,
		"managed_client_keys":policy.ManagedClientKeys,
	})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return errors.New("device policy was not found")}
	invalidatePolicyActivity()
	return nil
}

func DeleteDevicePolicy(id uint) error {
	if id==0{return errors.New("invalid policy ID")}
	err:=DB.Transaction(func(tx *gorm.DB)error{
		result:=tx.Where("id = ?",id).Delete(&model.DevicePolicy{})
		if result.Error!=nil{return result.Error}
		if result.RowsAffected==0{return gorm.ErrRecordNotFound}
		if err:=tx.Where("policy_id = ?",id).Delete(&model.DevicePolicyAssignment{}).Error;err!=nil{return err}
		if err:=tx.Where("policy_id = ?",id).Delete(&model.DevicePolicyUserAssignment{}).Error;err!=nil{return err}
		return tx.Where("policy_id = ?",id).Delete(&model.DevicePolicyGroupAssignment{}).Error
	})
	if err==nil {invalidatePolicyActivity()}
	return err
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
		err:=DB.Clauses(clause.OnConflict{
			Columns:[]clause.Column{{Name:"device_id"}},
			DoUpdates:clause.AssignmentColumns([]string{"policy_id","revision","updated_at"}),
		}).Create(item).Error
		if err==nil {invalidatePolicyActivity()}
		return err
	}
	result:=DB.Model(&model.DevicePolicyAssignment{}).Where("id = ?",item.Id).
		Updates(map[string]interface{}{"policy_id":item.PolicyId,"device_id":item.DeviceId,"revision":item.Revision})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return errors.New("policy assignment was not found")}
	invalidatePolicyActivity()
	return nil
}

func DeleteDevicePolicyAssignment(id uint) error {
	if id==0{return errors.New("invalid assignment ID")}
	err:=DB.Where("id = ?",id).Delete(&model.DevicePolicyAssignment{}).Error
	if err==nil {invalidatePolicyActivity()}
	return err
}

// EffectiveDevicePolicy preserves the original API for older callers.
func EffectiveDevicePolicy(deviceId string,userId uint,admin bool) (map[string]string,bool,error) {
	options,assigned,_,_,err:=EffectiveDevicePolicyDetails(deviceId,userId,admin)
	return options,assigned,err
}

func EffectiveDevicePolicyDetails(deviceId string,userId uint,admin bool) (map[string]string,bool,string,uint,error) {
	if len(deviceId)==0||len(deviceId)>253||userId==0{return nil,false,"",0,errors.New("invalid device or account")}
	var peer model.Peer
	if err:=DB.Where("id = ?",deviceId).First(&peer).Error;err!=nil{
		return nil,false,"",0,errors.New("device was not found")
	}
	if peer.UserId!=userId&&!admin{
		return nil,false,"",0,errors.New("device was not found")
	}
	resolved,err:=ResolveDevicePolicy(&peer)
	if err!=nil{return nil,false,"",0,err}
	if resolved==nil{return map[string]string{},false,"",0,nil}
	options:=map[string]string{
		"id-whitelist":resolved.Policy.IdWhitelist,
		"whitelist":resolved.Policy.IpWhitelist,
		"enable-webrtc":map[bool]string{true:"Y",false:"N"}[resolved.Policy.EnableWebRTC],
		"trusted-device-expiration-days":strconv.Itoa(resolved.Policy.TrustDurationDays),
	}
	for key,value := range extraClientPolicyOptions(&resolved.Policy) {
		options[key] = value
	}
	return options,true,resolved.Source,resolved.Policy.Id,nil
}

// HeartbeatDeviceStrategy returns native options for the highest-priority
// applicable policy, without changing the official client's response shape.
func HeartbeatDeviceStrategy(peer *model.Peer, modifiedAt int64) (map[string]interface{},error) {
	if peer==nil||peer.RowId==0{return map[string]interface{}{},nil}
	present,err:=devicePolicyAssignmentsPresent()
	if err!=nil{return nil,err}
	if !present {
		if modifiedAt==0{return map[string]interface{}{},nil}
		return map[string]interface{}{
			"modified_at":int64(0),
			"strategy":map[string]interface{}{"config_options":map[string]string{
				"id-whitelist":"","whitelist":"",
				"trusted-device-expiration-days":"",
				"access-mode":"","enable-keyboard":"","enable-clipboard":"",
				"enable-file-transfer":"","approve-mode":"",
			}},
		},nil
	}
	resolved,err:=resolveHeartbeatPolicy(peer)
	if err!=nil{return nil,err}
	if resolved==nil {
		if modifiedAt==0{return map[string]interface{}{},nil}
		return map[string]interface{}{
			"modified_at":int64(0),
			"strategy":map[string]interface{}{"config_options":map[string]string{
				"id-whitelist":"","whitelist":"",
				"trusted-device-expiration-days":"",
				"access-mode":"","enable-keyboard":"","enable-clipboard":"",
				"enable-file-transfer":"","approve-mode":"",
			}},
		},nil
	}
	revision:=resolved.Revision
	if revision==modifiedAt{return map[string]interface{}{},nil}
	options:=map[string]string{
		"id-whitelist":resolved.Policy.IdWhitelist,
		"whitelist":resolved.Policy.IpWhitelist,
		"trusted-device-expiration-days":strconv.Itoa(resolved.Policy.TrustDurationDays),
	}
	for key,value := range extraClientPolicyOptions(&resolved.Policy) {
		options[key] = value
	}
	return map[string]interface{}{
		"modified_at":revision,
		"strategy":map[string]interface{}{"config_options":options},
	},nil
}
