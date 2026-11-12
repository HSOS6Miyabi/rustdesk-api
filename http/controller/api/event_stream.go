package api

import (
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/service"
)

type EventStream struct{}

// Stream exposes a bounded per-user SSE subscription to an events.read API key.
// Browsers should use fetch with an Authorization header to avoid URL credentials.
func (*EventStream) Stream(c *gin.Context) {
    credential := c.GetHeader("Authorization")
    if !strings.HasPrefix(credential, "Bearer rdapi_") {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "events.read API key required"})
        return
    }
    token := strings.TrimPrefix(credential, "Bearer ")
    user, err := service.AuthenticateAPIToken(token, "events.read")
    if err != nil {
        c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid event API key"})
        return
    }
    raw := c.Query("events")
    var filter []string
    if raw != "" {
        filter = strings.Split(raw, ",")
        if len(filter) > 32 {
            c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "too many events"})
            return
        }
    }
    if _, err := service.ValidateStreamEvents(filter); err != nil {
        c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    subscription, err := service.SubscribePushEvents(user.Id, filter)
    if err != nil {
        c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
        return
    }
    defer service.UnsubscribePushEvents(subscription)

    c.Header("Content-Type", "text/event-stream; charset=utf-8")
    c.Header("Cache-Control", "no-cache, no-transform")
    c.Header("X-Accel-Buffering", "no")
    c.Header("Content-Encoding", "identity")
    c.Header("Vary", "Authorization")
    c.Status(http.StatusOK)
    _, _ = c.Writer.WriteString("retry: 3000\n: connected\n\n")
    c.Writer.Flush()

    heartbeat := time.NewTicker(20 * time.Second)
    defer heartbeat.Stop()
    lastAuth := time.Now()
    for {
        select {
        case <-c.Request.Context().Done(): return
        case <-subscription.Done: return
        case event := <-subscription.Messages:
            if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, event.Data); err != nil { return }
            c.Writer.Flush()
        case <-heartbeat.C:
            if time.Since(lastAuth) >= time.Minute {
                if _, err := service.AuthenticateAPIToken(token, "events.read"); err != nil { return }
                lastAuth = time.Now()
            }
            if _, err := c.Writer.WriteString(": heartbeat\n\n"); err != nil { return }
            c.Writer.Flush()
        }
    }
}
