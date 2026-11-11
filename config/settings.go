package config

import (
    "encoding/json"
    "errors"
    "fmt"
    "net"
    "os"
    "path/filepath"
    "reflect"
    "sort"
    "strconv"
    "strings"
    "sync"
    "time"
    "github.com/spf13/viper"
)

type SettingDefinition struct {
    Key string `json:"key"`
    Name string `json:"name"`
    Description string `json:"description"`
    Category string `json:"category"`
    ValueType string `json:"value_type"`
    Environment string `json:"environment"`
    Sensitive bool `json:"sensitive"`
    Bootstrap bool `json:"bootstrap"`
    RequiresRestart bool `json:"requires_restart"`
    Active bool `json:"active"`
}
type SettingOverride struct {
    Enabled bool `json:"enabled"`
    Value string `json:"value"`
}
type BootstrapSettings struct {
    Version int64 `json:"version"`
    Overrides map[string]SettingOverride `json:"overrides"`
}
type SettingRunningValue struct {
    Value string
    Source string
}

const BootstrapSettingsPath = "data/settings-bootstrap.json"

var settingCatalog = struct {
    sync.RWMutex
    base Config
    running Config
    hasRunning bool
    viper *viper.Viper
    bootstrap BootstrapSettings
    appliedDatabase map[string]SettingOverride
    runtime map[string]string
}{}

var fields = make(map[string][]int)
var definitions = make(map[string]SettingDefinition)
var sortedKeys []string
var registryOnce sync.Once

var settingsMetadata = "lang|语言|控制台与服务接口使用的语言\napp.web-client|网页客户端|控制网页客户端入口是否启用\napp.register|开放注册|允许新用户自行申请账户\napp.register-status|新账户状态|选择新注册账户的初始状态：1 启用、2 禁用\napp.show-swagger|接口文档|是否公开管理接口的 Swagger 页面\napp.token-expire|登录有效期|普通账户登录令牌的有效时间\napp.web-sso|网页登录单点登录|控制网页中的单点登录入口\napp.disable-pwd-login|停用密码登录|阻止账户使用密码登录，启用前应确认其他登录方式可用\napp.captcha-threshold|验证码阈值|登录失败达到多少次后要求填写验证码，负数表示关闭\napp.ban-threshold|IP 封禁阈值|登录失败达到多少次后封禁 IP 地址，0 表示关闭\napp.require-device-deployment|设备审批登记|启用后要求设备通过部署审批才能登记\napp.internal-key|内部通信密钥|API 与 hbbs 共用的内部接口认证密钥\nadmin.title|控制台标题|设置管理控制台显示的标题\nadmin.hello|欢迎文本|设置账户登录后显示的欢迎说明\nadmin.hello-file|欢迎文本文件|读取欢迎说明的文件路径\nadmin.id-server-port|ID 服务管理端口|发送 RustDesk ID 服务器管理命令的端口\nadmin.relay-server-port|中继服务管理端口|发送 RustDesk 中继管理命令的端口\ngin.api-addr|HTTP 监听地址|API 监听的 IP 地址和端口\ngin.admin-addr|管理端监听地址|保留配置项，当前版本未单独启用管理监听端口\ngin.mode|服务运行模式|选择 release、debug 或 test\ngin.resources-path|静态资源目录|控制台网页和静态文件所在的目录\ngin.trust-proxy|可信代理|设置可信反向代理地址，影响真实客户端 IP 的识别\ngorm.type|数据库类型|选择 sqlite、mysql 或 postgresql\ngorm.max-idle-conns|空闲数据库连接数|连接池允许保留的空闲连接数\ngorm.max-open-conns|数据库连接数上限|连接池的最大连接数，SQLite 实际最多四个\nmysql.addr|MySQL 地址|MySQL 服务器地址和端口\nmysql.username|MySQL 用户名|连接 MySQL 时使用的账户\nmysql.password|MySQL 密码|连接 MySQL 时使用的密码\nmysql.dbname|MySQL 数据库名称|保存 RustDesk 管理数据的 MySQL 数据库\nmysql.tls|MySQL TLS 模式|设置 MySQL 连接的证书验证方式\npostgresql.host|PostgreSQL 地址|PostgreSQL 服务器主机名\npostgresql.port|PostgreSQL 端口|PostgreSQL 服务监听端口\npostgresql.user|PostgreSQL 用户名|连接 PostgreSQL 时使用的账户\npostgresql.password|PostgreSQL 密码|连接 PostgreSQL 时使用的密码\npostgresql.dbname|PostgreSQL 数据库名称|保存管理数据的 PostgreSQL 数据库\npostgresql.sslmode|PostgreSQL TLS 模式|指定 PostgreSQL TLS 验证方式\npostgresql.time-zone|PostgreSQL 时区|数据库连接使用的会话时区\nrustdesk.id-server|ID 服务器地址|客户端连接的 RustDesk ID 服务器地址\nrustdesk.relay-server|中继服务器地址|客户端连接的 RustDesk 中继服务器地址\nrustdesk.api-server|公开 API 地址|客户端使用的 RustDesk API URL\nrustdesk.key|RustDesk 公钥|RustDesk 服务端的公开身份密钥\nrustdesk.key-file|RustDesk 公钥文件|从指定文件读取 RustDesk 服务端公钥\nrustdesk.personal|个人模式|原项目提供的个人模式控制选项\nrustdesk.webclient-magic-queryonline|网页在线查询模式|网页客户端特定的在线查询行为\nrustdesk.ws-host|WebSocket 公开地址|浏览器客户端的 WebSocket 代理地址\nlogger.path|日志文件|API 日志文件存放位置\nlogger.level|日志级别|控制调试、信息和错误日志的详细程度\nlogger.report-caller|记录调用位置|是否在日志中记录产生记录的源代码位置\nproxy.enable|启用网络代理|是否使用设置的 HTTP 网络代理\nproxy.host|网络代理地址|代理服务器 URL\njwt.key|JWT 签名密钥|用于签名账户访问令牌的密钥，变更可能使已有会话失效\njwt.expire-duration|JWT 有效时间|JWT 签名令牌的有效时间\nldap.enable|启用 LDAP|允许使用 LDAP 目录进行账户身份验证\nldap.url|LDAP 地址|LDAP 服务连接地址\nldap.tls-ca-file|LDAP CA 文件|LDAP TLS 服务器证书签发机构文件\nldap.tls-verify|验证 LDAP TLS|是否检查 LDAP 服务器的证书\nldap.base-dn|LDAP 基础 DN|LDAP 用户目录的基础查询位置\nldap.bind-dn|LDAP 绑定 DN|API 查询 LDAP 使用的账户 DN\nldap.bind-password|LDAP 绑定密码|API 查询 LDAP 使用的账户密码\nldap.user.base-dn|LDAP 用户目录|用户条目所在的目录位置\nldap.user.enable-attr|LDAP 启用属性|读取账户启用状态时检查的属性名\nldap.user.enable-attr-value|LDAP 属性值|代表账户启用状态的属性值\nldap.user.filter|LDAP 查询表达式|LDAP 用户搜索筛选条件\nldap.user.username|用户名属性|LDAP 用户名对应的字段\nldap.user.email|邮箱属性|LDAP 邮箱对应的字段\nldap.user.first-name|名字属性|LDAP 名字对应的字段\nldap.user.last-name|姓氏属性|LDAP 姓氏对应的字段\nldap.user.sync|同步账户资料|登录或创建用户时是否同步 LDAP 用户资料\nldap.user.admin-group|LDAP 管理员组|标识 LDAP 管理员的用户组 DN\nldap.user.allow-group|LDAP 允许登录组|允许使用 LDAP 登录的用户组 DN\nredis.addr|Redis 地址|兼容现有组件使用的 Redis 服务地址\nredis.password|Redis 密码|Redis 连接认证密码\nredis.db|Redis 数据库编号|Redis 逻辑数据库编号\ncache.type|缓存方式|选择内存、文件或 Redis 缓存\ncache.redis-addr|缓存 Redis 地址|Redis 缓存使用的服务器地址\ncache.redis-pwd|缓存 Redis 密码|Redis 缓存的连接密码\ncache.redis-db|缓存 Redis 编号|Redis 缓存使用的逻辑数据库\ncache.file-dir|文件缓存目录|本地文件缓存的保存目录\noss.access-key-id|对象存储访问标识|对象存储服务使用的访问标识\noss.access-key-secret|对象存储访问密钥|对象存储服务的私密访问凭据\noss.host|对象存储服务地址|对象存储服务访问地址\noss.callback-url|对象存储回调地址|上传完成后返回的回调地址\noss.expire-time|对象存储链接期限|上传访问链接允许的有效时间\noss.max-byte|上传文件大小上限|限制上传对象的字节数\nruntime.shared-event-workers|共享通知任务|多个 API 实例使用同一数据库时启用共享任务认领\nruntime.geoip-db|GeoIP 城市数据库|本地 GeoLite2 City MMDB 数据库文件路径\nruntime.push-geoip-db|兼容 GeoIP 文件路径|仅在主要 GeoIP 路径未填写时读取的旧环境变量\nruntime.push-encryption-key|推送加密密钥|用于加密个人消息推送渠道的凭据\nruntime.mail-encryption-key|邮件加密密钥|用于加密 SMTP、双重认证等持久保存的秘密\nruntime.push-allow-private-urls|允许内部推送地址|允许推送访问内部网络 HTTP 地址，启用会增加安全风险\nruntime.multi-instance-cache|多实例缓存同步|多台 API 共享数据库时将策略和通知订阅状态的缓存有效时间缩短为五秒"

func settingCategory(key string) string {
    prefix := strings.SplitN(key, ".", 2)[0]
    switch prefix {
    case "lang", "app", "admin": return "常规与登录"
    case "gin": return "HTTP 服务"
    case "gorm", "mysql", "postgresql": return "数据库"
    case "rustdesk": return "RustDesk 服务"
    case "logger": return "日志"
    case "jwt": return "令牌"
    case "ldap": return "LDAP"
    case "cache", "redis": return "缓存"
    case "oss": return "对象存储"
    case "runtime": return "附加服务"
    case "proxy": return "网络代理"
    }
    return "其他"
}
func sensitiveSetting(key string) bool {
    if strings.Contains(key, "password") || strings.Contains(key, "secret") ||
        strings.Contains(key, "encryption-key") { return true }
    switch key {
    case "app.internal-key", "jwt.key", "cache.redis-pwd", "redis.password": return true
    }
    return false
}
func bootstrapSetting(key string) bool {
    if sensitiveSetting(key) { return true }
    for _, prefix := range []string{"gorm.", "mysql.", "postgresql.", "gin.", "logger.", "jwt.",
        "cache.", "redis.", "oss.", "runtime."} {
        if strings.HasPrefix(key, prefix) { return true }
    }
    return key == "rustdesk.key-file" || key == "admin.hello-file"
}
func runtimeEnv(key string) string {
    special := map[string]string{
        "runtime.shared-event-workers":"RUSTDESK_API_SHARED_EVENT_WORKERS",
        "runtime.geoip-db":"RUSTDESK_API_GEOIP_DB",
        "runtime.push-geoip-db":"RUSTDESK_API_PUSH_GEOIP_DB",
        "runtime.push-encryption-key":"RUSTDESK_API_PUSH_ENCRYPTION_KEY",
        "runtime.mail-encryption-key":"RUSTDESK_API_MAIL_ENCRYPTION_KEY",
        "runtime.push-allow-private-urls":"RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS",
        "runtime.multi-instance-cache":"RUSTDESK_API_MULTI_INSTANCE_CACHE",
    }
    if env := special[key]; env != "" { return env }
    return "RUSTDESK_API_" + strings.ToUpper(strings.NewReplacer(".","_","-","_").Replace(key))
}
func registerFields(typ reflect.Type, indices []int, prefix string) {
    for i:=0; i<typ.NumField(); i++ {
        field := typ.Field(i)
        if field.PkgPath != "" { continue }
        name := field.Tag.Get("mapstructure")
        if name == "-" { continue }
        if name == "" { name = strings.ToLower(field.Name) }
        key := name
        if prefix != "" { key = prefix + "." + name }
        ix := append(append([]int(nil), indices...),i)
        if field.Type.Kind()==reflect.Struct && field.Type != reflect.TypeOf(time.Duration(0)) {
            registerFields(field.Type,ix,key)
            continue
        }
        fields[key]=ix
    }
}
func initSettingsCatalog() {
    registryOnce.Do(func() {
        registerFields(reflect.TypeOf(Config{}),nil,"")
        description := make(map[string][2]string)
        for _, line := range strings.Split(settingsMetadata,"\n") {
            parts := strings.SplitN(line,"|",3)
            if len(parts)==3 { description[parts[0]]=[2]string{parts[1],parts[2]} }
        }
        for key, ix := range fields {
            typ := reflect.TypeOf(Config{}).FieldByIndex(ix).Type
            valueType := "string"
            if typ == reflect.TypeOf(time.Duration(0)) { valueType="duration" } else {
                switch typ.Kind() {
                case reflect.Bool: valueType="bool"
                case reflect.Int, reflect.Int64: valueType="int"
                }
            }
            info := description[key]
            if info[0] == "" { info[0]=key; info[1]="应用配置项 "+key }
            definitions[key]=SettingDefinition{
                Key:key,Name:info[0],Description:info[1],Category:settingCategory(key),
                ValueType:valueType, Environment:runtimeEnv(key),
                Sensitive:sensitiveSetting(key),Bootstrap:bootstrapSetting(key),
                RequiresRestart:true,Active:key!="gin.admin-addr",
            }
        }
        for _,key:=range []string{"runtime.shared-event-workers","runtime.geoip-db",
            "runtime.push-geoip-db","runtime.push-encryption-key",
            "runtime.mail-encryption-key","runtime.push-allow-private-urls","runtime.multi-instance-cache"} {
            info:=description[key]
            kind:="string"
            if key=="runtime.shared-event-workers" || key=="runtime.push-allow-private-urls" || key=="runtime.multi-instance-cache" { kind="bool" }
            definitions[key]=SettingDefinition{
                Key:key,Name:info[0],Description:info[1],Category:settingCategory(key),
                ValueType:kind,Environment:runtimeEnv(key),Sensitive:sensitiveSetting(key),
                Bootstrap:true,RequiresRestart:true,Active:true,
            }
        }
        for key:=range definitions { sortedKeys=append(sortedKeys,key) }
        sort.Strings(sortedKeys)
    })
}
func SettingDefinitions() []SettingDefinition {
    initSettingsCatalog()
    out:=make([]SettingDefinition,0,len(sortedKeys))
    for _,key:=range sortedKeys { out=append(out,definitions[key]) }
    return out
}
func SettingInfo(key string) (SettingDefinition,bool) {
    initSettingsCatalog()
    value,ok:=definitions[key]
    return value,ok
}
func parseSettingsBoolean(value string)(bool,error) {
    switch strings.ToLower(strings.TrimSpace(value)) {
    case "true","1","yes","y","on":return true,nil
    case "false","0","no","n","off","":return false,nil
    }
    return false,errors.New("布尔值应填写 true 或 false")
}
func ValidateSettingValue(key,value string) error {
    def,ok:=SettingInfo(key)
    if !ok || !def.Active { return fmt.Errorf("未知或当前不受支持的设置：%s",key) }
    if len(value)>8192 { return errors.New("配置内容过长") }
    switch def.ValueType {
    case "bool":
        if _,err:=parseSettingsBoolean(value);err!=nil{return err}
    case "int":
        v,err:=strconv.ParseInt(value,10,64);if err!=nil{return fmt.Errorf("请输入整数：%w",err)}
        if v<0 && key!="app.captcha-threshold" { return errors.New("此配置值不能为负数") }
        if strings.HasSuffix(key,"-port")||key=="postgresql.port" {
            if v<1||v>65535{return errors.New("端口应在 1 至 65535 之间")}
        }
        if key=="app.register-status"&&(v!=1&&v!=2) { return errors.New("账户状态只能设置为 1 或 2") }
        if (key=="app.web-client"||key=="app.show-swagger") && v!=0 && v!=1 {
            return errors.New("此项设置只能填写 0 或 1")
        }
    case "duration":
        v,err:=time.ParseDuration(value)
        if err!=nil || v<0 { return errors.New("时间应采用 30s、5m、168h 等格式") }
    }
    switch key {
    case "gorm.type":
        if value!="sqlite"&&value!="mysql"&&value!="postgresql"{return errors.New("数据库类型只能为 sqlite、mysql 或 postgresql")}
    case "cache.type":
        if value!=""&&value!="file"&&value!="redis"&&value!="memory"{return errors.New("缓存类型无效")}
    case "gin.mode":
        if value!="release"&&value!="debug"&&value!="test"{return errors.New("Gin 模式无效")}
    case "logger.level":
        if value!="trace"&&value!="debug"&&value!="info"&&value!="warn"&&value!="error"&&value!="fatal"{return errors.New("日志级别无效")}
    case "gin.api-addr":
        if _,_,err:=net.SplitHostPort(value);err!=nil{return fmt.Errorf("监听地址必须包含主机与端口：%w",err)}
    case "runtime.shared-event-workers","runtime.push-allow-private-urls","runtime.multi-instance-cache":
        _,err:=parseSettingsBoolean(value);return err
    }
    return nil
}
func ApplySettingValue(c *Config,key,value string) error {
    if err:=ValidateSettingValue(key,value);err!=nil{return err}
    ix,ok:=fields[key]
    if !ok { return nil }
    target:=reflect.ValueOf(c).Elem().FieldByIndex(ix)
    if target.Type()==reflect.TypeOf(time.Duration(0)){
        duration,_:=time.ParseDuration(value);target.SetInt(int64(duration));return nil
    }
    switch target.Kind(){
    case reflect.Bool:
        b,_:=parseSettingsBoolean(value);target.SetBool(b)
    case reflect.Int,reflect.Int64:
        n,_:=strconv.ParseInt(value,10,64);target.SetInt(n)
    case reflect.String:
        target.SetString(value)
    default:return errors.New("不支持的设置类型")
    }
    return nil
}
func ReadSettingValue(c Config,key string) string {
    initSettingsCatalog()
    ix,ok:=fields[key]
    if !ok { return "" }
    val:=reflect.ValueOf(c).FieldByIndex(ix)
    if val.Type()==reflect.TypeOf(time.Duration(0)) { return time.Duration(val.Int()).String() }
    switch val.Kind(){
    case reflect.String:return val.String()
    case reflect.Int,reflect.Int64:return strconv.FormatInt(val.Int(),10)
    case reflect.Bool:return strconv.FormatBool(val.Bool())
    }
    return ""
}
func ReadBootstrapSettings() (BootstrapSettings,error) {
    store:=BootstrapSettings{Overrides:make(map[string]SettingOverride)}
    data,err:=os.ReadFile(BootstrapSettingsPath)
    if errors.Is(err,os.ErrNotExist) { return store,nil }
    if err!=nil { return store,err }
    if err:=json.Unmarshal(data,&store);err!=nil{return store,fmt.Errorf("read bootstrap settings: %w",err)}
    if store.Overrides==nil{store.Overrides=make(map[string]SettingOverride)}
    for key,entry:=range store.Overrides {
        def,ok:=SettingInfo(key)
        if !ok||!def.Bootstrap { return store,fmt.Errorf("invalid bootstrap setting: %s",key) }
        if entry.Enabled {if err:=ValidateSettingValue(key,entry.Value);err!=nil{return store,err}}
    }
    return store,nil
}
func WriteBootstrapSettings(store BootstrapSettings) error {
    folder:=filepath.Dir(BootstrapSettingsPath)
    if err:=os.MkdirAll(folder,0700);err!=nil{return err}
    f,err:=os.CreateTemp(folder,".bootstrap-*.tmp")
    if err!=nil{return err}
    name:=f.Name()
    defer os.Remove(name)
    if err:=f.Chmod(0600);err!=nil{f.Close();return err}
    data,err:=json.MarshalIndent(store,"","  ")
    if err!=nil{f.Close();return err}
    if _,err=f.Write(append(data,'\n'));err!=nil{f.Close();return err}
    if err=f.Sync();err!=nil{f.Close();return err}
    if err=f.Close();err!=nil{return err}
    if err=os.Rename(name,BootstrapSettingsPath);err!=nil{return err}
    if dir,err:=os.Open(folder);err==nil{_ = dir.Sync();_ = dir.Close()}
    return nil
}
func BaseConfigSnapshot() Config {
    settingCatalog.RLock();defer settingCatalog.RUnlock()
    return settingCatalog.base
}
func markSettingsBase(base Config, v *viper.Viper) {
    settingCatalog.Lock();defer settingCatalog.Unlock()
    settingCatalog.base=base
    settingCatalog.viper=v
}
func loadBootstrapOverrides(c *Config) error {
    store,err:=ReadBootstrapSettings();if err!=nil{return err}
    for key,entry:=range store.Overrides {
        if !entry.Enabled {continue}
        if _,ok:=fields[key];ok {
            if err:=ApplySettingValue(c,key,entry.Value);err!=nil{return err}
        }
    }
    settingCatalog.Lock()
    settingCatalog.bootstrap=store
    settingCatalog.Unlock()
    return nil
}
func ApplyDatabaseOverrides(c *Config, overrides map[string]SettingOverride) error {
    for key,entry:=range overrides {
        def,ok:=SettingInfo(key)
        if !ok||def.Bootstrap { return fmt.Errorf("invalid database setting: %s",key) }
        if entry.Enabled {
            if err:=ApplySettingValue(c,key,entry.Value);err!=nil{return err}
        }
    }
    settingCatalog.Lock()
    settingCatalog.appliedDatabase=overrides
    settingCatalog.Unlock()
    return nil
}
func RuntimeOption(key string) string {
    settingCatalog.RLock()
    value,ok:=settingCatalog.runtime[key]
    override:=settingCatalog.bootstrap.Overrides[key]
    settingCatalog.RUnlock()
    if ok && override.Enabled { return value }
    return readRuntimeEnvironment(key)
}
func RuntimeBool(key string) bool {
    value:=RuntimeOption(key)
    result,_:=parseSettingsBoolean(value)
    return result
}
func readRuntimeEnvironment(key string) string {
    env:=runtimeEnv(key)
    val:=os.Getenv(env)
    if key=="runtime.geoip-db"&&val=="" {val=os.Getenv("RUSTDESK_API_PUSH_GEOIP_DB")}
    return val
}
func CaptureRunningSettings(c Config) {
    initSettingsCatalog()
    extras:=make(map[string]string)
    store,err:=ReadBootstrapSettings()
    if err==nil {
        for _,key:=range sortedKeys {
            if !strings.HasPrefix(key,"runtime.") { continue }
            extra:=readRuntimeEnvironment(key)
            if val,ok:=store.Overrides[key];ok&&val.Enabled { extra=val.Value }
            extras[key]=extra
        }
    }
    settingCatalog.Lock()
    settingCatalog.running=c
    settingCatalog.runtime=extras
    settingCatalog.hasRunning=true
    settingCatalog.Unlock()
}
func RunningSetting(key string) SettingRunningValue {
    def,ok:=SettingInfo(key)
    if !ok {return SettingRunningValue{}}
    settingCatalog.RLock()
    cfg,base:=settingCatalog.running,settingCatalog.base
    bootstrap:=settingCatalog.bootstrap.Overrides[key]
    database:=settingCatalog.appliedDatabase[key]
    current:=settingCatalog.runtime[key]
    v:=settingCatalog.viper
    settingCatalog.RUnlock()
    if !strings.HasPrefix(key,"runtime.") {current=ReadSettingValue(cfg,key)}
    source:="程序默认值"
    if def.Bootstrap&&bootstrap.Enabled || !def.Bootstrap&&database.Enabled {
        source="网页设置"
    } else if _,exists:=os.LookupEnv(def.Environment);exists {
        source="环境变量"
    } else if key=="app.internal-key"&&os.Getenv("RUSTDESK_API_INTERNAL_KEY")!="" ||
      key=="app.require-device-deployment"&&os.Getenv("REQUIRE_DEVICE_DEPLOYMENT")!="" ||
      key=="runtime.geoip-db"&&os.Getenv("RUSTDESK_API_PUSH_GEOIP_DB")!="" {
        source="环境变量"
    } else if v!=nil && v.InConfig(key) {
        source="配置文件"
    } else if ReadSettingValue(base,key)!="" {
        source="程序默认值"
    }
    return SettingRunningValue{Value:current,Source:source}
}
