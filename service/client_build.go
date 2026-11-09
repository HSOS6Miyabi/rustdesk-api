package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

const currentClientVersion = "1.5.1"

type clientServerSettings struct {
	Key string `json:"key,omitempty"`
	Host string `json:"host,omitempty"`
	API string `json:"api,omitempty"`
	Relay string `json:"relay,omitempty"`
}

var clientHostCharacters = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9.\-:\[\]]*$`)

func validateClientHost(value string) bool {
	if value == "" || len(value) > 180 || !clientHostCharacters.MatchString(value) {
		return false
	}
	if strings.Contains(value,"..") || strings.ContainsAny(value,"\r\n\t ,/\\") {
		return false
	}
	if strings.HasPrefix(value,"[") {
		closing := strings.IndexByte(value,']')
		if closing < 0 || net.ParseIP(value[1:closing]) == nil { return false }
		if closing+1 < len(value) && value[closing+1] != ':' { return false }
	} else if strings.Count(value,":") > 1 && net.ParseIP(value) == nil {
		return false
	}
	return true
}

func validateClientAPI(value string) bool {
	if value == "" { return true }
	if len(value) > 200 || strings.ContainsAny(value,"\r\n\t ") { return false }
	parsed,err := url.Parse(value)
	if err!=nil || (parsed.Scheme!="https" && parsed.Scheme!="http") ||
		parsed.User!=nil || parsed.Hostname()=="" || parsed.RawQuery!="" ||
		parsed.Fragment!="" || (parsed.Path!="" && parsed.Path!="/") {
		return false
	}
	return true
}

func ValidateClientBuildProfile(profile *model.ClientBuildProfile) error {
	if profile == nil { return errors.New("client build profile is required") }
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Host = strings.TrimSpace(profile.Host)
	profile.PublicKey = strings.TrimSpace(profile.PublicKey)
	profile.APIServer = strings.TrimRight(strings.TrimSpace(profile.APIServer),"/")
	profile.RelayServer = strings.TrimSpace(profile.RelayServer)
	if profile.Name == "" || len([]rune(profile.Name)) > 100 {
		return errors.New("profile name must contain 1 to 100 characters")
	}
	if profile.Platform!="windows-x64" && profile.Platform!="windows-arm64" {
		return errors.New("supported platforms are Windows x64 and Windows ARM64")
	}
	if profile.Version=="" { profile.Version=currentClientVersion }
	if profile.Version!=currentClientVersion { return errors.New("unsupported official client version") }
	if !validateClientHost(profile.Host) || !validateClientAPI(profile.APIServer) ||
		(profile.RelayServer!="" && !validateClientHost(profile.RelayServer)) {
		return errors.New("invalid server, API or relay address")
	}
	key,err:=base64.StdEncoding.DecodeString(profile.PublicKey)
	if err!=nil || len(key)!=32 { return errors.New("server public key must contain 32 base64-encoded bytes") }
	_,err=BuildClientFilename(profile)
	return err
}

func BuildClientFilename(profile *model.ClientBuildProfile) (string,error) {
	if profile==nil{return "",errors.New("client build profile is required")}
	payload,err:=json.Marshal(clientServerSettings{
		Host:profile.Host,Key:profile.PublicKey,API:profile.APIServer,Relay:profile.RelayServer,
	})
	if err!=nil{return "",err}
	raw:=base64.RawURLEncoding.EncodeToString(payload)
	data:=[]byte(raw)
	for left,right:=0,len(data)-1;left<right;left,right=left+1,right-1 {
		data[left],data[right]=data[right],data[left]
	}
	filename:="rustdesk-licensed-"+string(data)+".exe"
	if len(filename)>240 {return "",errors.New("server settings exceed the Windows filename limit; shorten server addresses")}
	return filename,nil
}

func SaveClientBuildProfile(profile *model.ClientBuildProfile) error {
	if err:=ValidateClientBuildProfile(profile);err!=nil{return err}
	if profile.Id==0 {return DB.Create(profile).Error}
	result:=DB.Model(&model.ClientBuildProfile{}).Where("id = ?",profile.Id).
		Updates(map[string]interface{}{
			"name":profile.Name,"platform":profile.Platform,"version":profile.Version,
			"host":profile.Host,"public_key":profile.PublicKey,
			"api_server":profile.APIServer,"relay_server":profile.RelayServer,
		})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected==0{return gorm.ErrRecordNotFound}
	return nil
}

func ListClientBuildProfiles(page,size uint) ([]model.ClientBuildProfile,int64,error) {
	if page==0{page=1};if size==0{size=20};if size>100{size=100}
	var total int64
	query:=DB.Model(&model.ClientBuildProfile{})
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	var records []model.ClientBuildProfile
	err:=query.Order("id DESC").Scopes(Paginate(page,size)).Find(&records).Error
	return records,total,err
}

func ListClientBuildTasks(page,size uint) ([]model.ClientBuildTask,int64,error) {
	if page==0{page=1};if size==0{size=20};if size>100{size=100}
	var total int64
	query:=DB.Model(&model.ClientBuildTask{})
	if err:=query.Count(&total).Error;err!=nil{return nil,0,err}
	var records []model.ClientBuildTask
	err:=query.Order("id DESC").Scopes(Paginate(page,size)).Find(&records).Error
	return records,total,err
}

func QueueClientBuild(profileID,requestedBy uint) (*model.ClientBuildTask,error) {
	if profileID==0 || requestedBy==0 {return nil,errors.New("a client profile and administrator are required")}
	var profile model.ClientBuildProfile
	if err:=DB.Where("id = ?",profileID).First(&profile).Error;err!=nil{return nil,err}
	if err:=ValidateClientBuildProfile(&profile);err!=nil{return nil,err}
	filename,err:=BuildClientFilename(&profile)
	if err!=nil{return nil,err}
	configuration,err:=json.Marshal(clientServerSettings{
		Host:profile.Host,Key:profile.PublicKey,API:profile.APIServer,Relay:profile.RelayServer,
	})
	if err!=nil{return nil,err}
	task:=&model.ClientBuildTask{
		ProfileID:profileID,RequestedBy:requestedBy,
		Version:profile.Version,Platform:profile.Platform,ConfigSnapshot:string(configuration),
		Status:"pending",Filename:filename,
	}
	if err:=DB.Create(task).Error;err!=nil{return nil,err}
	return task,nil
}

func RetryClientBuild(id uint)error{
	if id==0{return errors.New("client build task ID is required")}
	result:=DB.Model(&model.ClientBuildTask{}).Where("id = ? AND status = ?",id,"failed").
		Updates(map[string]interface{}{
			"status":"pending","error":"","started_at":int64(0),"finished_at":int64(0),
			"bytes":int64(0),"sha256":"",
		})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected!=1{return errors.New("task is not eligible for retry")}
	return nil
}

func clientBuildRoot()string {
	if dir:=strings.TrimSpace(os.Getenv("RUSTDESK_CLIENT_BUILD_DIR"));dir!=""{
		return filepath.Clean(dir)
	}
	return filepath.Join("data","client-builds")
}

func ClientBuildFile(taskID uint)(string,string,error){
	if taskID==0{return "","",errors.New("invalid build ID")}
	var task model.ClientBuildTask
	if err:=DB.First(&task,taskID).Error;err!=nil{return "","",err}
	if task.Status!="ready"{return "","",errors.New("client package is not ready")}
	if task.Filename==""||strings.ContainsAny(task.Filename,"/\\") {
		return "","",errors.New("invalid client package name")
	}
	path:=filepath.Join(clientBuildRoot(),fmt.Sprintf("%d",task.Id),task.Filename)
	info,err:=os.Stat(path)
	if err!=nil{return "","",err}
	if !info.Mode().IsRegular()||info.Size()!=task.Bytes{
		return "","",errors.New("client package is unavailable")
	}
	return path,task.Filename,nil
}

// DropClientBuildProfile preserves historical task records and installers.
func DropClientBuildProfile(id uint)error {
	if id==0{return errors.New("invalid profile ID")}
	result:=DB.Where("id = ?",id).Delete(&model.ClientBuildProfile{})
	if result.Error!=nil{return result.Error}
	if result.RowsAffected!=1{return gorm.ErrRecordNotFound}
	return nil
}

