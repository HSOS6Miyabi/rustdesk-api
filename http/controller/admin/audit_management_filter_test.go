package admin

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuditExportAppliesDeviceDateAndActionFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:audit_export_filter_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	sqlDB, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.AuditConn{}); err != nil { t.Fatal(err) }
	previous := service.DB
	service.DB = db
	defer func(){service.DB = previous}()
	oldDate := time.Date(2025, 6, 12, 12, 0, 0, 0, time.UTC)
	newDate := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	a := &model.AuditConn{PeerId: "desk-a", FromPeer: "source-a", Action: "close", ConnId: 1, FromUserId: 11}
	b := &model.AuditConn{PeerId: "desk-b", FromPeer: "source-b", Action: "new", ConnId: 2, FromUserId: 12}
	for _, record := range []*model.AuditConn{a, b} {
		if err := db.Create(record).Error; err != nil { t.Fatal(err) }
	}
	if err := db.Model(&model.AuditConn{}).Where("id = ?", a.Id).Update("created_at", oldDate).Error; err != nil { t.Fatal(err) }
	if err := db.Model(&model.AuditConn{}).Where("id = ?", b.Id).Update("created_at", newDate).Error; err != nil { t.Fatal(err) }
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/audit/export", (&AuditManagement{}).Export)
	collect := func(query string) (int, [][]string) {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet,"/audit/export?kind=audit_conn"+query,nil))
		if response.Code != http.StatusOK { return response.Code,nil }
		rows, err := csv.NewReader(strings.NewReader(response.Body.String())).ReadAll()
		if err != nil { t.Fatal(err) }
		return response.Code,rows
	}
	for _, tc := range []struct {query string; id string; count int}{
		{"&peer_id=desk-a", "desk-a",2},
		{"&from_peer=source-b","desk-b",2},
		{"&from_user_id=11","desk-a",2},
		{"&action=close","desk-a",2},
		{"&from_date=2026-01-01&to_date=2026-12-31","desk-b",2},
		{"&from_date=2026-11-02","",1},
	} {
		status,rows:=collect(tc.query)
		if status!=http.StatusOK || len(rows)!=tc.count {
			t.Fatalf("%s produced %d rows (%d status): %+v",tc.query,len(rows),status,rows)
		}
		if tc.id!=""&&rows[1][1]!=tc.id {
			t.Fatalf("export filtered wrong device for %s: %+v",tc.query,rows)
		}
	}
	for _, q := range []string{"&from_date=2026-99-99","&from_user_id=unknown","&action=unknown"}{
		response:=httptest.NewRecorder()
		router.ServeHTTP(response,httptest.NewRequest(http.MethodGet,"/audit/export?kind=audit_conn"+q,nil))
		if !strings.Contains(response.Body.String(),"\"code\":101") {
			t.Fatalf("invalid audit export filter accepted: %s",response.Body.String())
		}
	}
}
