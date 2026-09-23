package service

import (
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestIngestCreatesAlertBeyondThreshold(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:service_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "测试温室"}
	db.Create(&g)
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "online"}
	db.Create(&sensor)
	db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 10, MaxValue: 30})
	svc := NewMonitoringService(repository.NewGreenhouseRepository(db), repository.NewSensorRepository(db), repository.NewAlertRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)), ws.NewHub())
	_, alert, err := svc.Ingest(sensor.ID, 35)
	if err != nil {
		t.Fatal(err)
	}
	if alert == nil {
		t.Fatal("expected alert for out-of-range reading")
	}
	alerts, err := repository.NewAlertRepository(db).List(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 1 {
		t.Fatalf("want 1 alert, got %d", len(alerts))
	}
}

func newMonitoringServiceDB(t *testing.T) (*gorm.DB, *MonitoringService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:online_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	svc := NewMonitoringService(repository.NewGreenhouseRepository(db), repository.NewSensorRepository(db), repository.NewAlertRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)), ws.NewHub())
	return db, svc
}

func TestIngestRecoversOfflineSensor(t *testing.T) {
	db, svc := newMonitoringServiceDB(t)
	g := model.Greenhouse{Name: "在线状态温室"}
	db.Create(&g)
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "湿度", Type: "humidity", Unit: "%", Status: constants.StatusOffline}
	db.Create(&sensor)
	db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 40, MaxValue: 80})
	stale := time.Now().Add(-constants.OfflineThreshold - time.Minute)
	db.Model(&model.Sensor{}).Where("id = ?", sensor.ID).Update("last_reported_at", stale)

	if _, _, err := svc.Ingest(sensor.ID, 55); err != nil {
		t.Fatal(err)
	}
	updated, err := repository.NewSensorRepository(db).Get(sensor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != constants.StatusOnline || updated.LastReportedAt == nil {
		t.Fatalf("sensor should be online with lastReportedAt after ingest, got status=%q last=%v", updated.Status, updated.LastReportedAt)
	}
	if !updated.IsOnline(time.Now()) {
		t.Fatal("IsOnline should be true immediately after ingest")
	}
}

func TestDetailMarksStaleSensorsOffline(t *testing.T) {
	db, svc := newMonitoringServiceDB(t)
	g := model.Greenhouse{Name: "拔线温室"}
	db.Create(&g)
	online := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: constants.StatusOnline}
	offline := model.Sensor{GreenhouseID: g.ID, Name: "湿度", Type: "humidity", Unit: "%", Status: constants.StatusOnline}
	db.Create(&online)
	db.Create(&offline)
	fresh := time.Now().Add(-time.Minute)
	stale := time.Now().Add(-constants.OfflineThreshold - time.Minute)
	db.Model(&model.Sensor{}).Where("id = ?", online.ID).Update("last_reported_at", fresh)
	db.Model(&model.Sensor{}).Where("id = ?", offline.ID).Update("last_reported_at", stale)

	detail, err := svc.Detail(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]string{}
	for _, row := range detail.Sensors {
		status[row.Name] = row.Status
	}
	if status["温度"] != constants.StatusOnline {
		t.Fatalf("fresh sensor should be online, got %q", status["温度"])
	}
	if status["湿度"] != constants.StatusOffline {
		t.Fatalf("stale sensor should be offline, got %q", status["湿度"])
	}
}
