package service

import (
    "errors"
    "time"
    "gorm.io/gorm"
    "github.com/lejianwen/rustdesk-api/v2/model"
)

const outboxMaxAttempts = 8

func recordOutboxFailure(table interface{}, id uint, attempts int, cause error, owner string, permanent bool) error {
    attempts++
    status := "retry"
    wait := int64(0)
    if permanent || attempts >= outboxMaxAttempts {
        status = "failed"
    } else if attempts > 1 {
        exponent := attempts - 2
        if exponent > 7 { exponent = 7 }
        wait = int64(5 << exponent)
    }
    message := cause.Error()
    if len(message) > 240 { message = message[:240] }
    query:=DB.Model(table).Where("id = ?", id)
    if owner!="" {query=query.Where("lease_owner = ?",owner)}
    return query.Updates(map[string]interface{}{
        "status": status, "attempts": attempts, "last_error": message,
        "next_attempt_at": time.Now().Unix()+wait, "lease_until": int64(0), "lease_owner":"",
    }).Error
}

func nextOutboxDelay(table interface{}) (time.Duration, bool, error) {
    if DB == nil { return 0, false, nil }
    var row struct { Due *int64 `gorm:"column:due"` }
    expression := "MIN(next_attempt_at) AS due"
    if sharedEventWorkersEnabled() {
        expression = "MIN(CASE WHEN lease_until > next_attempt_at THEN lease_until ELSE next_attempt_at END) AS due"
    }
    if err := DB.Model(table).Select(expression).Where("status IN ?", []string{"pending", "retry"}).Scan(&row).Error; err != nil {
        return 0, false, err
    }
    if row.Due == nil { return 0, false, nil }
    delay := time.Until(time.Unix(*row.Due, 0))
    if delay < 0 { delay = 0 }
    return delay, true, nil
}

func RetryFailedOutboxEvent(category string, id uint) error {
    if id == 0 { return errors.New("invalid event identifier") }
    var table interface{}
    var wake func()
    switch category {
    case "push":
        table, wake = &model.PushEventOutbox{}, wakePushOutbox
    case "notification":
        table, wake = &model.NotificationEventOutbox{}, wakeNotificationOutbox
    default:
        return errors.New("invalid event category")
    }
    result := DB.Model(table).Where("id = ? AND status = ?", id, "failed").
        Updates(map[string]interface{}{"status":"pending", "attempts":0, "last_error":"",
            "next_attempt_at":int64(0), "lease_until":int64(0)})
    if result.Error != nil { return result.Error }
    if result.RowsAffected == 0 { return errors.New("event is not eligible for retry") }
    wake()
    return nil
}

type OutboxStatus struct {
    Pending int64 `json:"pending"`
    Failed int64 `json:"failed"`
    OldestPendingAt *time.Time `json:"oldest_pending_at"`
}

func ReadOutboxStatus(table interface{}) (OutboxStatus, error) {
    var stats OutboxStatus
    if err := DB.Model(table).Where("status IN ?", []string{"pending", "retry"}).Count(&stats.Pending).Error; err != nil { return stats, err }
    if err := DB.Model(table).Where("status = ?", "failed").Count(&stats.Failed).Error; err != nil { return stats, err }
    if stats.Pending>0 {
        var oldest struct {CreatedAt *time.Time `gorm:"column:created_at"`}
        if err:=DB.Model(table).Select("MIN(created_at) AS created_at").Where("status IN ?",[]string{"pending","retry"}).Scan(&oldest).Error;err!=nil{return stats,err}
        stats.OldestPendingAt=oldest.CreatedAt
    }
    return stats,nil
}

func FailedOutboxEntries(category string, limit int) (interface{}, error) {
    if limit <= 0 || limit > 100 { limit = 50 }
    switch category {
    case "push":
        var rows []model.PushEventOutbox
        err := DB.Select("id, status, attempts, last_error, created_at").
            Where("status = ?", "failed").Order("id DESC").Limit(limit).Find(&rows).Error
        return rows, err
    case "notification":
        var rows []model.NotificationEventOutbox
        err := DB.Select("id, status, attempts, last_error, created_at").
            Where("status = ?", "failed").Order("id DESC").Limit(limit).Find(&rows).Error
        return rows, err
    }
    return nil, gorm.ErrInvalidData
}
