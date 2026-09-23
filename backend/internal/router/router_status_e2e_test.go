package router_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	"github.com/cygreenenv/greenhouse-panel/internal/router"
	"github.com/cygreenenv/greenhouse-panel/internal/service"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func setup(t *testing.T) (*gin.Engine, *gorm.DB, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:e2e_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	gh := repository.NewGreenhouseRepository(db)
	sr := repository.NewSensorRepository(db)
	ar := repository.NewAlertRepository(db)
	hub := ws.NewHub()
	auth := service.NewAuthService("test-secret")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mon := service.NewMonitoringService(gh, sr, ar, logger, hub)
	alertsSvc := service.NewAlertService(ar, logger)
	control := service.NewControlService(repository.NewDeviceRepository(db), logger, hub)
	reports := service.NewReportService(sr, ar)
	r := router.New(router.Dependencies{Logger: logger, Auth: auth, Monitoring: mon, Alerts: alertsSvc, Control: control, Reports: reports, Hub: hub})
	// login
	body := `{"username":"admin","password":"admin123"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	return r, db, resp.Data.Token
}

func TestOnlineOfflineFlow(t *testing.T) {
	r, db, token := setup(t)
	g := model.Greenhouse{Name: "E2E 温室", Location: "X", Area: 10}
	if err := db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: constants.StatusOffline}
	db.Create(&sensor)
	db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 10, MaxValue: 30})

	do := func(method, path, body string, auth bool) apiResp {
		var reader *bytes.Buffer
		if body != "" {
			reader = bytes.NewBufferString(body)
		} else {
			reader = &bytes.Buffer{}
		}
		req, _ := http.NewRequest(method, path, reader)
		if auth {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		var resp apiResp
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp
	}

	// 从未上报：详情里传感器离线
	resp := do(http.MethodGet, "/api/v1/greenhouses/1", "", false)
	var detail model.Greenhouse
	json.Unmarshal(resp.Data, &detail)
	if len(detail.Sensors) != 1 || detail.Sensors[0].Status != constants.StatusOffline {
		t.Fatalf("new sensor should be offline: %s", resp.Data)
	}

	// 上报一次（阈值内）：恢复在线，且不产生报警
	resp = do(http.MethodPost, "/api/v1/readings", `{"sensorId":1,"value":22}`, true)
	if resp.Code != 0 {
		t.Fatalf("ingest failed: %s", resp.Message)
	}
	resp = do(http.MethodGet, "/api/v1/readings/latest?greenhouse_id=1", "", false)
	var latest []model.SensorReading
	json.Unmarshal(resp.Data, &latest)
	if len(latest) != 1 || latest[0].Sensor.Status != constants.StatusOnline || latest[0].Sensor.LastReportedAt == nil {
		t.Fatalf("sensor should be online after ingest: %s", resp.Data)
	}

	// 超过 5 分钟没有上报：再次查询为离线
	stale := time.Now().Add(-constants.OfflineThreshold - time.Minute)
	db.Model(&model.Sensor{}).Where("id = ?", sensor.ID).Update("last_reported_at", stale)
	resp = do(http.MethodGet, "/api/v1/greenhouses/1", "", false)
	json.Unmarshal(resp.Data, &detail)
	if detail.Sensors[0].Status != constants.StatusOffline {
		t.Fatalf("sensor should be offline after stale: %s", resp.Data)
	}
	resp = do(http.MethodGet, "/api/v1/readings/latest?greenhouse_id=1", "", false)
	json.Unmarshal(resp.Data, &latest)
	if latest[0].Sensor.Status != constants.StatusOffline {
		t.Fatalf("latest sensor should be resolved offline: %s", resp.Data)
	}
	alertsBefore := do(http.MethodGet, "/api/v1/alerts?greenhouse_id=1", "", false)
	var alerts []model.Alert
	json.Unmarshal(alertsBefore.Data, &alerts)
	if len(alerts) != 0 {
		t.Fatalf("stale status must not create alerts, got %d", len(alerts))
	}

	// 重新上报立即恢复在线；超阈值依旧触发报警
	resp = do(http.MethodPost, "/api/v1/readings", `{"sensorId":1,"value":40}`, true)
	if resp.Code != 0 {
		t.Fatalf("recover ingest failed: %s", resp.Message)
	}
	resp = do(http.MethodGet, "/api/v1/greenhouses/1", "", false)
	json.Unmarshal(resp.Data, &detail)
	if detail.Sensors[0].Status != constants.StatusOnline {
		t.Fatalf("sensor should recover online: %s", resp.Data)
	}
	alertsAfter := do(http.MethodGet, "/api/v1/alerts?greenhouse_id=1", "", false)
	json.Unmarshal(alertsAfter.Data, &alerts)
	if len(alerts) != 1 {
		t.Fatalf("threshold alert should still work, got %d alerts", len(alerts))
	}
}
