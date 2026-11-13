package main

import (
	"strings"
	"fmt"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/lib/orm"
	"github.com/lejianwen/rustdesk-api/v2/lib/upload"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

const DatabaseVersion = 291

// @title 管理系统API
// @version 1.0
// @description 接口
// @basePath /api
// @securityDefinitions.apikey token
// @in header
// @name api-token
// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization

var rootCmd = &cobra.Command{
	Use:   "apimain",
	Short: "RUSTDESK API SERVER",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return InitGlobal()
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		global.Logger.Info("API SERVER START")
		return http.ApiInit()
	},
}

var resetPwdCmd = &cobra.Command{
	Use:     "reset-admin-pwd [pwd]",
	Example: "reset-admin-pwd 123456",
	Short:   "Reset Admin Password",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwd := args[0]
		admin := service.AllService.UserService.InfoById(1)
		if admin.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err := service.AllService.UserService.UpdatePassword(admin, pwd)
		if err != nil {
			global.Logger.Error("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success! ")
	},
}
var resetUserPwdCmd = &cobra.Command{
	Use:     "reset-pwd [userId] [pwd]",
	Example: "reset-pwd 2 123456",
	Short:   "Reset User Password",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		userId := args[0]
		pwd := args[1]
		uid, err := strconv.Atoi(userId)
		if err != nil {
			global.Logger.Warn("userId must be int!")
			return
		}
		if uid <= 0 {
			global.Logger.Warn("userId must be greater than 0! ")
			return
		}
		u := service.AllService.UserService.InfoById(uint(uid))
		if u.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err = service.AllService.UserService.UpdatePassword(u, pwd)
		if err != nil {
			global.Logger.Warn("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success!")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&global.ConfigPath, "config", "c", "./conf/config.yaml", "choose config file")
	rootCmd.AddCommand(resetPwdCmd, resetUserPwdCmd)
}
func main() {
	err:=rootCmd.Execute()
	workersStopped := service.StopServiceWorkers(30*time.Second)
	if !workersStopped && global.Logger != nil {
		global.Logger.Warn("background workers did not finish before shutdown")
	}
	if workersStopped && global.DB!=nil {
		if connection,dbErr:=global.DB.DB();dbErr==nil {
			if closeErr:=connection.Close();closeErr!=nil && global.Logger!=nil {
				global.Logger.WithError(closeErr).Warn("database close failed")
			}
		}
	}
	if closer,ok:=global.Cache.(interface{Close() error});ok {_=closer.Close()}
	if global.Redis!=nil {_=global.Redis.Close()}
	if err!=nil {
		if global.Logger!=nil {global.Logger.Error(err)}
		os.Exit(1)
	}
}

func InitGlobal() error {
	//配置解析
	global.Viper = config.Init(&global.Config, global.ConfigPath)

	//日志
	global.Logger = logger.New(&logger.Config{
		Path:         global.Config.Logger.Path,
		Level:        global.Config.Logger.Level,
		ReportCaller: global.Config.Logger.ReportCaller,
	})

	global.InitI18n()

	//gorm
	if global.Config.Gorm.Type == config.TypeMysql {

		dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			global.Config.Mysql.Username,
			global.Config.Mysql.Password,
			global.Config.Mysql.Addr,
			global.Config.Mysql.Dbname,
			global.Config.Mysql.Tls,
		)

		global.DB = orm.NewMysql(&orm.MysqlConfig{
			Dsn:          dsn,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
	} else if global.Config.Gorm.Type == config.TypePostgresql {
		dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
			global.Config.Postgresql.Host,
			global.Config.Postgresql.Port,
			global.Config.Postgresql.User,
			global.Config.Postgresql.Password,
			global.Config.Postgresql.Dbname,
			global.Config.Postgresql.Sslmode,
			global.Config.Postgresql.TimeZone,
		)
		global.DB = orm.NewPostgresql(&orm.PostgresqlConfig{
			Dsn:          dsn,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
	} else {
		//sqlite
		global.DB = orm.NewSqlite(&orm.SqliteConfig{
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
	}

	if err := DatabaseAutoUpdate(); err != nil { return fmt.Errorf("database startup migration: %w", err) }
	service.DB = global.DB
	if err := service.LoadConfiguredSettings(&global.Config); err != nil { return fmt.Errorf("load application settings: %w", err) }
	if err := initOptionalCache(); err != nil { return err }
	//validator
	global.ApiInitValidator()

	//oss
	global.Oss = &upload.Oss{
		AccessKeyId:     global.Config.Oss.AccessKeyId,
		AccessKeySecret: global.Config.Oss.AccessKeySecret,
		Host:            global.Config.Oss.Host,
		CallbackUrl:     global.Config.Oss.CallbackUrl,
		ExpireTime:      global.Config.Oss.ExpireTime,
		MaxByte:         global.Config.Oss.MaxByte,
	}

	//jwt
	//fmt.Println(global.Config.Jwt.PrivateKey)
	global.Jwt = jwt.NewJwt(global.Config.Jwt.Key, global.Config.Jwt.ExpireDuration)
	//locker
	global.Lock = lock.NewLocal()

	//service
	service.New(&global.Config, global.DB, global.Logger, global.Jwt, global.Lock)

	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{
		CaptchaThreshold: global.Config.App.CaptchaThreshold,
		BanThreshold:     global.Config.App.BanThreshold,
		AttemptsWindow:   10 * time.Minute,
		BanDuration:      30 * time.Minute,
	})
	global.LoginLimiter.RegisterProvider(utils.B64StringCaptchaProvider{})
	service.StartAuditMaintenance()
	service.StartSessionMaintenance()
	service.StartNotificationWorker()
	service.StartPushWorker()
	service.StartClientBuildWorker()
	if fileCache, ok := global.Cache.(*cache.FileCache); ok {
		service.StartFileCacheMaintenance(fileCache)
	}
	return nil
}

func redisConnectionOptions(addr, password string, db int) *redis.Options {
    return &redis.Options{
        Addr: addr, Password: password, DB: db,
        DialTimeout: 2 * time.Second,
        ReadTimeout: 2 * time.Second,
        WriteTimeout: 2 * time.Second,
    }
}

// initOptionalCache enables Redis when an address is configured and otherwise
// uses a bounded, expiring in-memory cache. Both Redis settings are supported.
func initOptionalCache() error {
    cfg := global.Config
    cacheType := strings.ToLower(strings.TrimSpace(cfg.Cache.Type))
    cacheAddr := strings.TrimSpace(cfg.Cache.RedisAddr)
    sharedAddr := strings.TrimSpace(cfg.Redis.Addr)
    sharedPassword, sharedDB := cfg.Redis.Password, cfg.Redis.Db

    if sharedAddr == "" && (cacheType == "" || cacheType == cache.TypeRedis) && cacheAddr != "" {
        sharedAddr = cacheAddr
        sharedPassword, sharedDB = cfg.Cache.RedisPwd, cfg.Cache.RedisDb
    }
    global.Redis = nil
    if sharedAddr != "" {
        global.Redis = redis.NewClient(redisConnectionOptions(sharedAddr, sharedPassword, sharedDB))
    }

    switch cacheType {
    case "", cache.TypeMem:
        if cacheType == "" && global.Redis != nil {
            global.Cache = cache.WrapRedis(global.Redis)
        } else {
            global.Cache = cache.NewMemoryCache(defaultMemoryCacheBytes)
        }
    case cache.TypeRedis:
        if cacheAddr == "" {
            if global.Redis != nil {
                global.Cache = cache.WrapRedis(global.Redis)
            } else {
                global.Cache = cache.NewMemoryCache(defaultMemoryCacheBytes)
                if global.Logger != nil {
                    global.Logger.Warn("Redis cache selected without an address; using in-memory cache")
                }
            }
        } else if global.Redis != nil && cacheAddr == sharedAddr &&
            cfg.Cache.RedisPwd == sharedPassword && cfg.Cache.RedisDb == sharedDB {
            global.Cache = cache.WrapRedis(global.Redis)
        } else {
            global.Cache = cache.NewRedis(redisConnectionOptions(
                cacheAddr, cfg.Cache.RedisPwd, cfg.Cache.RedisDb))
        }
    case cache.TypeFile:
        directory := cfg.Cache.FileDir
        if directory == "" { directory = "./data/cache" }
        if err := os.MkdirAll(directory, 0700); err != nil {
            return fmt.Errorf("prepare file cache directory: %w", err)
        }
        fileCache := cache.NewFileCache()
        fileCache.SetDir(directory)
        global.Cache = fileCache
        global.Config.Cache.FileDir = directory
    default:
        return fmt.Errorf("unsupported cache type %q", cfg.Cache.Type)
    }
    return nil
}

func DatabaseAutoUpdate() error {
	version := DatabaseVersion

	db := global.DB

	if global.Config.Gorm.Type == config.TypeMysql {
		//检查存不存在数据库，不存在则创建
		dbName := db.Migrator().CurrentDatabase()
		if dbName == "" {
			dbName = global.Config.Mysql.Dbname
			// 移除 DSN 中的数据库名称，以便初始连接时不指定数据库
			dsnWithoutDB := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				global.Config.Mysql.Username,
				global.Config.Mysql.Password,
				global.Config.Mysql.Addr,
				"",
			)

			//新链接
			dbWithoutDB := orm.NewMysql(&orm.MysqlConfig{
				Dsn: dsnWithoutDB,
			}, global.Logger)
			// 获取底层的 *sql.DB 对象，并确保在程序退出时关闭连接
			sqlDBWithoutDB, err := dbWithoutDB.DB()
			if err != nil {
				return fmt.Errorf("open MySQL administration connection: %w", err)
			}
			defer func() {
				if err := sqlDBWithoutDB.Close(); err != nil {
					global.Logger.Errorf("关闭连接失败: %v", err)
				}
			}()

			err = dbWithoutDB.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + " DEFAULT CHARSET utf8mb4").Error
			if err != nil {
				return fmt.Errorf("create MySQL database: %w", err)
			}
		}
	}

	if !db.Migrator().HasTable(&model.Version{}) {
		if err := Migrate(uint(version)); err != nil { return err }
	} else {
		//查找最后一个version
		var v model.Version
		if err := db.Last(&v).Error; err != nil { return fmt.Errorf("read database migration version: %w", err) }
		if v.Version < uint(version) {
			if err := Migrate(uint(version)); err != nil { return err }
		}

		// 245迁移
		if v.Version < 245 {
			//oauths 表的 oauth_type 字段设置为 op同样的值
			for _, statement := range []string{"update oauths set oauth_type = op", "update oauths set issuer = 'https://accounts.google.com' where op = 'google'", "update user_thirds set oauth_type = third_type, op = third_type"} {
				if err := db.Exec(statement).Error; err != nil { return fmt.Errorf("legacy OAuth migration: %w", err) }
			}
			//通过email迁移旧的google授权
			uts := make([]model.UserThird, 0)
			if err := db.Where("oauth_type = ?", "google").Find(&uts).Error; err != nil { return err }
			for _, ut := range uts {
				if ut.UserId > 0 {
					if err := db.Model(&model.User{}).Where("id = ?", ut.UserId).Update("email", ut.OpenId).Error; err != nil { return err }
				}
			}
		}
		if v.Version < 246 {
			if err := db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google' and issuer is null").Error; err != nil { return err }
		}
	}
	return nil
}
func Migrate(version uint) error {
	global.Logger.Info("Migrating....", version)
	err := global.DB.AutoMigrate(
		&model.Version{},
		&model.User{},
		&model.UserToken{},
		&model.Tag{},
		&model.AddressBook{},
		&model.Peer{},
		&model.Group{},
		&model.GroupMembership{},
		&model.GroupAnchor{},
		&model.Contact{},
		&model.UserThird{},
		&model.Oauth{},
		&model.LoginLog{},
		&model.ShareRecord{},
		&model.AuditConn{},
		&model.AuditFile{},
		&model.AuditAlarm{},
		&model.AuditEventNonce{},
		&model.AuditOperation{},
		&model.AuditRetentionPolicy{},
		&model.DevicePolicy{},
		&model.DevicePolicyAssignment{},
		&model.DevicePolicyUserAssignment{},
		&model.DevicePolicyGroupAssignment{},
		&model.DeviceDeployment{},
		&model.DeviceActiveConnection{},
		&model.DeviceDisconnectRequest{},
		&model.APIToken{},
		&model.SMTPSetting{},
		&model.NotificationSetting{},
		&model.NotificationDelivery{},
		&model.PushChannel{},
		&model.PushDelivery{},
		&model.PushPresence{},
		&model.PushEventOutbox{},
		&model.NotificationEventOutbox{},
		&model.ClientBuildProfile{},
		&model.ClientBuildTask{},
		&model.PasswordResetToken{},
		&model.UserInvitation{},
		&model.UserTOTP{},
		&model.TOTPRecoveryCode{},
		&model.TOTPLoginChallenge{},
		&model.EmailLoginSetting{},
		&model.EmailCodeChallenge{},
        &model.UserPasskey{},
        &model.UserPasskeyHandle{},
        &model.UserPasskeyChallenge{},
		&model.RelayNode{},
		&model.ControlRole{},
		&model.ControlRoleAssignment{},
		&model.AdminRole{},
		&model.AdminRoleAssignment{},
		&model.AccessRule{},
		&model.AddressBookCollection{},
		&model.AddressBookCollectionRule{},
		&model.ServerCmd{},
		&model.DeviceGroup{},
		&model.SystemSettingOverride{},
		&model.SystemSettingRevision{},
		&model.SystemSettingHistory{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate database tables: %w", err)
	}
	if err:=ensureAuditCleanupIndexes(global.DB);err!=nil {
		return fmt.Errorf("create audit cleanup indexes: %w", err)
	}
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.Version{Version: version}).Error; err != nil { return fmt.Errorf("write database migration version: %w", err) }
		var vc int64
		if err := tx.Model(&model.Version{}).Count(&vc).Error; err != nil { return err }
		if vc == 1 {
		localizer := global.Localizer("")
		defaultGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "DefaultGroup",
		})
		group := &model.Group{
			Name: defaultGroup,
			Type: model.GroupTypeDefault,
		}
		if err := tx.Create(group).Error; err != nil { return err }

		shareGroup, _ := localizer.LocalizeMessage(&i18n.Message{
			ID: "ShareGroup",
		})
		groupShare := &model.Group{
			Name: shareGroup,
			Type: model.GroupTypeShare,
		}
		if err := tx.Create(groupShare).Error; err != nil { return err }
		//是true
		is_admin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "Admin",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &is_admin,
			GroupId:  1,
		}

		// 生成随机密码
		pwd := utils.RandomString(8)
		global.Logger.Info("Admin Password Is: ", pwd)
		var err error
		admin.Password, err = utils.EncryptPassword(pwd)
		if err != nil {
			global.Logger.Fatalf("failed to generate admin password: %v", err)
		}
		if err := tx.Create(admin).Error; err != nil { return err }
		}
		return nil
	})
}

func ensureAuditCleanupIndexes(db *gorm.DB) error {
	for _,entry:=range []struct{
		model interface{}
		table string
		name string
	}{
		{&model.AuditConn{},"audit_conns","idx_audit_conns_created_at"},
		{&model.AuditFile{},"audit_files","idx_audit_files_created_at"},
		{&model.AuditAlarm{},"audit_alarms","idx_audit_alarms_created_at"},
		{&model.LoginLog{},"login_logs","idx_login_logs_created_at"},
		{&model.AuditOperation{},"audit_operations","idx_audit_operations_created_at"},
	}{
		if db.Migrator().HasIndex(entry.model,entry.name){continue}
		if err:=db.Exec("CREATE INDEX "+entry.name+" ON "+entry.table+" (created_at)").Error;err!=nil{
			return fmt.Errorf("create audit retention index %s: %w",entry.name,err)
		}
	}
	return nil
}
