package api

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestDeviceDeploymentRequestValidation(t *testing.T) {
	uuid := base64.RawURLEncoding.EncodeToString([]byte("device-uuid-value"))
	key := base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("k", 32)))
	form := &deviceDeploymentRequest{Id: "desk-123456", Uuid: uuid, Pk: key}
	if !validateDeployment(form) {
		t.Fatal("valid deployment request rejected")
	}
	if form.Uuid != base64.StdEncoding.EncodeToString([]byte("device-uuid-value")) {
		t.Fatal("device UUID was not normalized")
	}
	for _, id := range []string{"short", "bad id", strings.Repeat("x", 254)} {
		f := &deviceDeploymentRequest{Id: id, Uuid: uuid, Pk: key}
		if validateDeployment(f) {
			t.Fatalf("invalid ID %q accepted", id)
		}
	}
	for _, broken := range []string{"", "%%%"} {
		f := &deviceDeploymentRequest{Id: "desk-123456", Uuid: broken, Pk: key}
		if validateDeployment(f) {
			t.Fatalf("invalid UUID %q accepted", broken)
		}
	}
	f := &deviceDeploymentRequest{Id: "desk-123456", Uuid: uuid, Pk: base64.StdEncoding.EncodeToString([]byte("too-short"))}
	if validateDeployment(f) {
		t.Fatal("short public key accepted")
	}
}
