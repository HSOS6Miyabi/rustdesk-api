package admin

import (
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeviceExportFiltersOwnerAndAvoidsSpreadsheetFormulas(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:device_export_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	sqlDB, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.Peer{}); err != nil { t.Fatal(err) }
	old := service.DB
	service.DB = db
	defer func() { service.DB = old }()
	for _, peer := range []*model.Peer{
		{Id: "desk-support-1", UserId: 7, Hostname: "Support", Note: "=HYPERLINK(1)"},
		{Id: "desk-private-1", UserId: 9, Hostname: "Private"},
		{Id: "desk-support-2", UserId: 7, Hostname: "Support extra"},
	} {
		if err := db.Create(peer).Error; err != nil { t.Fatal(err) }
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("adminRoleScope", service.AdminRoleAccess{UserIDs: []uint{7}})
		c.Next()
	})
	router.GET("/peer/export", (&Peer{}).ExportCSV)
	collect := func(query string) [][]string {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/peer/export"+query, nil)
		writer := httptest.NewRecorder()
		router.ServeHTTP(writer, request)
		if writer.Code != http.StatusOK { t.Fatalf("unexpected export response: %d %s", writer.Code, writer.Body.String()) }
		if writer.Header().Get("Cache-Control") != "no-store" { t.Fatal("CSV export cache headers missing") }
		rows, err := csv.NewReader(strings.NewReader(writer.Body.String())).ReadAll()
		if err != nil { t.Fatal(err) }
		return rows
	}
	rows := collect("")
	if len(rows) != 3 { t.Fatalf("scoped export included private devices: %+v", rows) }
	if rows[1][1] != "desk-support-2" || rows[2][11] != "'=HYPERLINK(1)" {
		t.Fatalf("unexpected scoped export rows: %+v", rows)
	}
	rows = collect("?search=desk-support-1")
	if len(rows) != 2 || rows[1][1] != "desk-support-1" {
		t.Fatalf("search filter selected unexpected devices: %+v", rows)
	}
	rows = collect("?user_id=9")
	if len(rows) != 1 { t.Fatalf("ownership restriction was bypassed: %+v", rows) }
}
