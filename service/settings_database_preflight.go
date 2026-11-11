package service

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "net"
    "net/url"
    "strconv"
    "strings"
    "time"

    mysql "github.com/go-sql-driver/mysql"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/lejianwen/rustdesk-api/v2/config"
    "github.com/lejianwen/rustdesk-api/v2/global"
)

func databaseTargetChanged(current, proposed config.Config) bool {
    if current.Gorm.Type!=proposed.Gorm.Type{return true}
    switch proposed.Gorm.Type {
    case config.TypeMysql: return current.Mysql!=proposed.Mysql
    case config.TypePostgresql: return current.Postgresql!=proposed.Postgresql
    default: return false
    }
}

func validateChangedDatabaseSettings(changes []SettingsChange,store config.BootstrapSettings) error {
    validate:=false
    for _,entry:=range changes {
        if entry.Enabled && (entry.Key=="gorm.type"||
            strings.HasPrefix(entry.Key,"mysql.")||strings.HasPrefix(entry.Key,"postgresql.")) {
            validate=true
            break
        }
    }
    if !validate{return nil}
    effective:=config.BaseConfigSnapshot()
    for key,entry:=range store.Overrides {
        if entry.Enabled {
            if err:=config.ApplySettingValue(&effective,key,entry.Value);err!=nil{return err}
        }
    }
    if !databaseTargetChanged(global.Config,effective){return nil}
    return verifyDatabaseTarget(effective)
}

func verifyDatabaseTarget(target config.Config) error {
    if target.Gorm.Type==config.TypeSqlite {return nil}
    var driver,dsn string
    switch target.Gorm.Type {
    case config.TypeMysql:
        opts:=mysql.NewConfig()
        opts.User=target.Mysql.Username
        opts.Passwd=target.Mysql.Password
        opts.Net="tcp"
        opts.Addr=target.Mysql.Addr
        opts.DBName=target.Mysql.Dbname
        opts.ParseTime=true
        opts.TLSConfig=target.Mysql.Tls
        if opts.TLSConfig=="" {opts.TLSConfig="false"}
        driver,dsn="mysql",opts.FormatDSN()
    case config.TypePostgresql:
        host:=strings.TrimSpace(target.Postgresql.Host)
        port:=strings.TrimSpace(target.Postgresql.Port)
        portNumber,err:=strconv.Atoi(port)
        if err!=nil||portNumber<1||portNumber>65535 {
            return errors.New("PostgreSQL 端口需要填写 1 至 65535 之间的整数")
        }
        if host=="" {return errors.New("请填写 PostgreSQL 服务器地址")}
        u:=url.URL{Scheme:"postgres",Host:net.JoinHostPort(host,port),
            Path:"/"+target.Postgresql.Dbname,
            User:url.UserPassword(target.Postgresql.User,target.Postgresql.Password)}
        q:=u.Query()
        if target.Postgresql.Sslmode!="" {q.Set("sslmode",target.Postgresql.Sslmode)}
        if target.Postgresql.TimeZone!="" {q.Set("TimeZone",target.Postgresql.TimeZone)}
        u.RawQuery=q.Encode()
        driver,dsn="pgx",u.String()
    default:
        return errors.New("数据库类型无效")
    }
    db,err:=sql.Open(driver,dsn)
    if err!=nil{return errors.New("无法验证目标数据库的连接参数")}
    defer db.Close()
    ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
    defer cancel()
    if err:=db.PingContext(ctx);err!=nil {
        return errors.New("无法连接目标数据库，请检查服务地址、网络和登录凭据")
    }
    var version sql.NullInt64
    if err:=db.QueryRowContext(ctx,"SELECT MAX(version) FROM versions").Scan(&version);err!=nil {
        return errors.New("目标数据库缺少已有的 RustDesk API 数据，请先完成数据库迁移")
    }
    if !version.Valid||version.Int64<=0 {
        return fmt.Errorf("目标数据库没有有效的迁移版本，无法切换")
    }
    var users int64
    if err:=db.QueryRowContext(ctx,"SELECT COUNT(*) FROM users").Scan(&users);err!=nil||users==0 {
        return errors.New("目标数据库缺少原有用户记录，请先恢复数据库备份")
    }
    return nil
}
