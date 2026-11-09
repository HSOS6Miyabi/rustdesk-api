package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const (
	accessSourceAll="all"
	accessSourceUser="user"
	accessSourceGroup="group"
	accessTargetAll="all"
	accessTargetDevice="device"
	accessTargetUser="user"
	accessTargetDeviceGroup="device_group"
	accessAllow="allow"
	accessDeny="deny"
)

type ConnectionAccessResult struct {
	Allow bool `json:"allow"`
	Reason string `json:"reason"`
	RuleID uint `json:"rule_id,omitempty"`
}

func NormalizeAccessRule(rule *model.AccessRule) error {
	if rule==nil{return errors.New("connection rule is required")}
	rule.Name=strings.TrimSpace(rule.Name)
	rule.TargetID=strings.TrimSpace(rule.TargetID)
	if len([]rune(rule.Name))==0||len([]rune(rule.Name))>100{
		return errors.New("rule name must contain 1 to 100 characters")
	}
	if rule.Effect!=accessAllow&&rule.Effect!=accessDeny{
		return errors.New("effect must be allow or deny")
	}
	if rule.SourceType!=accessSourceAll&&rule.SourceType!=accessSourceUser&&rule.SourceType!=accessSourceGroup{
		return errors.New("source must be all, user or group")
	}
	if rule.TargetType!=accessTargetAll&&rule.TargetType!=accessTargetDevice&&
		rule.TargetType!=accessTargetUser&&rule.TargetType!=accessTargetDeviceGroup{
		return errors.New("target must be all, device, user or device_group")
	}
	if rule.SourceType==accessSourceAll{
		rule.SourceID=0
	}else {
		if rule.SourceID==0{return errors.New("source user or group ID is required")}
		var count int64
		var err error
		if rule.SourceType==accessSourceUser{
			err=DB.Model(&model.User{}).Where("id = ?",rule.SourceID).Count(&count).Error
		}else{
			err=DB.Model(&model.Group{}).Where("id = ?",rule.SourceID).Count(&count).Error
		}
		if err!=nil{return err}
		if count==0{return errors.New("source account or group does not exist")}
	}
	if rule.TargetType==accessTargetAll{
		rule.TargetID=""
		return nil
	}
	if rule.TargetID==""||len(rule.TargetID)>253{
		return errors.New("target device, account or device group is required")
	}
	var count int64
	switch rule.TargetType{
	case accessTargetDevice:
		if err:=DB.Model(&model.Peer{}).Where("id = ?",rule.TargetID).Count(&count).Error;err!=nil{return err}
	case accessTargetUser,accessTargetDeviceGroup:
		n,err:=strconv.ParseUint(rule.TargetID,10,32)
		if err!=nil||n==0{return errors.New("target account or group ID is invalid")}
		rule.TargetID=strconv.FormatUint(n,10)
		if rule.TargetType==accessTargetUser{
			err=DB.Model(&model.User{}).Where("id = ?",n).Count(&count).Error
		}else{
			err=DB.Model(&model.DeviceGroup{}).Where("id = ?",n).Count(&count).Error
		}
		if err!=nil{return err}
	}
	if count==0{return errors.New("target device, account or device group does not exist")}
	return nil
}

func SaveAccessRule(rule *model.AccessRule) error {
	if err:=NormalizeAccessRule(rule);err!=nil{return err}
	if rule.Id==0{return DB.Create(rule).Error}
	result:=DB.Model(&model.AccessRule{}).Where("id = ?",rule.Id).
		Updates(map[string]interface{}{
			"name":rule.Name,"source_type":rule.SourceType,"source_id":rule.SourceID,
			"target_type":rule.TargetType,"target_id":rule.TargetID,
			"effect":rule.Effect,"enabled":rule.Enabled,
		})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteAccessRule(id uint)error{
	if id==0{return errors.New("rule ID is required")}
	res:=DB.Where("id = ?",id).Delete(&model.AccessRule{})
	if res.Error!=nil{return res.Error}
	if res.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func ListAccessRules(page,size uint)([]model.AccessRule,int64,error){
	var count int64
	var rows []model.AccessRule
	query:=DB.Model(&model.AccessRule{})
	if err:=query.Count(&count).Error;err!=nil{return nil,0,err}
	err:=query.Order("id DESC").Scopes(Paginate(page,size)).Find(&rows).Error
	return rows,count,err
}

func accessRuleMatches(rule *model.AccessRule,source *model.User,groups map[uint]bool,peer *model.Peer) bool {
	sourceMatch:=false
	switch rule.SourceType{
	case accessSourceAll:sourceMatch=true
	case accessSourceUser:sourceMatch=rule.SourceID==source.Id
	case accessSourceGroup:sourceMatch=groups[rule.SourceID]
	}
	if !sourceMatch{return false}
	switch rule.TargetType{
	case accessTargetAll:return true
	case accessTargetDevice:return rule.TargetID==peer.Id
	case accessTargetUser:
		return peer.UserId>0&&rule.TargetID==strconv.FormatUint(uint64(peer.UserId),10)
	case accessTargetDeviceGroup:
		return peer.GroupId>0&&rule.TargetID==strconv.FormatUint(uint64(peer.GroupId),10)
	}
	return false
}

// AuthorizeConnection checks a normal RustDesk account session and the
// registered destination. In strict mode, an unmatched request is denied.
// The device owner is always allowed to reach their own registered device.
// An explicit deny takes precedence over any matching allow.
func AuthorizeConnection(rawToken,deviceID string)(ConnectionAccessResult,error){
	denied:=ConnectionAccessResult{Allow:false,Reason:"not_authorized"}
	if rawToken==""||len(rawToken)>4096||deviceID==""||len(deviceID)>253{
		return denied,nil
	}
	user,token:=(&UserService{}).InfoByAccessToken(rawToken)
	if user==nil||user.Id==0||token==nil||token.Id==0||!(&UserService{}).CheckUserEnable(user){
		denied.Reason="login_required"
		return denied,nil
	}
	var peer model.Peer
	err:=DB.Where("id = ?",deviceID).First(&peer).Error
	if errors.Is(err,gorm.ErrRecordNotFound){denied.Reason="device_not_registered";return denied,nil}
	if err!=nil{return denied,err}
	if peer.UserId==user.Id{
		return ConnectionAccessResult{Allow:true,Reason:"device_owner"},nil
	}
	var rules []model.AccessRule
	if err:=DB.Where("enabled = ?",true).Order("id ASC").Find(&rules).Error;err!=nil{return denied,err}
	if len(rules)==0{return denied,nil}
	groups,err:=(&GroupService{}).GroupIdsForUser(user)
	if err!=nil{return denied,err}
	groupSet:=make(map[uint]bool,len(groups))
	for _,id:=range groups{groupSet[id]=true}
	allowed:=ConnectionAccessResult{Allow:false,Reason:"no_matching_rule"}
	for _,rule:=range rules{
		if !accessRuleMatches(&rule,user,groupSet,&peer){continue}
		if rule.Effect==accessDeny{
			return ConnectionAccessResult{Allow:false,Reason:"explicit_deny",RuleID:rule.Id},nil
		}
		if rule.Effect==accessAllow{
			allowed=ConnectionAccessResult{Allow:true,Reason:"matching_allow",RuleID:rule.Id}
		}
	}
	if !allowed.Allow{return ConnectionAccessResult{Allow:false,Reason:fmt.Sprintf("no_matching_rule_for_%s",deviceID)},nil}
	return allowed,nil
}
