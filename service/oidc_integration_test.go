package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/sirupsen/logrus"
)

func signedOIDCTestToken(t *testing.T, key *rsa.PrivateKey, issuer, nonce string) string {
	t.Helper()
	header, err := json.Marshal(map[string]interface{}{
		"alg":"RS256","kid":"test-signing-key","typ":"JWT",
	})
	if err != nil { t.Fatal(err) }
	claims, err := json.Marshal(map[string]interface{}{
		"iss":issuer,
		"aud":"rustdesk-api-oidc-test",
		"sub":"oidc-test-subject",
		"iat":time.Now().Add(-time.Minute).Unix(),
		"exp":time.Now().Add(time.Hour).Unix(),
		"nonce":nonce,
	})
	if err != nil { t.Fatal(err) }
	value := base64.RawURLEncoding.EncodeToString(header)+"."+
		base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(value))
	sig, err := rsa.SignPKCS1v15(rand.Reader,key,crypto.SHA256,digest[:])
	if err != nil { t.Fatal(err) }
	return value+"."+base64.RawURLEncoding.EncodeToString(sig)
}

func TestOIDCProviderAuthorizationAndAutomaticGroupEnrollment(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	users:=seedGroups(t)

	privateKey, err:=rsa.GenerateKey(rand.Reader,2048)
	if err!=nil { t.Fatal(err) }
	var issuer, validNonce string
	var tokenRequests, userInfoRequests int
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","application/json")
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"issuer":issuer,"authorization_endpoint":issuer+"/authorize",
				"token_endpoint":issuer+"/token","jwks_uri":issuer+"/keys",
				"userinfo_endpoint":issuer+"/userinfo",
				"id_token_signing_alg_values_supported":[]string{"RS256"},
			})
		case "/keys":
			e:=big.NewInt(int64(privateKey.PublicKey.E))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"keys":[]map[string]string{{
					"kty":"RSA","kid":"test-signing-key","use":"sig","alg":"RS256",
					"n":base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
					"e":base64.RawURLEncoding.EncodeToString(e.Bytes()),
				}},
			})
		case "/token":
			if r.Method!=http.MethodPost {
				http.Error(w,"POST required",http.StatusMethodNotAllowed);return
			}
			tokenRequests++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token":"test-provider-access-token",
				"token_type":"Bearer",
				"expires_in":3600,
				"id_token":signedOIDCTestToken(t,privateKey,issuer,validNonce),
			})
		case "/userinfo":
			if r.Header.Get("Authorization")!="Bearer test-provider-access-token" {
				http.Error(w,"invalid access token",http.StatusUnauthorized);return
			}
			userInfoRequests++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"sub":"oidc-test-subject","preferred_username":"oidc-member",
				"name":"OIDC Test Member","email":"oidc@example.test","email_verified":true,
			})
		default:
			http.NotFound(w,r)
		}
	}))
	defer server.Close()
	issuer=server.URL

	oldConfig,oldLogger:=Config,Logger
	Config=&config.Config{}
	Config.Rustdesk.ApiServer="http://api.local.test"
	Logger=logrus.New()
	defer func(){Config=oldConfig;Logger=oldLogger}()

	pkceEnabled:=true
	autoRegister:=true
	if err:=DB.Create(&model.Oauth{
		Op:"local-oidc-test",OauthType:model.OauthTypeOidc,
		Issuer:issuer,ClientId:"rustdesk-api-oidc-test",
		ClientSecret:"local-oidc-secret",Scopes:"openid,profile,email",
		AutoRegister:&autoRegister,
		AutoGroupIds:"2,3",PkceEnable:&pkceEnabled,PkceMethod:model.PKCEMethodS256,
	}).Error;err!=nil{t.Fatal(err)}

	oauth:=AllService.OauthService
	if oauth==nil {t.Fatal("OAuth service is not initialized")}
	err,state,verifier,nonce,authorizationURL:=oauth.BeginAuth("local-oidc-test")
	if err!=nil{t.Fatal(err)}
	if state==""||nonce==""||verifier=="" {t.Fatal("OIDC state, nonce or PKCE verifier missing")}
	uri,err:=url.Parse(authorizationURL)
	if err!=nil{t.Fatal(err)}
	if uri.Host!=strings.TrimPrefix(issuer,"http://") {
		t.Fatalf("authorization host differs from configured issuer: %s",authorizationURL)
	}
	if uri.Query().Get("state")!=state||uri.Query().Get("nonce")!=nonce ||
		uri.Query().Get("code_challenge_method")!="S256"||uri.Query().Get("code_challenge")=="" {
		t.Fatalf("authorization request missing state, nonce or PKCE: %s",authorizationURL)
	}

	validNonce=nonce
	err,profile:=oauth.Callback("fake-authorization-code",verifier,"local-oidc-test",nonce)
	if err!=nil{t.Fatal(err)}
	if profile==nil||profile.OpenId!="oidc-test-subject"||
		profile.Email!="oidc@example.test"||profile.Username!="oidc-member" {
		t.Fatalf("OIDC profile mismatch: %+v",profile)
	}
	if tokenRequests==0||userInfoRequests==0 {
		t.Fatalf("expected token exchange and userinfo, got %d and %d",tokenRequests,userInfoRequests)
	}

	member:=users["member"]
	if err:=AllService.GroupService.AutoJoinOAuthGroups(member.Id,"local-oidc-test");err!=nil{
		t.Fatal(err)
	}
	if err:=AllService.GroupService.AutoJoinOAuthGroups(member.Id,"local-oidc-test");err!=nil{
		t.Fatal(err)
	}
	ids,err:=AllService.GroupService.GroupIdsForUser(member)
	if err!=nil{t.Fatal(err)}
	if !includeID(ids,2)||!includeID(ids,3){
		t.Fatalf("OIDC group configuration was not applied: %v",ids)
	}
	members,err:=AllService.GroupService.MemberIds(2)
	if err!=nil {t.Fatal(err)}
	if len(members)!=1||members[0]!=member.Id {
		t.Fatalf("duplicate OIDC member created: %v",members)
	}
	// A signed ID token with a different nonce must be rejected.
	err,_,_,nextNonce,_:=oauth.BeginAuth("local-oidc-test")
	if err!=nil{t.Fatal(err)}
	validNonce="unexpected-nonce"
	err,_=oauth.Callback("another-code","","local-oidc-test",nextNonce)
	if err==nil{
		t.Fatal(fmt.Sprintf("OIDC accepted a token whose nonce differs from %q",nextNonce))
	}
}
