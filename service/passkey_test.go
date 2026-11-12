package service

import (
    "encoding/json"
    "errors"
    "testing"
    "time"

    "github.com/go-webauthn/webauthn/webauthn"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/gorm"
)

func TestPasskeyOriginValidation(t *testing.T){
    tests:=[]struct{origin,host string;valid bool}{
        {"https://desk.example.test","desk.example.test",true},
        {"https://desk.example.test:8443","desk.example.test:8443",true},
        {"http://localhost:21114","localhost:21114",true},
        {"http://127.0.0.1:21114","127.0.0.1:21114",true},
        {"http://desk.example.test","desk.example.test",false},
        {"https://desk.example.test","another.example.test",false},
        {"https://desk.example.test/unsafe","desk.example.test",false},
        {"https://desk.example.test?x=1","desk.example.test",false},
        {"https://desk.example.test@evil.test","evil.test",false},
        {"https://desk.example.test","desk.example.test/evil",false},
        {"","desk.example.test",false},
    }
    for _,item:=range tests {
        _,_,err:=PasskeyOrigin(item.origin,item.host)
        if (err==nil)!=item.valid {t.Errorf("origin %q and Host %q: %v",item.origin,item.host,err)}
    }
}

func TestPasskeyChallengeExpiresAndRejectsReplay(t *testing.T){
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.UserPasskey{},&model.UserPasskeyHandle{},&model.UserPasskeyChallenge{});err!=nil{t.Fatal(err)}
    token,err:=passkeyChallengeBegin(12,"register","https://desk.example.test","192.0.2.1",
        &webauthn.SessionData{Challenge:"secure-challenge",RelyingPartyID:"desk.example.test"})
    if err!=nil{t.Fatal(err)}
    if _,err:=consumePasskeyChallenge(token,"register","https://other.test",12);err==nil{
        t.Error("accepted a different origin")
    }
    if _,err:=consumePasskeyChallenge(token,"register","https://desk.example.test",13);err==nil{
        t.Error("accepted a different user")
    }
    session,err:=consumePasskeyChallenge(token,"register","https://desk.example.test",12)
    if err!=nil||session.Challenge!="secure-challenge"{t.Fatalf("failed valid ceremony: %v %v",session.Challenge,err)}
    if _,err:=consumePasskeyChallenge(token,"register","https://desk.example.test",12);err==nil{
        t.Fatal("the challenge can be used twice")
    }
    expired,err:=passkeyChallengeBegin(0,"login","https://desk.example.test","192.0.2.1",
        &webauthn.SessionData{Challenge:"expired",RelyingPartyID:"desk.example.test"})
    if err!=nil{t.Fatal(err)}
    if err:=DB.Model(&model.UserPasskeyChallenge{}).Where("token_hash = ?",passkeyDigest(expired)).
        Update("expires_at",time.Now().Add(-time.Minute).Unix()).Error;err!=nil{t.Fatal(err)}
    if _,err:=consumePasskeyChallenge(expired,"login","https://desk.example.test",0);err==nil{
        t.Fatal("expired challenge was accepted")
    }
}

func TestPasskeyAccountIsolationAndCredentialRevocation(t *testing.T){
    cleanup:=setupGroupDB(t)
    defer cleanup()
    AllService.UserService=&UserService{}
    if err:=DB.AutoMigrate(&model.UserPasskey{},&model.UserPasskeyHandle{},&model.UserPasskeyChallenge{});err!=nil{t.Fatal(err)}
    users:=seedGroups(t)
    first,err:=loadPasskeyAccount(users["member"].Id,true)
    if err!=nil{t.Fatal(err)}
    second,err:=loadPasskeyAccount(users["other"].Id,true)
    if err!=nil{t.Fatal(err)}
    if string(first.Handle)==string(second.Handle){t.Fatal("two accounts share a WebAuthn user handle")}
    again,err:=loadPasskeyAccount(users["member"].Id,false)
    if err!=nil||string(first.Handle)!=string(again.Handle){t.Fatalf("unstable passkey user handle: %v",err)}
    cred,_:=json.Marshal(&webauthn.Credential{ID:[]byte{1,2,3,4}})
    row:=model.UserPasskey{UserId:users["member"].Id,CredentialID:"AQIDBA",
        CredentialDigest:passkeyDigest("AQIDBA"),CredentialJSON:string(cred),RPID:"desk.example.test",Name:"工作电脑"}
    if err:=DB.Create(&row).Error;err!=nil{t.Fatal(err)}
    list,err:=ListAccountPasskeys(users["member"].Id)
    if err!=nil||len(list)!=1||list[0].Name!="工作电脑"{t.Fatalf("invalid passkey list: %+v %v",list,err)}
    outside,err:=ListAccountPasskeys(users["other"].Id)
    if err!=nil||len(outside)!=0{t.Fatal("another account can see registered passkeys")}
    if err:=RevokeAccountPasskey(users["other"].Id,row.Id);err==nil{t.Fatal("another user revoked a passkey")}
    if err:=RevokeAccountPasskey(users["member"].Id,row.Id);err!=nil{t.Fatal(err)}
    if err:=RevokeAccountPasskey(users["member"].Id,row.Id);!errors.Is(err,gorm.ErrRecordNotFound){
        t.Fatal("already removed passkey accepted")
    }
}
