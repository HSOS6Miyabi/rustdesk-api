package service

import (
    "encoding/json"
    "errors"
    "fmt"
    "strings"
    "sync"
    "time"
)

// PushEventEnvelope is shared by live SSE consumers and JSON webhook receivers.
type PushEventEnvelope struct {
    ID string `json:"id"`
    Type string `json:"type"`
    OccurredAt string `json:"occurred_at"`
    DeviceID string `json:"device_id,omitempty"`
    FromPeer string `json:"from_peer,omitempty"`
    SourceIP string `json:"source_ip,omitempty"`
    ConnectionID int64 `json:"connection_id,omitempty"`
    Detail string `json:"detail,omitempty"`
    Title string `json:"title"`
    Body string `json:"body"`
}

func marshalPushEventEnvelope(event PushEvent, key, title, body string) ([]byte, error) {
    when := event.When
    if when.IsZero() { when = time.Now() }
    payload := PushEventEnvelope{
        ID: eventHash(event.Code + ":" + key),
        Type: event.Code,
        OccurredAt: when.UTC().Format(time.RFC3339Nano),
        DeviceID: pushShort(event.DeviceID, 253),
        FromPeer: pushShort(event.FromPeer, 253),
        SourceIP: pushShort(event.SourceIP, 64),
        ConnectionID: event.ConnID,
        Detail: pushShort(event.Detail, 2000),
        Title: title,
        Body: body,
    }
    return json.Marshal(payload)
}

const maxStreamConnections = 2048
const maxStreamsPerUser = 5
const streamBufferSize = 64

type StreamEvent struct {
    Type string
    Data []byte
}

// StreamSubscription holds one bounded stream queue. Its Done channel is
// closed when the connection is too slow or the handler is being removed.
type StreamSubscription struct {
    userID uint
    events map[string]struct{}
    Messages chan StreamEvent
    Done chan struct{}
    once sync.Once
}

func (subscription *StreamSubscription) close() {
    subscription.once.Do(func() { close(subscription.Done) })
}

type streamRegistry struct {
    sync.RWMutex
    byUser map[uint]map[*StreamSubscription]struct{}
    interested map[string]int
    count int
}

var activeStreams = streamRegistry{
    byUser: make(map[uint]map[*StreamSubscription]struct{}),
    interested: make(map[string]int),
}

// ValidateStreamEvents accepts an empty list as a subscription to all events.
func ValidateStreamEvents(events []string) ([]string, error) {
    if len(events) == 0 { return nil, nil }
    valid := make(map[string]struct{}, len(pushEventCatalog))
    for _, item := range pushEventCatalog { valid[item.Code] = struct{}{} }
    selected := make(map[string]struct{}, len(events))
    result := make([]string, 0, len(events))
    for _, input := range events {
        code := strings.TrimSpace(input)
        if _, ok := valid[code]; !ok {
            return nil, fmt.Errorf("unknown event type: %s", code)
        }
        if _, ok := selected[code]; !ok {
            selected[code] = struct{}{}
            result = append(result, code)
        }
    }
    return result, nil
}

func SubscribePushEvents(userID uint, events []string) (*StreamSubscription, error) {
    if userID == 0 { return nil, errors.New("user required") }
    codes, err := ValidateStreamEvents(events)
    if err != nil { return nil, err }
    subscription := &StreamSubscription{
        userID: userID,
        events: make(map[string]struct{}, len(codes)),
        Messages: make(chan StreamEvent, streamBufferSize),
        Done: make(chan struct{}),
    }
    if len(codes) == 0 { subscription.events["*"] = struct{}{} }
    for _, code := range codes { subscription.events[code] = struct{}{} }
    activeStreams.Lock()
    defer activeStreams.Unlock()
    if activeStreams.count >= maxStreamConnections ||
        len(activeStreams.byUser[userID]) >= maxStreamsPerUser {
        return nil, errors.New("too many active event streams")
    }
    if activeStreams.byUser[userID] == nil {
        activeStreams.byUser[userID] = make(map[*StreamSubscription]struct{})
    }
    activeStreams.byUser[userID][subscription] = struct{}{}
    for code := range subscription.events { activeStreams.interested[code]++ }
    activeStreams.count++
    signalStreamInterest()
    return subscription, nil
}

func UnsubscribePushEvents(subscription *StreamSubscription) {
    if subscription == nil { return }
    subscription.close()
    activeStreams.Lock()
    defer activeStreams.Unlock()
    streams := activeStreams.byUser[subscription.userID]
    if _, present := streams[subscription]; !present { return }
    delete(streams, subscription)
    if len(streams) == 0 { delete(activeStreams.byUser, subscription.userID) }
    for code := range subscription.events {
        activeStreams.interested[code]--
        if activeStreams.interested[code] == 0 { delete(activeStreams.interested, code) }
    }
    activeStreams.count--
    signalStreamInterest()
}

func HasLivePushSubscribers(code string) bool {
    activeStreams.RLock()
    found := activeStreams.interested[code] > 0 || activeStreams.interested["*"] > 0
    activeStreams.RUnlock()
    return found
}

func hasUserStreamSubscriber(users []uint, code string) bool {
    activeStreams.RLock()
    defer activeStreams.RUnlock()
    for _, userID := range users {
        for subscription := range activeStreams.byUser[userID] {
            if _, ok := subscription.events[code]; ok { return true }
            if _, ok := subscription.events["*"]; ok { return true }
        }
    }
    return false
}

// PublishLivePushForUsers never blocks a producer on a slow network client.
// A subscriber that exhausts its 64-message buffer is disconnected.
func PublishLivePushForUsers(users []uint, eventType string, data []byte) {
    activeStreams.RLock()
    defer activeStreams.RUnlock()
    seen := make(map[uint]struct{}, len(users))
    for _, userID := range users {
        if _, already := seen[userID]; already { continue }
        seen[userID] = struct{}{}
        for subscription := range activeStreams.byUser[userID] {
            if _, ok := subscription.events[eventType]; !ok {
                if _, wildcard := subscription.events["*"]; !wildcard { continue }
            }
            select {
            case <-subscription.Done:
            default:
                select {
                case subscription.Messages <- StreamEvent{Type: eventType, Data: data}:
                default: subscription.close()
                }
            }
        }
    }
}
