package admin

import "testing"

func TestValidateRustdeskCommand(t *testing.T) {
	tests := []struct {
		name, target, command, option string
		valid bool
	}{
		{"id help", "21115", "h", "", true},
		{"relay help", "21117", "h", "", true},
		{"read relay", "21115", "relay-servers", "", true},
		{"set relay", "21115", "relay-servers", "relay.example.com:21117,relay2.example.com:21117", true},
		{"block ip", "21115", "ip-blocker", "192.0.2.1", true},
		{"unblock ip", "21115", "ip-blocker", "192.0.2.1 -", true},
		{"relay enabled", "21115", "always-use-relay", "y", true},
		{"geo", "21115", "test-geo", "192.0.2.1 2001:db8::1", true},
		{"blocklist", "21117", "blocklist-add", "203.0.113.8", true},
		{"zero limit", "21117", "limit-speed", "0", true},
		{"decimal bandwidth", "21117", "single-bandwidth", "2.5", true},
		{"usage", "21117", "usage", "", true},
		{"unknown", "21117", "exec", "rm", false},
		{"wrong target", "21115", "blacklist-add", "203.0.113.8", false},
		{"negative threshold", "21117", "downgrade-threshold", "-1", false},
		{"invalid geo", "21115", "test-geo", "bad 192.0.2.1", false},
		{"invalid ip", "21117", "blacklist-add", "999.999.999.999", false},
		{"extra args", "21117", "usage", "h", false},
		{"newline", "21117", "blacklist-add", "192.0.2.1\nusage", false},
		{"semicolon", "21115", "relay-servers", "good;bad", false},
		{"empty IP", "21117", "blocklist", "", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRustdeskCommand(test.target, test.command, test.option)
			if (err == nil) != test.valid {
				t.Fatalf("validateRustdeskCommand(%q,%q,%q) error = %v, valid = %v", test.target, test.command, test.option, err, test.valid)
			}
		})
	}
}
