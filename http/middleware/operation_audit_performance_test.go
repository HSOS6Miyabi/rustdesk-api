package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGeoIPReadOnlyPostDoesNotWriteOperationAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:operation_audit_readonly?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.AuditOperation{}); err != nil { t.Fatal(err) }
	oldDB := service.DB
	service.DB = db
	defer func() { service.DB = oldDB }()
	user := &model.User{Username:"operator"}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set("curUser", user); user.Id=1; c.Next() }, RecordOperations())
	router.POST("/api/admin/geoip/lookup", func(c *gin.Context) { c.Status(200) })
	router.POST("/api/admin/peer/update", func(c *gin.Context) { c.Status(200) })
	for _, path := range []string{"/api/admin/geoip/lookup", "/api/admin/peer/update"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusOK { t.Fatalf("%s: %d", path, response.Code) }
	}
	var rows []model.AuditOperation
	if err := db.Find(&rows).Error; err != nil { t.Fatal(err) }
	if len(rows) != 1 || rows[0].Path != "/api/admin/peer/update" {
		t.Fatalf("read-only operation produced audit history: %+v", rows)
	}
}
