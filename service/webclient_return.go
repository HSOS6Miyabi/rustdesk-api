package service

import (
    "net/url"
    "strings"
)

// SafeWebClientReturnPath accepts only a local WebClient2 URL with an optional
// device identifier and display name. It never accepts external destinations.
func SafeWebClientReturnPath(value string) string {
    if value == "/webclient2/" {
        return value
    }
    if !strings.HasPrefix(value, "/webclient2/?") {
        return ""
    }
    target, err := url.Parse(value)
    if err != nil || target.IsAbs() || target.Host != "" || target.User != nil ||
        target.Fragment != "" || target.Path != "/webclient2/" {
        return ""
    }
    values, err := url.ParseQuery(target.RawQuery)
    if err != nil {
        return ""
    }
    for key, items := range values {
        if (key != "id" && key != "name") || len(items) != 1 {
            return ""
        }
    }
    id := strings.TrimSpace(values.Get("id"))
    if id == "" || len(id) > 128 || strings.ContainsAny(id, "\r\n\x00") {
        return ""
    }
    params := url.Values{"id": {id}}
    name := strings.TrimSpace(values.Get("name"))
    if name != "" {
        if len(name) > 160 || strings.ContainsAny(name, "\r\n\x00") {
            return ""
        }
        params.Set("name", name)
    }
    return "/webclient2/?" + params.Encode()
}
