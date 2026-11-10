package orm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSQLiteUsesWALAndBoundsConnections(t *testing.T){
	filename:=filepath.Join(t.TempDir(),"test.db")
	db:=NewSqlite(&SqliteConfig{Path:filename,MaxOpenConns:100,MaxIdleConns:10},logrus.New())
	connection,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	defer connection.Close()
	if connection.Stats().MaxOpenConnections!=4{
		t.Fatalf("SQLite allowed %d connections",connection.Stats().MaxOpenConnections)
	}
	var journal string
	if err:=db.Raw("PRAGMA journal_mode").Scan(&journal).Error;err!=nil{t.Fatal(err)}
	if journal!="wal"{t.Fatalf("journal mode is %q",journal)}
}

func TestSQLiteClosedDatabaseCanBeCopiedAndRestored(t *testing.T) {
	dir:=t.TempDir()
	source:=filepath.Join(dir,"source.db")
	db:=NewSqlite(&SqliteConfig{Path:source,MaxOpenConns:4,MaxIdleConns:4},logrus.New())
	if err:=db.Exec("CREATE TABLE checkpoint_test (value TEXT)").Error;err!=nil{t.Fatal(err)}
	if err:=db.Exec("INSERT INTO checkpoint_test (value) VALUES (?)","persisted").Error;err!=nil{t.Fatal(err)}
	connection,err:=db.DB()
	if err!=nil{t.Fatal(err)}
	if err:=connection.Close();err!=nil{t.Fatal(err)}
	bytes,err:=os.ReadFile(source)
	if err!=nil{t.Fatal(err)}
	backup:=filepath.Join(dir,"backup.db")
	if err:=os.WriteFile(backup,bytes,0600);err!=nil{t.Fatal(err)}
	restored,err:=gorm.Open(sqlite.Open(backup),&gorm.Config{})
	if err!=nil{t.Fatal(err)}
	restoredConnection,err:=restored.DB()
	if err!=nil{t.Fatal(err)}
	defer restoredConnection.Close()
	var count int64
	if err:=restored.Raw("SELECT COUNT(*) FROM checkpoint_test WHERE value = ?","persisted").
		Scan(&count).Error;err!=nil{t.Fatal(err)}
	if count!=1 {t.Fatalf("closed SQLite database did not retain committed changes: %d",count)}
}
