package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHeartbeatDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:heartbeat_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestIngestRefreshesHeartbeatAndRestoresOnline(t *testing.T) {
	db := newHeartbeatDB(t)
	g := model.Greenhouse{Name: "心跳温室"}
	db.Create(&g)
	stale := time.Now().Add(-10 * time.Minute)
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: constants.StatusOffline, LastReportedAt: &stale}
	db.Create(&sensor)
	db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 10, MaxValue: 30})

	svc := NewMonitoringService(repository.NewGreenhouseRepository(db), repository.NewSensorRepository(db), repository.NewAlertRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)), ws.NewHub())
	reading, alert, err := svc.Ingest(sensor.ID, 22)
	if err != nil {
		t.Fatal(err)
	}
	if alert != nil {
		t.Fatal("正常读数不应产生报警")
	}
	reloaded, err := repository.NewSensorRepository(db).Get(sensor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != constants.StatusOnline || reloaded.LastReportedAt == nil || !reloaded.LastReportedAt.Equal(reading.RecordedAt) {
		t.Fatalf("上报后应立即恢复在线并记录心跳时间，实际 status=%s last=%v", reloaded.Status, reloaded.LastReportedAt)
	}
}

func TestSensorOfflineAfterFiveMinutes(t *testing.T) {
	now := time.Now()
	onlineAt := now.Add(-4 * time.Minute)
	offlineAt := now.Add(-(constants.SensorOfflineThreshold + time.Minute))

	online := model.Sensor{Status: constants.StatusOnline, LastReportedAt: &onlineAt}
	if got := online.Online(now); got != constants.StatusOnline {
		t.Fatalf("4 分钟前上报应在线，得到 %s", got)
	}
	stale := model.Sensor{Status: constants.StatusOnline, LastReportedAt: &offlineAt}
	if got := stale.Online(now); got != constants.StatusOffline {
		t.Fatalf("超过 5 分钟未上报应离线，得到 %s", got)
	}
	never := model.Sensor{Status: constants.StatusOnline}
	if got := never.Online(now); got != constants.StatusOffline {
		t.Fatalf("从未上报应离线，得到 %s", got)
	}
}

func TestThresholdAlertStillFiresOnIngest(t *testing.T) {
	db := newHeartbeatDB(t)
	g := model.Greenhouse{Name: "报警温室"}
	db.Create(&g)
	sensor := model.Sensor{GreenhouseID: g.ID, Name: "湿度", Type: "humidity", Unit: "%", Status: constants.StatusOnline}
	db.Create(&sensor)
	db.Create(&model.Threshold{SensorID: sensor.ID, MinValue: 40, MaxValue: 80})

	svc := NewMonitoringService(repository.NewGreenhouseRepository(db), repository.NewSensorRepository(db), repository.NewAlertRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)), ws.NewHub())
	if _, alert, err := svc.Ingest(sensor.ID, 90); err != nil || alert == nil {
		t.Fatalf("超阈值读数应照常报警，alert=%v err=%v", alert, err)
	}
}
