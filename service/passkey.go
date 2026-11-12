package service

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "net"
    "net/url"
    "strings"
    "time"

    "github.com/go-webauthn/webauthn/protocol"
    "github.com/go-webauthn/webauthn/webauthn"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/gorm"
)

const passkeyChallengeLifetime = 5 * time.Minute
const maxPasskeysPerAccount = 25

type PasskeyAccount struct {
    Owner *model.User
    Handle []byte
    Credentials []webauthn.Credential
}
func (p *PasskeyAccount) WebAuthnID() []byte {return p.Handle}
func (p *PasskeyAccount) WebAuthnName() string {return p.Owner.Username}
func (p *PasskeyAccount) WebAuthnDisplayName() string {
    if p.Owner.Nickname!="" {return p.Owner.Nickname}
    return p.Owner.Username
}
func (p *PasskeyAccount) WebAuthnCredentials() []webauthn.Credential {return p.Credentials}

func securePasskeyToken()(string,error){
    bytes:=make([]byte,32)
    if _,err:=rand.Read(bytes);err!=nil{return "",err}
    return base64.RawURLEncoding.EncodeToString(bytes),nil
}
func passkeyDigest(token string)string{
    digest:=sha256.Sum256([]byte(token))
    return hex.EncodeToString(digest[:])
}

// PasskeyOrigin rejects cross-site requests, malformed Host values and
// insecure origins. Reverse proxies must preserve the public Host header.
func PasskeyOrigin(origin,host string)(string,string,error){
    parsed,err:=url.Parse(origin)
    if err!=nil||parsed==nil||parsed.User!=nil||parsed.Path!=""||parsed.RawQuery!=""||parsed.Fragment!="" {
        return "","",errors.New("Passkey 需要从控制台的完整网页地址发起")
    }
    if parsed.Host==""||parsed.Hostname()==""||strings.Contains(host,"/")||
        !strings.EqualFold(parsed.Host,host)||strings.ContainsAny(parsed.Hostname()," \r\n\t") {
        return "","",errors.New("Passkey 页面地址与服务器地址不匹配")
    }
    hostname:=strings.ToLower(parsed.Hostname())
    if ip:=net.ParseIP(hostname);ip!=nil&&!ip.IsLoopback(){
        return "","",errors.New("Passkey 需要使用已配置域名")
    }
    local:=hostname=="localhost"||hostname=="127.0.0.1"||hostname=="::1"
    if parsed.Scheme!="https" && !(parsed.Scheme=="http"&&local){
        return "","",errors.New("Passkey 需要 HTTPS 或本机 localhost")
    }
    if len(hostname)>253||(hostname!="::1"&&strings.ContainsAny(hostname,":/\\")){
        return "","",errors.New("Passkey 域名无效")
    }
    return parsed.Scheme+"://"+parsed.Host,hostname,nil
}

func newPasskeyVerifier(origin,rpID string)(*webauthn.WebAuthn,error){
    return webauthn.New(&webauthn.Config{
        RPDisplayName:"RustDesk",
        RPID:rpID,
        RPOrigins:[]string{origin},
        AuthenticatorSelection:protocol.AuthenticatorSelection{
            ResidentKey:protocol.ResidentKeyRequirementRequired,
            UserVerification:protocol.VerificationRequired,
        },
    })
}

func passkeyHandleForUser(userID uint,create bool)([]byte,error){
    if userID==0{return nil,errors.New("账户不存在")}
    var row model.UserPasskeyHandle
    err:=DB.Where("user_id = ?",userID).First(&row).Error
    if err==nil{return base64.RawURLEncoding.DecodeString(row.Handle)}
    if !errors.Is(err,gorm.ErrRecordNotFound)||!create{return nil,err}
    token,err:=securePasskeyToken()
    if err!=nil{return nil,err}
    row=model.UserPasskeyHandle{UserId:userID,Handle:token}
    if err=DB.Create(&row).Error;err!=nil{
        var existing model.UserPasskeyHandle
        if err2:=DB.Where("user_id = ?",userID).First(&existing).Error;err2==nil{
            return base64.RawURLEncoding.DecodeString(existing.Handle)
        }
        return nil,err
    }
    return base64.RawURLEncoding.DecodeString(token)
}

func loadPasskeyAccount(userID uint,create bool)(*PasskeyAccount,error){
    user:=AllService.UserService.InfoById(userID)
    if user==nil||user.Id==0||!AllService.UserService.CheckUserEnable(user){
        return nil,errors.New("账户不可用")
    }
    handle,err:=passkeyHandleForUser(userID,create)
    if err!=nil{return nil,err}
    rows:=[]model.UserPasskey{}
    if err=DB.Where("user_id = ?",userID).Order("id ASC").Find(&rows).Error;err!=nil{return nil,err}
    credentials:=make([]webauthn.Credential,0,len(rows))
    for _,row:=range rows{
        credential:=webauthn.Credential{}
        if err=json.Unmarshal([]byte(row.CredentialJSON),&credential);err!=nil{
            return nil,fmt.Errorf("无法读取已注册的 Passkey 凭证: %w",err)
        }
        credentials=append(credentials,credential)
    }
    return &PasskeyAccount{Owner:user,Handle:handle,Credentials:credentials},nil
}

func passkeyChallengeBegin(userID uint,purpose,origin,ip string,session *webauthn.SessionData)(string,error){
    if purpose!="register"&&purpose!="login"{return "",errors.New("无效的 Passkey 操作")}
    if session==nil||session.Challenge==""{return "",errors.New("无法创建 Passkey 验证请求")}
    now:=time.Now().Unix()
    ipHash:=passkeyDigest(ip)
    var count int64
    if err:=DB.Model(&model.UserPasskeyChallenge{}).
        Where("ip_hash = ? AND expires_at > ? AND used_at = 0",ipHash,now).
        Count(&count).Error;err!=nil{return "",err}
    if count>=30{return "",errors.New("Passkey 请求过于频繁，请稍后重试")}
    token,err:=securePasskeyToken()
    if err!=nil{return "",err}
    data,err:=json.Marshal(session)
    if err!=nil{return "",err}
    row:=model.UserPasskeyChallenge{
        UserId:userID,Purpose:purpose,TokenHash:passkeyDigest(token),
        SessionJSON:string(data),Origin:origin,IPHash:ipHash,
        ExpiresAt:now+int64(passkeyChallengeLifetime.Seconds()),
    }
    if err=DB.Create(&row).Error;err!=nil{return "",err}
    // A short deletion of old rows bounds database storage for abandoned attempts.
    if row.Id%30==0{
        DB.Where("expires_at < ?",now-86400).Delete(&model.UserPasskeyChallenge{})
    }
    return token,nil
}

func consumePasskeyChallenge(token,purpose,origin string,userID uint)(webauthn.SessionData,error){
    empty:=webauthn.SessionData{}
    if len(token)!=43 {return empty,errors.New("Passkey 请求无效或已过期")}
    now:=time.Now().Unix()
    var row model.UserPasskeyChallenge
    if err:=DB.Where("token_hash = ? AND purpose = ?",passkeyDigest(token),purpose).
        First(&row).Error;err!=nil{return empty,errors.New("Passkey 请求无效或已过期")}
    if row.Origin!=origin||row.ExpiresAt<=now||row.UsedAt!=0||
       purpose=="register"&&row.UserId!=userID{
        return empty,errors.New("Passkey 验证请求无效或已过期")
    }
    result:=DB.Model(&model.UserPasskeyChallenge{}).
        Where("id = ? AND used_at = 0 AND expires_at > ?",row.Id,now).
        Update("used_at",now)
    if result.Error!=nil{return empty,result.Error}
    if result.RowsAffected!=1{return empty,errors.New("Passkey 请求已经使用")}
    if err:=json.Unmarshal([]byte(row.SessionJSON),&empty);err!=nil{
        return webauthn.SessionData{},errors.New("Passkey 验证请求格式无效")
    }
    return empty,nil
}

func BeginPasskeyRegistration(userID uint,origin,host,ip string)(interface{},string,error){
    accepted,rpID,err:=PasskeyOrigin(origin,host)
    if err!=nil{return nil,"",err}
    account,err:=loadPasskeyAccount(userID,true)
    if err!=nil{return nil,"",err}
    if len(account.Credentials)>=maxPasskeysPerAccount{
        return nil,"",errors.New("最多可以注册 25 个 Passkey")
    }
    verifier,err:=newPasskeyVerifier(accepted,rpID)
    if err!=nil{return nil,"",err}
    options,session,err:=verifier.BeginRegistration(account,
        webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
        webauthn.WithExclusions(webauthn.Credentials(account.Credentials).CredentialDescriptors()))
    if err!=nil{return nil,"",err}
    token,err:=passkeyChallengeBegin(userID,"register",accepted,ip,session)
    return options,token,err
}

func FinishPasskeyRegistration(userID uint,origin,host,token,name string,requestBody []byte)(*model.UserPasskey,error){
    accepted,rpID,err:=PasskeyOrigin(origin,host)
    if err!=nil{return nil,err}
    session,err:=consumePasskeyChallenge(token,"register",accepted,userID)
    if err!=nil{return nil,err}
    if session.RelyingPartyID!=rpID{return nil,errors.New("Passkey 服务器地址已改变")}
    account,err:=loadPasskeyAccount(userID,false)
    if err!=nil{return nil,err}
    verifier,err:=newPasskeyVerifier(accepted,rpID)
    if err!=nil{return nil,err}
    cred,err:=VerifyPasskeyRegistrationResponse(verifier,account,session,requestBody)
    if err!=nil{return nil,fmt.Errorf("Passkey 注册验证失败: %w",err)}
    raw,err:=json.Marshal(cred)
    if err!=nil{return nil,err}
    label:=strings.TrimSpace(name)
    if label==""{label="新的 Passkey"}
    if len([]rune(label))>80{return nil,errors.New("Passkey 名称不能超过 80 个字符")}
    record:=&model.UserPasskey{
        UserId:userID,CredentialID:base64.RawURLEncoding.EncodeToString(cred.ID),
        CredentialJSON:string(raw),RPID:rpID,Name:label,
    }
    if err=DB.Create(record).Error;err!=nil{return nil,errors.New("该 Passkey 已被注册")}
    return record,nil
}

func BeginPasskeyLogin(origin,host,ip string)(interface{},string,error){
    accepted,rpID,err:=PasskeyOrigin(origin,host)
    if err!=nil{return nil,"",err}
    verifier,err:=newPasskeyVerifier(accepted,rpID)
    if err!=nil{return nil,"",err}
    options,session,err:=verifier.BeginDiscoverableLogin(
        webauthn.WithUserVerification(protocol.VerificationRequired))
    if err!=nil{return nil,"",err}
    token,err:=passkeyChallengeBegin(0,"login",accepted,ip,session)
    return options,token,err
}

func FinishPasskeyLogin(origin,host,token,ip string,body []byte)(*model.User,error){
    accepted,rpID,err:=PasskeyOrigin(origin,host)
    if err!=nil{return nil,err}
    session,err:=consumePasskeyChallenge(token,"login",accepted,0)
    if err!=nil{return nil,err}
    if session.RelyingPartyID!=rpID{return nil,errors.New("Passkey 登录地址已改变")}
    verifier,err:=newPasskeyVerifier(accepted,rpID)
    if err!=nil{return nil,err}
    lookup:=func(rawID,userHandle []byte)(webauthn.User,error){
        if len(userHandle)==0||len(rawID)==0{return nil,errors.New("Passkey 凭证无效")}
        handle:=base64.RawURLEncoding.EncodeToString(userHandle)
        var bound model.UserPasskeyHandle
        if err:=DB.Where("handle = ?",handle).First(&bound).Error;err!=nil{return nil,err}
        credentialID:=base64.RawURLEncoding.EncodeToString(rawID)
        var record model.UserPasskey
        if err:=DB.Where("credential_id = ? AND user_id = ? AND rp_id = ?",
            credentialID,bound.UserId,rpID).First(&record).Error;err!=nil{return nil,err}
        return loadPasskeyAccount(bound.UserId,false)
    }
    verified,credential,err:=VerifyPasskeyLoginResponse(verifier,lookup,session,body)
    if err!=nil{return nil,fmt.Errorf("Passkey 登录验证失败: %w",err)}
    account,ok:=verified.(*PasskeyAccount)
    if !ok||account.Owner==nil{return nil,errors.New("Passkey 用户信息无效")}
    if !AllService.UserService.CheckUserEnable(account.Owner){return nil,errors.New("账户已停用")}
    credentialID:=base64.RawURLEncoding.EncodeToString(credential.ID)
    data,err:=json.Marshal(credential)
    if err!=nil{return nil,err}
    result:=DB.Model(&model.UserPasskey{}).
        Where("credential_id = ? AND user_id = ? AND rp_id = ?",
            credentialID,account.Owner.Id,rpID).
        Updates(map[string]interface{}{
            "credential_json":string(data),"last_used_at":time.Now().Unix(),
        })
    if result.Error!=nil{return nil,result.Error}
    if result.RowsAffected!=1{return nil,errors.New("Passkey 凭证已被撤销")}
    return account.Owner,nil
}

type PasskeySummary struct{
    ID uint `json:"id"`
    Name string `json:"name"`
    CreatedAt interface{} `json:"created_at"`
    LastUsedAt int64 `json:"last_used_at"`
}
func ListAccountPasskeys(userID uint)([]PasskeySummary,error){
    var rows []model.UserPasskey
    if err:=DB.Where("user_id = ?",userID).Order("id DESC").Find(&rows).Error;err!=nil{return nil,err}
    out:=make([]PasskeySummary,0,len(rows))
    for _,r:=range rows{
        out=append(out,PasskeySummary{
            ID:r.Id,Name:r.Name,CreatedAt:r.CreatedAt,LastUsedAt:r.LastUsedAt,
        })
    }
    return out,nil
}

func RevokeAccountPasskey(userID,id uint)error{
    if userID==0||id==0{return errors.New("请选择要移除的 Passkey")}
    r:=DB.Where("user_id = ? AND id = ?",userID,id).Delete(&model.UserPasskey{})
    if r.Error!=nil{return r.Error}
    if r.RowsAffected!=1{return gorm.ErrRecordNotFound}
    return nil
}
