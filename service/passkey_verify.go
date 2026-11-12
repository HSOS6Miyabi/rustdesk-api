package service

import (
    "bytes"
    "io"
    "net/http"

    "github.com/go-webauthn/webauthn/webauthn"
)

// Each WebAuthn response is verified by the upstream FIDO2 implementation.
// A self-contained request lets the service use its parser while keeping
// the controller's JSON envelope for the one-time challenge identifier.
func passkeyResponseRequest(body []byte) *http.Request {
    r:=&http.Request{
        Method:http.MethodPost,
        Header:make(http.Header),
        Body:io.NopCloser(bytes.NewReader(body)),
        ContentLength:int64(len(body)),
    }
    r.Header.Set("Content-Type","application/json")
    return r
}

func VerifyPasskeyRegistrationResponse(verifier *webauthn.WebAuthn,
    user *PasskeyAccount,session webauthn.SessionData,body []byte)(*webauthn.Credential,error){
    return verifier.FinishRegistration(user,session,passkeyResponseRequest(body))
}

func VerifyPasskeyLoginResponse(verifier *webauthn.WebAuthn,
    find webauthn.DiscoverableUserHandler,session webauthn.SessionData,
    body []byte)(webauthn.User,*webauthn.Credential,error){
    return verifier.FinishPasskeyLogin(find,session,passkeyResponseRequest(body))
}
