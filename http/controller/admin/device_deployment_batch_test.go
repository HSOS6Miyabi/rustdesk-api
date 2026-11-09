package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeploymentBatchStatusIsAtomic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:deployment_batch_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil { t.Fatal(err) }
	conn, err := db.DB()
	if err != nil { t.Fatal(err) }
	defer conn.Close()
	conn.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.DeviceDeployment{}); err != nil { t.Fatal(err) }
	oldDB := service.DB
	service.DB = db
	defer func() { service.DB = oldDB }()
	records := []*model.DeviceDeployment{
		{DeviceId: "deployment-a", Uuid: "uuid-a", OwnerUserId: 1, PublicKey: "public-a", Status: "approved"},
		{DeviceId: "deployment-b", Uuid: "uuid-b", OwnerUserId: 1, PublicKey: "public-b", Status: "approved"},
		{DeviceId: "deployment-c", Uuid: "uuid-c", OwnerUserId: 2, PublicKey: "public-c", Status: "approved"},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil { t.Fatal(err) }
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/deployment/batch", (&DeviceDeployment{}).SetBatchStatus)
	send := func(ids []uint, status string) int {
		t.Helper()
		body, err := json.Marshal(gin.H{"ids": ids, "status": status})
		if err != nil { t.Fatal(err) }
		request := httptest.NewRequest(http.MethodPost, "/deployment/batch", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var envelope struct{ Code int `json:"code"` }
		if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("invalid response: %v %s", err, response.Body.String())
		}
		return envelope.Code
	}
	getStatus := func(record *model.DeviceDeployment) string {
		t.Helper()
		var fresh model.DeviceDeployment
		if err := db.First(&fresh, record.Id).Error; err != nil { t.Fatal(err) }
		return fresh.Status
	}
	if code := send([]uint{records[0].Id, 99999}, "revoked"); code == 0 {
		t.Fatal("batch updated non-existing deployment")
	}
	if getStatus(records[0]) != "approved" {
		t.Fatal("invalid batch partially updated a valid deployment")
	}
	if code := send([]uint{records[0].Id, records[1].Id}, "revoked"); code != 0 {
		t.Fatalf("valid batch rejected: %d", code)
	}
	if getStatus(records[0]) != "revoked" || getStatus(records[1]) != "revoked" ||
		getStatus(records[2]) != "approved" {
		t.Fatal("batch affected the wrong deployments")
	}
	if code := send([]uint{records[0].Id, records[0].Id}, "approved"); code != 0 ||
		getStatus(records[0]) != "approved" {
		t.Fatal("duplicate IDs did not update a single deployment correctly")
	}
	if code := send([]uint{records[2].Id}, "unknown"); code == 0 ||
		getStatus(records[2]) != "approved" {
		t.Fatal("invalid status changed a deployment")
	}
}
