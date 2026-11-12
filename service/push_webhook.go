package service

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

// sendPushWebhook posts the structured event directly to the configured URL.
// The shared HTTP client enforces TLS, address restrictions and timeouts.
func sendPushWebhook(channel *model.PushChannel, delivery *model.PushDelivery) error {
    if channel == nil || delivery == nil { return errors.New("missing webhook message") }
    endpoint, err := validPushURL(channel.BaseURL)
    if err != nil { return err }
    data := []byte(delivery.Payload)
    if len(data) == 0 {
        occurred := time.Time(delivery.CreatedAt)
        if occurred.IsZero() { occurred = time.Now() }
        data, err = json.Marshal(PushEventEnvelope{
            ID: delivery.EventKey,
            Type: delivery.EventType,
            OccurredAt: occurred.UTC().Format(time.RFC3339Nano),
            Title: delivery.Title,
            Body: delivery.Body,
        })
        if err != nil { return err }
    }
    req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(data))
    if err != nil { return err }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("User-Agent", "RustDesk-API-Push/1.0")
    if channel.KeyCiphertext != "" {
        key, err := decryptPushSecret(channel.KeyCiphertext)
        if err != nil { return err }
        req.Header.Set("Authorization", "Bearer "+key)
    }
    res, err := pushHTTP.Do(req)
    if err != nil { return errors.New("webhook connection failed or timed out") }
    defer res.Body.Close()
    _, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
    if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
        return fmt.Errorf("webhook HTTP %d", res.StatusCode)
    }
    return nil
}
