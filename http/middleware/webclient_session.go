package middleware

import (
    "net/http"
    "net/url"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

const webClientSessionCookie = "rustdesk_webclient_session"

func webClientRequestHTTPS(c *gin.Context) bool {
    if c.Request.TLS != nil {
        return true
    }
    protocol := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0])
    return strings.EqualFold(protocol, "https")
}

// SetWebClientSessionCookie grants the authenticated browser access to WebClient files.
func SetWebClientSessionCookie(c *gin.Context, token string) {
    if token == "" {
        return
    }
    http.SetCookie(c.Writer, &http.Cookie{
        Name: webClientSessionCookie,
        Value: token,
        Path: "/",
        HttpOnly: true,
        Secure: webClientRequestHTTPS(c),
        SameSite: http.SameSiteLaxMode,
    })
}

func ClearWebClientSessionCookie(c *gin.Context) {
    http.SetCookie(c.Writer, &http.Cookie{
        Name: webClientSessionCookie,
        Path: "/",
        HttpOnly: true,
        Secure: webClientRequestHTTPS(c),
        SameSite: http.SameSiteLaxMode,
        MaxAge: -1,
    })
}

func webClientLoginLocation(c *gin.Context) string {
    params := url.Values{"return": {"webclient2"}}
    if strings.HasPrefix(c.Request.URL.Path, "/webclient2/") {
        requested := c.Request.URL.Query()
        id := strings.TrimSpace(requested.Get("id"))
        if id != "" && len(id) <= 128 && !strings.ContainsAny(id, "\r\n\x00") {
            params.Set("id", id)
            name := strings.TrimSpace(requested.Get("name"))
            if name != "" && len(name) <= 160 && !strings.ContainsAny(name, "\r\n\x00") {
                params.Set("name", name)
            }
        }
    }
    return "/_admin/#/login?" + params.Encode()
}

// WebClientSessionAuth verifies the active account in the database on every request.
func WebClientSessionAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Cache-Control", "private, no-store")
        c.Header("Vary", "Cookie")
        c.Header("X-Content-Type-Options", "nosniff")
        token, err := c.Cookie(webClientSessionCookie)
        if err == nil && token != "" {
            user, _ := (&service.UserService{}).InfoByAccessToken(token)
            if user.Id != 0 && (&service.UserService{}).CheckUserEnable(user) {
                c.Next()
                return
            }
            ClearWebClientSessionCookie(c)
        }
        path := c.Request.URL.Path
        if c.Request.Method == http.MethodGet &&
            (path == "/webclient2/" || path == "/webclient2/index.html" ||
                path == "/webclient/" || path == "/webclient/index.html") {
            c.Redirect(http.StatusFound, webClientLoginLocation(c))
        } else {
            c.Status(http.StatusUnauthorized)
        }
        c.Abort()
    }
}
