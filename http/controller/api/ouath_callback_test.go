package api

import (
  "net/url"
  "strings"
  "testing"
)

func TestWebAdminOAuthCompletionURL(t *testing.T) {
  cases := []struct {
    state string
    returnTo string
    expectedReturn string
  }{
    {"MobileState12345", "", ""},
    {"MobileState12345", "/webclient2/", "webclient2"},
    {"MobileState12345", "https://example.invalid/redirect", ""},
    {"MobileState12345", "//example.invalid", ""},
    {"State With&Special?", "/webclient2/", "webclient2"},
  }
  for _, tc := range cases {
    uri, err := url.Parse(webAdminOAuthCompletionURL(tc.state, tc.returnTo))
    if err != nil { t.Fatal(err) }
    if uri.Path != "/_admin/" || uri.RawQuery != "" {
      t.Fatalf("authorization code must stay in the URL fragment: %s", uri)
    }
    fragment, rawQuery, ok := strings.Cut(uri.EscapedFragment(), "?")
    if !ok || fragment != "/oidc/complete" {
      t.Fatalf("invalid SSO completion route: %s", uri)
    }
    params, err := url.ParseQuery(rawQuery)
    if err != nil { t.Fatal(err) }
    if params.Get("code") != tc.state || params.Get("return") != tc.expectedReturn {
      t.Fatalf("incorrect callback state or destination: %s", uri)
    }
    if len(params) != 1 && (tc.expectedReturn == "" || len(params) != 2) {
      t.Fatalf("unexpected callback parameters: %s", uri)
    }
  }
}
