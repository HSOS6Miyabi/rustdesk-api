package service

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAPITokenIssuancePermissionsAndRevocation(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.APIToken{});err!=nil{t.Fatal(err)}
	users:=seedGroups(t)
	u:=users["member"]
	record,raw,err:=IssueAPIToken(u.Id,"Automated device management",[]string{"devices.deploy","devices.assign"},0)
	if err!=nil{t.Fatal(err)}
	if !strings.HasPrefix(raw,"rdapi_")||len(raw)!=49||record.TokenHash==""{
		t.Fatalf("invalid API token format or record: %+v",record)
	}
	if strings.Contains(record.TokenHash,raw) || record.TokenHash==raw{
		t.Fatal("plaintext API token was persisted")
	}
	serialized,err:=json.Marshal(record)
	if err!=nil{t.Fatal(err)}
	if strings.Contains(string(serialized),record.TokenHash)||
		strings.Contains(string(serialized),raw){
		t.Fatal("API token leaked through JSON serialization")
	}
	who,err:=AuthenticateAPIToken(raw,"devices.assign")
	if err!=nil||who.Id!=u.Id{t.Fatalf("authorized token rejected: %v",err)}
	if _,err:=AuthenticateAPIToken(raw,"devices.read");err==nil{
		t.Fatal("token was accepted for an unspecified permission")
	}
	_,restricted,err:=IssueAPIToken(u.Id,"Deployment only",[]string{"devices.deploy"},1)
	if err!=nil{t.Fatal(err)}
	if _,err:=AuthenticateAPIToken(restricted,"devices.assign");err==nil{
		t.Fatal("deployment-only token can assign devices")
	}
	if _,err:=AuthenticateAPIToken(restricted,"devices.deploy");err!=nil{t.Fatal(err)}
	if err:=RevokeAPIToken(record.Id);err!=nil{t.Fatal(err)}
	if _,err:=AuthenticateAPIToken(raw,"devices.deploy");err==nil{
		t.Fatal("revoked token is still accepted")
	}
	var restrictedRow model.APIToken
	if err:=DB.Where("token_hash = ?",apiTokenHash(restricted)).First(&restrictedRow).Error;err!=nil{t.Fatal(err)}
	if err:=DB.Model(&restrictedRow).Update("expires_at",time.Now().Unix()-1).Error;err!=nil{t.Fatal(err)}
	if _,err:=AuthenticateAPIToken(restricted,"devices.deploy");err==nil{
		t.Fatal("expired token is still accepted")
	}
	if _,_,err:=IssueAPIToken(u.Id,"Bad scope",[]string{"users.delete"},0);err==nil{
		t.Fatal("unsupported permission was accepted")
	}
	if _,_,err:=IssueAPIToken(u.Id,"Bad expiry",[]string{"devices.assign"},3651);err==nil{
		t.Fatal("unreasonably long expiry was accepted")
	}
}

func TestAPITokenDisabledAccountCannotUseToken(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.APIToken{});err!=nil{t.Fatal(err)}
	user:=seedGroups(t)["member"]
	_,raw,err:=IssueAPIToken(user.Id,"Headless deploy",[]string{"devices.deploy"},0)
	if err!=nil{t.Fatal(err)}
	if err:=DB.Model(user).Update("status",model.COMMON_STATUS_DISABLED).Error;err!=nil{t.Fatal(err)}
	if _,err:=AuthenticateAPIToken(raw,"devices.deploy");err==nil{
		t.Fatal("disabled account's API token was accepted")
	}
}
