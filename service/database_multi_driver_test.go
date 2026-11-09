package service

import (
	"os"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type legacyDevicePolicyForDrivers struct {
	Id uint `gorm:"primaryKey"`
	Name string
	Status int
	IdWhitelist string
	IpWhitelist string
	EnableWebRTC bool
	TrustDurationDays int
	Revision int64
}
func (legacyDevicePolicyForDrivers) TableName() string { return "device_policies" }

type legacyAuditConnForDrivers struct {
	Id uint `gorm:"primaryKey"`
	PeerId string
	FromPeer string
	ConnId int64
}
func (legacyAuditConnForDrivers) TableName() string { return "audit_conns" }

func TestDatabaseMigrationAcrossDrivers(t *testing.T) {
	cases := []struct {
		name string
		open func() (*gorm.DB,error)
	}{
		{"sqlite",func()(*gorm.DB,error){
			return gorm.Open(sqlite.Open("file:drivers_upgrade?mode=memory&cache=shared"),&gorm.Config{})
		}},
	}
	if dsn:=os.Getenv("RUSTDESK_TEST_MYSQL_DSN");dsn!="" {
		cases=append(cases,struct{name string;open func()(*gorm.DB,error)}{
			"mysql",func()(*gorm.DB,error){return gorm.Open(mysql.Open(dsn),&gorm.Config{})},
		})
	}
	if dsn:=os.Getenv("RUSTDESK_TEST_POSTGRES_DSN");dsn!="" {
		cases=append(cases,struct{name string;open func()(*gorm.DB,error)}{
			"postgres",func()(*gorm.DB,error){return gorm.Open(postgres.Open(dsn),&gorm.Config{})},
		})
	}
	for _,tc:=range cases{
		t.Run(tc.name,func(t *testing.T){
			db,err:=tc.open()
			if err!=nil{t.Fatal(err)}
			connection,err:=db.DB()
			if err!=nil{t.Fatal(err)}
			defer connection.Close()
			connection.SetMaxOpenConns(1)
			if err:=db.AutoMigrate(&legacyDevicePolicyForDrivers{},&legacyAuditConnForDrivers{});err!=nil{
				t.Fatalf("old schema preparation failed: %v",err)
			}
			priorPolicy:=&legacyDevicePolicyForDrivers{
				Name:"Legacy policy",Status:1,IdWhitelist:"desk-one",IpWhitelist:"192.0.2.0/24",TrustDurationDays:30,
			}
			if err:=db.Create(priorPolicy).Error;err!=nil{t.Fatal(err)}
			if err:=db.Create(&legacyAuditConnForDrivers{Id:5,PeerId:"target",FromPeer:"source",ConnId:42}).Error;err!=nil{
				t.Fatal(err)
			}
			if err:=db.AutoMigrate(&model.DevicePolicy{},&model.AuditConn{},
				&model.NotificationSetting{},&model.NotificationDelivery{},
				&model.ClientBuildProfile{},&model.ClientBuildTask{},
			);err!=nil{t.Fatalf("new schema migration failed: %v",err)}
			var policy model.DevicePolicy
			if err:=db.Where("id = ?",priorPolicy.Id).First(&policy).Error;err!=nil{t.Fatal(err)}
			if policy.Name!="Legacy policy"||policy.IdWhitelist!="desk-one"||
				policy.IpWhitelist!="192.0.2.0/24"||policy.TrustDurationDays!=30||
				policy.ManagedClientKeys!=""||policy.AccessMode!=""{
				t.Fatalf("historical policy changed: %+v",policy)
			}
			var audit model.AuditConn
			if err:=db.Where("id = ?",5).First(&audit).Error;err!=nil{t.Fatal(err)}
			if audit.PeerId!="target"||audit.FromPeer!="source"||audit.ConnId!=42{
				t.Fatalf("historical audit changed: %+v",audit)
			}
			if err:=db.Model(&model.DevicePolicy{}).Where("id = ?",priorPolicy.Id).
				Updates(map[string]interface{}{"access_mode":"view","managed_client_keys":"access-mode"}).Error;err!=nil{
				t.Fatal(err)
			}
			if err:=db.Where("id = ?",priorPolicy.Id).First(&policy).Error;err!=nil||
				policy.AccessMode!="view"||policy.ManagedClientKeys!="access-mode"{
				t.Fatalf("extended security fields are unavailable: %+v %v",policy,err)
			}
			for _,table:=range []interface{}{
				&model.NotificationSetting{},&model.NotificationDelivery{},
				&model.ClientBuildProfile{},&model.ClientBuildTask{},
			}{
				if !db.Migrator().HasTable(table){t.Fatalf("database migration missing table %T",table)}
			}
			profile:=&model.ClientBuildProfile{
				Name:"Support office",Platform:"windows-x64",Version:"1.5.1",
				Host:"desk.example.test",PublicKey:"test-only-public-key",
			}
			if err:=db.Create(profile).Error;err!=nil{t.Fatal(err)}
			task:=&model.ClientBuildTask{
				ProfileID:profile.Id,RequestedBy:3,Status:"pending",
				Platform:profile.Platform,Version:profile.Version,
				ConfigSnapshot:`{"host":"desk.example.test"}`,Filename:"test-client.exe",
			}
			if err:=db.Create(task).Error;err!=nil{t.Fatal(err)}
			var archived model.ClientBuildTask
			if err:=db.Where("id = ?",task.Id).First(&archived).Error;err!=nil||
				archived.ConfigSnapshot!=task.ConfigSnapshot||archived.Filename!=task.Filename {
				t.Fatalf("client build history was not persisted: %+v %v",archived,err)
			}
			if err:=db.AutoMigrate(&model.DevicePolicy{},&model.AuditConn{},
				&model.NotificationSetting{},&model.NotificationDelivery{},
				&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{
				t.Fatalf("repeated migrations are not safe: %v",err)
			}
			var count int64
			if err:=db.Model(&model.AuditConn{}).Count(&count).Error;err!=nil||count!=1 {
				t.Fatalf("repeated migration changed historical audit: %d %v",count,err)
			}
		})
	}
}
