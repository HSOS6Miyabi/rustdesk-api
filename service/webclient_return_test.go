package service

import "testing"

func TestSafeWebClientReturnPath(t *testing.T) {
    cases := []struct {
        input string
        output string
    }{
        {"/webclient2/", "/webclient2/"},
        {"/webclient2/?id=464956471&name=meili", "/webclient2/?id=464956471&name=meili"},
        {"/webclient2/?name=meili&id=123", "/webclient2/?id=123&name=meili"},
        {"/webclient2/?id=a+b", "/webclient2/?id=a+b"},
        {"https://outside.example/", ""},
        {"//outside.example/secret", ""},
        {"/webclient2/?id=1&return=https://outside.example", ""},
        {"/webclient2/?id=1&id=2", ""},
        {"/webclient2/?name=only-name", ""},
        {"/webclient/?id=1", ""},
        {"/webclient2/?id=%00", ""},
        {"/webclient2/?id=123#bad", ""},
    }
    for _, tc := range cases {
        if got := SafeWebClientReturnPath(tc.input); got != tc.output {
            t.Errorf("for %q got %q, want %q", tc.input, got, tc.output)
        }
    }
}
