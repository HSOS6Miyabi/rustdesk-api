package service

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type InternalRelayNode struct {
	Address string `json:"address"`
	Subnets []string `json:"subnets"`
	Priority int `json:"priority"`
}

var relayHostName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.-]*$`)

func validateRelayAddress(address string) error {
	if address==""||len(address)>253||strings.ContainsAny(address," /@\t\r\n"){
		return errors.New("relay address must be a hostname or IP address")
	}
	host:=address
	if strings.HasPrefix(address,"[")||strings.Contains(address,":"){
		parsedHost,parsedPort,err:=net.SplitHostPort(address)
		if err!=nil{return errors.New("relay address with IPv6 or a port must use host:port")}
		host=parsedHost
		port,err:=strconv.Atoi(parsedPort)
		if err!=nil||port<1||port>65535{return errors.New("relay port must be between 1 and 65535")}
	}
	if host==""||(net.ParseIP(host)==nil&&!relayHostName.MatchString(host)){
		return errors.New("relay hostname or IP address is invalid")
	}
	return nil
}

func normalizeRelaySubnets(value string)(string,error){
	parts:=strings.FieldsFunc(value,func(r rune)bool{return r==','||r==';'||r=='\n'||r=='\r'})
	seen:=make(map[string]bool)
	out:=make([]string,0,len(parts))
	if len(parts)>64{return "",errors.New("at most 64 network ranges are allowed")}
	for _,part:=range parts{
		raw:=strings.TrimSpace(part)
		if raw==""{continue}
		_,network,err:=net.ParseCIDR(raw)
		if err!=nil{return "",fmt.Errorf("invalid IP network: %s",raw)}
		cidr:=network.String()
		if !seen[cidr]{
			seen[cidr]=true
			out=append(out,cidr)
		}
	}
	sort.Strings(out)
	return strings.Join(out,","),nil
}

func SaveRelayNode(node *model.RelayNode)error{
	if node==nil{return errors.New("relay server details are required")}
	node.Address=strings.TrimSpace(node.Address)
	node.Name=strings.TrimSpace(node.Name)
	if node.Name==""{node.Name=node.Address}
	if len([]rune(node.Name))>100{return errors.New("relay name exceeds 100 characters")}
	if err:=validateRelayAddress(node.Address);err!=nil{return err}
	if node.Priority<0||node.Priority>10000{
		return errors.New("relay priority must be between 0 and 10000")
	}
	subnets,err:=normalizeRelaySubnets(node.Subnets)
	if err!=nil{return err}
	node.Subnets=subnets
	if node.Id==0{return DB.Create(node).Error}
	updates:=map[string]interface{}{
		"name":node.Name,"address":node.Address,"subnets":node.Subnets,
		"priority":node.Priority,"enabled":node.Enabled,
	}
	result:=DB.Model(&model.RelayNode{}).Where("id = ?",node.Id).Updates(updates)
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func DeleteRelayNode(id uint)error {
	if id==0{return errors.New("relay node ID is required")}
	result:=DB.Where("id = ?",id).Delete(&model.RelayNode{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func ListRelayNodes(page,pageSize uint)([]model.RelayNode,int64,error){
	var total int64
	var nodes []model.RelayNode
	query:=DB.Model(&model.RelayNode{})
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	err:=query.Order("priority ASC, id ASC").Scopes(Paginate(page,pageSize)).
		Find(&nodes).Error
	return nodes,total,err
}

func EnabledRelayNodes() ([]InternalRelayNode,error) {
	var nodes []model.RelayNode
	if err:=DB.Where("enabled = ?",true).
		Order("priority ASC, id ASC").Limit(64).Find(&nodes).Error;err!=nil{
		return nil,err
	}
	out:=make([]InternalRelayNode,0,len(nodes))
	for _,node:=range nodes{
		cidrs:=[]string{}
		if node.Subnets!=""{cidrs=strings.Split(node.Subnets,",")}
		out=append(out,InternalRelayNode{
			Address:node.Address,Subnets:cidrs,Priority:node.Priority,
		})
	}
	return out,nil
}
