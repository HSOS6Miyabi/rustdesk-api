package service

import (
    "errors"
    "fmt"
    "sort"
    "strings"
    "sync"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/model"
    "gorm.io/gorm"
    "gorm.io/gorm/clause"
)

var ErrSettingsConflict = errors.New("设置已被其他管理员修改，请重新加载后再保存")
var settingsUpdateMu sync.Mutex

type SettingsChange struct {
    Key string `json:"key"`
    Enabled bool `json:"enabled"`
    Value *string `json:"value,omitempty"`
}
type SettingState struct {
    config.SettingDefinition
    CurrentValue string `json:"current_value,omitempty"`
    Configured bool `json:"configured"`
    Source string `json:"source"`
    OverrideEnabled bool `json:"override_enabled"`
    PendingRestart bool `json:"pending_restart"`
    StoredValue string `json:"stored_value,omitempty"`
}
type SettingsView struct {
    BootstrapVersion int64 `json:"bootstrap_version"`
    DatabaseVersion int64 `json:"database_version"`
    Items []SettingState `json:"items"`
}
type SettingsRevisionInput struct {
    BootstrapVersion int64 `json:"bootstrap_version"`
    DatabaseVersion int64 `json:"database_version"`
    Changes []SettingsChange `json:"changes"`
}

func loadDatabaseSettingRows() (map[string]config.SettingOverride, error) {
    var rows []model.SystemSettingOverride
    if err:=DB.Find(&rows).Error;err!=nil{return nil,err}
    out:=make(map[string]config.SettingOverride,len(rows))
    for _,row:=range rows {out[row.Key]=config.SettingOverride{Enabled:row.Enabled,Value:row.Value}}
    return out,nil
}
func LoadConfiguredSettings(target *config.Config) error {
    overrides,err:=loadDatabaseSettingRows()
    if err!=nil{return fmt.Errorf("读取数据库设置失败: %w",err)}
    if err:=config.ApplyDatabaseOverrides(target,overrides);err!=nil{return err}
    config.CaptureRunningSettings(*target)
    return DB.Clauses(clause.OnConflict{DoNothing:true}).
        Create(&model.SystemSettingRevision{ID:1,Version:1}).Error
}
func settingsRevision() (int64,error) {
    var row model.SystemSettingRevision
    err:=DB.First(&row,1).Error
    return row.Version,err
}
func SettingStates() (SettingsView,error) {
    view:=SettingsView{Items:make([]SettingState,0)}
    var err error
    store,err:=config.ReadBootstrapSettings()
    if err!=nil{return view,err}
    rows,err:=loadDatabaseSettingRows()
    if err!=nil{return view,err}
    view.BootstrapVersion=store.Version
    view.DatabaseVersion,err=settingsRevision()
    if err!=nil{return view,err}
    for _,def:=range config.SettingDefinitions() {
        active:=rows[def.Key]
        if def.Bootstrap {active=store.Overrides[def.Key]}
        running:=config.RunningSetting(def.Key)
        value:=running.Value
        state:=SettingState{SettingDefinition:def,CurrentValue:value,Configured:value!="",
            Source:running.Source,OverrideEnabled:active.Enabled}
        if def.Sensitive {
            state.CurrentValue=""
            state.Configured=value!=""
        } else if active.Enabled {
            state.StoredValue=active.Value
        }
        if active.Enabled && (value!=active.Value || running.Source!="网页设置") {
            state.PendingRestart=true
        }
        if !active.Enabled && running.Source=="网页设置" {
            state.PendingRestart=true
        }
        view.Items=append(view.Items,state)
    }
    return view,nil
}
func validateChanges(changes []SettingsChange) (bool,error) {
    if len(changes)==0||len(changes)>50{return false,errors.New("每次需要修改 1 至 50 项配置")}
    seen:=make(map[string]bool)
    boot,db:=false,false
    for _,item:=range changes {
        def,ok:=config.SettingInfo(item.Key)
        if !ok||!def.Active{return false,fmt.Errorf("不支持的设置：%s",item.Key)}
        if seen[item.Key] {return false,errors.New("同一设置不能在一次请求中重复修改")}
        seen[item.Key]=true
        if def.Bootstrap {boot=true}else{db=true}
        if item.Enabled && item.Value!=nil {
            if err:=config.ValidateSettingValue(item.Key,*item.Value);err!=nil{return false,fmt.Errorf("%s: %w",item.Key,err)}
        }
    }
    if boot&&db{return false,errors.New("启动设置与应用设置请分别保存")}
    return boot,nil
}
func ValidateSettingChanges(changes []SettingsChange) error {
    _,err:=validateChanges(changes)
    return err
}
func SaveSettingChanges(actor uint, input SettingsRevisionInput) error {
    if actor == 0 { return errors.New("需要系统管理员身份") }
    isBootstrap,err:=validateChanges(input.Changes)
    if err!=nil{return err}
    settingsUpdateMu.Lock()
    defer settingsUpdateMu.Unlock()

    now:=time.Now()
    storage:="database"
    if isBootstrap {storage="bootstrap"}
    history:=make([]model.SystemSettingHistory,0,len(input.Changes))
    for _,change:=range input.Changes{
        history=append(history,model.SystemSettingHistory{
            ActorID:actor,SettingKey:change.Key,Enabled:change.Enabled,
            Storage:storage,ChangedAt:now,
        })
    }

    if isBootstrap {
        original,readErr:=config.ReadBootstrapSettings()
        if readErr!=nil{return readErr}
        if original.Version!=input.BootstrapVersion{return ErrSettingsConflict}
        updated:=config.BootstrapSettings{
            Version:original.Version+1,Overrides:make(map[string]config.SettingOverride,len(original.Overrides)),
        }
        for key,value:=range original.Overrides {updated.Overrides[key]=value}
        for _,change:=range input.Changes{
            old:=updated.Overrides[change.Key]
            if change.Enabled&&change.Value==nil&&!old.Enabled {
                return fmt.Errorf("%s 尚未设置覆盖值",change.Key)
            }
            if change.Value!=nil {old.Value=*change.Value}
            old.Enabled=change.Enabled
            if old.Enabled {
                if err:=config.ValidateSettingValue(change.Key,old.Value);err!=nil{return err}
            }
            updated.Overrides[change.Key]=old
        }
        if err:=validateChangedDatabaseSettings(input.Changes,updated);err!=nil{return err}
        // Both the audit entry and the file update must succeed.
        err=DB.Transaction(func(tx *gorm.DB) error {
            if err:=tx.CreateInBatches(&history,50).Error;err!=nil{return err}
            return config.WriteBootstrapSettings(updated)
        })
        if err!=nil {
            // A transaction failure may follow the file replacement. Restore
            // the startup configuration whenever this request wrote it.
            if current,readErr:=config.ReadBootstrapSettings();readErr==nil&&current.Version==updated.Version {
                if restoreErr:=config.WriteBootstrapSettings(original);restoreErr!=nil {
                    return fmt.Errorf("保存启动设置未完成且无法恢复原配置: %w; %v",err,restoreErr)
                }
            }
        }
        return err
    }

    return DB.Transaction(func(tx *gorm.DB) error {
        updated:=tx.Model(&model.SystemSettingRevision{}).
            Where("id = ? AND version = ?",1,input.DatabaseVersion).
            Update("version",input.DatabaseVersion+1)
        if updated.Error!=nil{return updated.Error}
        if updated.RowsAffected==0{return ErrSettingsConflict}
        for _,change:=range input.Changes {
            var previous model.SystemSettingOverride
            query:=tx.Where("key = ?",change.Key).Take(&previous)
            if query.Error!=nil&&!errors.Is(query.Error,gorm.ErrRecordNotFound){return query.Error}
            if change.Enabled&&change.Value==nil&&!previous.Enabled{
                return fmt.Errorf("%s 尚未设置覆盖值",change.Key)
            }
            if change.Value!=nil {previous.Value=*change.Value}
            if change.Enabled {
                if err:=config.ValidateSettingValue(change.Key,previous.Value);err!=nil{return err}
            }
            row:=model.SystemSettingOverride{Key:change.Key,Value:previous.Value,
                Enabled:change.Enabled,UpdatedAt:now}
            if err:=tx.Clauses(clause.OnConflict{
                Columns:[]clause.Column{{Name:"key"}},
                DoUpdates:clause.AssignmentColumns([]string{"enabled","value","updated_at"}),
            }).Create(&row).Error;err!=nil{return err}
        }
        return tx.CreateInBatches(&history,50).Error
    })
}
func ResetSettingChanges(actor uint, input SettingsRevisionInput) error {
    for i:=range input.Changes {
        input.Changes[i].Enabled=false
        input.Changes[i].Value=nil
    }
    return SaveSettingChanges(actor,input)
}
func ListSettingHistory(limit int) ([]model.SystemSettingHistory,error) {
    if limit<1||limit>200 {limit=100}
    var list []model.SystemSettingHistory
    err:=DB.Order("id DESC").Limit(limit).Find(&list).Error
    return list,err
}
func SpecializedSettingsSummary() (map[string]interface{},error) {
    categories:=[]struct{key string; model interface{}}{
        {"push_channels",&model.PushChannel{}},
        {"relay_nodes",&model.RelayNode{}},
        {"device_policies",&model.DevicePolicy{}},
        {"access_rules",&model.AccessRule{}},
        {"client_build_tasks",&model.ClientBuildTask{}},
    }
    totals:=make(map[string]interface{})
    for _,category:=range categories {
        var amount int64
        if err:=DB.Model(category.model).Count(&amount).Error;err!=nil{return nil,err}
        totals[category.key]=amount
    }

    smtp,err:=GetPublicMailSettings()
    if err!=nil{return nil,err}
    notification,err:=CurrentNotificationSettings()
    if err!=nil{return nil,err}
    retention,err:=GetAuditRetention()
    if err!=nil{return nil,err}
    totals["smtp_settings"]=smtp
    totals["notification_settings"]=map[string]interface{}{
        "enabled":notification.Enabled,"events":notification.Events,"recipients":notification.Recipients,
    }
    totals["audit_retention"]=retention
    return totals,nil
}
func SettingsCategoryNames() []string {
    unique:=make(map[string]bool)
    for _,def:=range config.SettingDefinitions(){unique[def.Category]=true}
    names:=make([]string,0,len(unique))
    for category:=range unique{names=append(names,category)}
    sort.Strings(names)
    return names
}
func SettingChangesHaveSensitive(changes []SettingsChange) bool {
    for _,change:=range changes{def,ok:=config.SettingInfo(change.Key);if ok&&def.Sensitive{return true}}
    return false
}
func SettingsTextSearch(items []SettingState,query string) []SettingState {
    query=strings.ToLower(strings.TrimSpace(query))
    if query==""{return items}
    result:=make([]SettingState,0)
    for _,item:=range items{
        text:=strings.ToLower(item.Key+" "+item.Name+" "+item.Description+" "+item.Environment)
        if strings.Contains(text,query){result=append(result,item)}
    }
    return result
}
