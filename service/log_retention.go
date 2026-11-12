package service

import (
    "errors"
    "fmt"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
    apilog "github.com/lejianwen/rustdesk-api/v2/lib/logger"
    "gorm.io/gorm"
)

const logCleanupBatchSize = 500

// LogCleanupRequest selects a recorded-history category and a deletion rule.
type LogCleanupRequest struct {
    Kind         string `json:"kind"`
    Mode         string `json:"mode"`
    Days         int    `json:"days"`
    Confirmation string `json:"confirmation,omitempty"`
}

type LogCleanupResult struct {
    Kind          string `json:"kind"`
    Name          string `json:"name"`
    RetentionDays int    `json:"retention_days"`
    Eligible      int64  `json:"eligible"`
    Deleted       int64  `json:"deleted"`
}

type logCleanupCategory struct {
    key      string
    name     string
    table    interface{}
    column   string
    unixTime bool
    statuses []string
    days     int
}

func logCleanupCategories(policy *model.AuditRetentionPolicy) []logCleanupCategory {
    return []logCleanupCategory{
        {key: "connection", name: "远程连接记录", table: &model.AuditConn{}, column: "created_at", days: policy.ConnectionDays},
        {key: "file", name: "文件传输记录", table: &model.AuditFile{}, column: "created_at", days: policy.FileDays},
        {key: "alarm", name: "安全告警", table: &model.AuditAlarm{}, column: "created_at", days: policy.AlarmDays},
        {key: "login", name: "用户登录记录", table: &model.LoginLog{}, column: "created_at", days: policy.LoginDays},
        {key: "operation", name: "管理操作记录", table: &model.AuditOperation{}, column: "created_at", days: policy.OperationDays},
        {key: "disconnect", name: "结束连接记录", table: &model.DeviceDisconnectRequest{}, column: "updated_at", statuses: []string{"completed", "expired"}, days: policy.DisconnectDays},
        {key: "settings_history", name: "设置修改历史", table: &model.SystemSettingHistory{}, column: "changed_at", days: policy.SettingsHistoryDays},
        {key: "push_sent", name: "个人推送成功记录", table: &model.PushDelivery{}, column: "sent_at", unixTime: true, statuses: []string{"sent"}, days: policy.PushSentDays},
        {key: "push_failed", name: "个人推送失败记录", table: &model.PushDelivery{}, column: "updated_at", statuses: []string{"failed"}, days: policy.PushFailedDays},
        {key: "notification_sent", name: "管理员邮件成功记录", table: &model.NotificationDelivery{}, column: "sent_at", unixTime: true, statuses: []string{"sent"}, days: policy.NotificationSentDays},
        {key: "notification_failed", name: "管理员邮件失败记录", table: &model.NotificationDelivery{}, column: "updated_at", statuses: []string{"failed"}, days: policy.NotificationFailedDays},
        {key: "push_outbox_failed", name: "个人推送事件处理失败记录", table: &model.PushEventOutbox{}, column: "updated_at", statuses: []string{"failed"}, days: policy.PushOutboxFailedDays},
        {key: "notification_outbox_failed", name: "管理员邮件事件处理失败记录", table: &model.NotificationEventOutbox{}, column: "updated_at", statuses: []string{"failed"}, days: policy.NotificationOutboxFailedDays},
        {key: "server_log", name: "服务器运行日志文件", days: policy.ServerLogDays},
    }
}

func validateLogCleanupRequest(request LogCleanupRequest) error {
    if request.Kind == "" || request.Mode == "" {
        return errors.New("请选择记录类别和清理方式")
    }
    switch request.Mode {
    case "retention", "all":
    case "older":
        if request.Days < 1 || request.Days > 3650 {
            return errors.New("手动指定的保留天数必须在 1 至 3650 之间")
        }
    default:
        return errors.New("不支持的清理方式")
    }
    return nil
}

func logCleanupQuery(category logCleanupCategory, request LogCleanupRequest, now time.Time) (*gorm.DB, bool) {
    days := request.Days
    if request.Mode == "retention" {
        days = category.days
        if days == 0 {
            return nil, false
        }
    }
    query := DB.Model(category.table)
    if len(category.statuses) != 0 {
        query = query.Where("status IN ?", category.statuses)
    }
    if request.Mode != "all" {
        cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
        if category.unixTime {
            query = query.Where(category.column+" < ?", cutoff.Unix())
        } else {
            query = query.Where(category.column+" < ?", cutoff)
        }
    }
    return query, true
}

func runLogCleanup(request LogCleanupRequest, now time.Time, preview bool, requireConfirmation bool, selected map[string]bool) ([]LogCleanupResult, error) {
    if err := validateLogCleanupRequest(request); err != nil {
        return nil, err
    }
    if !preview && requireConfirmation && request.Confirmation != "DELETE_LOGS" {
        return nil, errors.New("请先确认删除操作")
    }
    if DB == nil {
        return nil, errors.New("数据库尚未初始化")
    }
    policy, err := GetAuditRetention()
    if err != nil {
        return nil, err
    }
    categories := logCleanupCategories(policy)
    validKind := request.Kind == "all"
    for _, category := range categories {
        if request.Kind == category.key {
            validKind = true
            break
        }
    }
    if !validKind {
        return nil, errors.New("不支持的记录类别")
    }
    results := make([]LogCleanupResult, 0, len(categories))
    for _, category := range categories {
        if request.Kind != "all" && request.Kind != category.key {
            continue
        }
        if selected != nil && !selected[category.key] {
            continue
        }
        item := LogCleanupResult{Kind: category.key, Name: category.name, RetentionDays: category.days}
        if category.key == "server_log" {
            days := request.Days
            if request.Mode == "retention" { days = category.days }
            if request.Mode == "all" { days = 0 }
            if (request.Mode != "retention" || days > 0) && Config != nil {
                count, err := apilog.ProcessLogArchives(Config.Logger.Path, now, days, !preview)
                if err != nil { return nil, fmt.Errorf("清理服务器运行日志失败: %w", err) }
                if preview { item.Eligible = count } else { item.Deleted = count }
            }
            results = append(results, item)
            continue
        }
        if !DB.Migrator().HasTable(category.table) {
            results = append(results, item)
            continue
        }
        query, enabled := logCleanupQuery(category, request, now)
        if !enabled {
            results = append(results, item)
            continue
        }
        if preview {
            if err := query.Count(&item.Eligible).Error; err != nil {
                return nil, fmt.Errorf("统计%s失败: %w", category.name, err)
            }
        } else {
            for {
                var ids []uint
                if err := query.Order("id ASC").Limit(logCleanupBatchSize).Pluck("id", &ids).Error; err != nil {
                    return nil, fmt.Errorf("读取%s失败: %w", category.name, err)
                }
                if len(ids) == 0 {
                    break
                }
                conditional, _ := logCleanupQuery(category, request, now)
                deletion := conditional.Where("id IN ?", ids).Delete(category.table)
                if deletion.Error != nil {
                    return nil, fmt.Errorf("清理%s失败: %w", category.name, deletion.Error)
                }
                item.Deleted += deletion.RowsAffected
                if len(ids) < logCleanupBatchSize {
                    break
                }
            }
        }
        results = append(results, item)
    }
    return results, nil
}

// PreviewLogCleanup reports eligible records without deleting them.
func PreviewLogCleanup(request LogCleanupRequest, now time.Time) ([]LogCleanupResult, error) {
    return runLogCleanup(request, now, true, false, nil)
}

// RunLogCleanup deletes selected records while protecting active work.
func RunLogCleanup(request LogCleanupRequest, now time.Time) ([]LogCleanupResult, error) {
    return runLogCleanup(request, now, false, true, nil)
}

// PruneAdditionalLogRecords applies retention to ended sessions, settings
// history, and unsuccessful event preparation records.
func PruneAdditionalLogRecords(now time.Time) error {
    _, err := runLogCleanup(LogCleanupRequest{Kind: "all", Mode: "retention"}, now, false, false,
        map[string]bool{
            "disconnect": true, "settings_history": true,
            "push_outbox_failed": true, "notification_outbox_failed": true,
            "server_log": true,
        })
    return err
}
