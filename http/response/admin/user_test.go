package admin

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestLoginPayloadReportsLocalPasswordAvailability(t *testing.T) {
	for _, item := range []struct {
		hash string
		expected bool
	}{
		{"", false},
		{"$2a$10$example-stored-password-hash", true},
	} {
		var response LoginPayload
		response.FromUser(&model.User{Password: item.hash})
		if response.HasLocalPassword != item.expected {
			t.Fatalf("password availability for %q: got %v", item.hash, response.HasLocalPassword)
		}
	}
}
