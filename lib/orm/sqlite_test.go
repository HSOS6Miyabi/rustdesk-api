package orm

import (
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
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
