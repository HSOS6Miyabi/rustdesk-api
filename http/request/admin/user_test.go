package admin

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestChangeCurPasswordFormSupportsFirstPasswordAndExistingPasswords(t *testing.T) {
	validate := validator.New()
	for _, item := range []struct {
		name, oldPassword, newPassword string
		valid bool
	}{
		{"external account first password", "", "FirstLocalPassword2026", true},
		{"existing local password", "OldPassword2026", "NewPassword2026", true},
		{"72-character existing password", strings.Repeat("p", 72), "NewPassword2026", true},
		{"too short replacement", "", "short", false},
		{"old password shorter than supported", "bad", "NewPassword2026", false},
		{"old password longer than supported", strings.Repeat("p", 73), "NewPassword2026", false},
	} {
		t.Run(item.name, func(t *testing.T) {
			err := validate.Struct(&ChangeCurPasswordForm{OldPassword: item.oldPassword, NewPassword: item.newPassword})
			if (err == nil) != item.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
		})
	}
}
