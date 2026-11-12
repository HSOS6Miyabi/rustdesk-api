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
		{"zero limit", "21117", "limit-speed", "0", false},
        {"threshold ratio", "21117", "downgrade-threshold", "0.66", true},
        {"invalid threshold ratio", "21117", "downgrade-threshold", "2", false},
        {"fractional check seconds", "21117", "downgrade-start-check", "1.2", false},
        {"valid check seconds", "21117", "downgrade-start-check", "12", true},
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
		{"read all blocklist entries", "21117", "blocklist", "", true},
        {"read all blacklist entries", "21117", "blacklist", "", true},
        {"must login enabled", "21115", "must-login", "y", true},
        {"must login state", "21115", "must-login", "", true},
        {"reload GeoIP", "21115", "reload-geo", "", true},
        {"recent punch requests", "21115", "punch-requests", "", true},
        {"must login invalid", "21117", "must-login", "y", false},
        {"no argument reload", "21115", "reload-geo", "abc", false},
        {"no argument punch", "21115", "punch-requests", "-", false},
        {"empty IP for adding blacklist", "21117", "blacklist-add", "", false},
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
