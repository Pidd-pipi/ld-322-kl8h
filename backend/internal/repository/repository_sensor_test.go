package repository

import (
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestSensorRepositoryHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:repo_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "测试温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	s := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "online"}
	if err = db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, value := range []float64{21, 22} {
		if err = db.Create(&model.SensorReading{SensorID: s.ID, Value: value, RecordedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := NewSensorRepository(db)
	rows, err := repo.History(g.ID, []string{"temperature"}, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 readings, got %d", len(rows))
	}
}

func TestSensorRepositoryLatestLoadsThreshold(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:latest_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "最新数据温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	s := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "online"}
	if err = db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.Threshold{SensorID: s.ID, MinValue: 15, MaxValue: 32}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.SensorReading{SensorID: s.ID, Value: 24, RecordedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := NewSensorRepository(db).LatestForGreenhouse(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Sensor.Threshold.MinValue != 15 || rows[0].Sensor.Threshold.MaxValue != 32 {
		t.Fatalf("latest reading did not include the expected threshold: %#v", rows)
	}
}

func TestLatestMarksStaleSensorOffline(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:offline_latest_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "离线温室"}
	if err = db.Create(&g).Error; err != nil {
		t.Fatal(err)
	}
	s := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "online"}
	if err = db.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.Threshold{SensorID: s.ID, MinValue: 15, MaxValue: 32}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.SensorReading{SensorID: s.ID, Value: 24, RecordedAt: time.Now().Add(-10 * time.Minute)}).Error; err != nil {
		t.Fatal(err)
	}
	rows, err := NewSensorRepository(db).LatestForGreenhouse(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("旧读数仍应返回，got %d 条", len(rows))
	}
	if rows[0].Sensor.Status != "offline" {
		t.Fatalf("十分钟未上报应推导为离线，实际 %s", rows[0].Sensor.Status)
	}
	detail, err := NewGreenhouseRepository(db).Get(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Sensors[0].Status != "offline" {
		t.Fatalf("温室详情中的传感器应离线，实际 %s", detail.Sensors[0].Status)
	}
}

func TestAddReadingRestoresOnline(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:heartbeat_repo_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(model.All()...); err != nil {
		t.Fatal(err)
	}
	g := model.Greenhouse{Name: "恢复温室"}
	db.Create(&g)
	stale := time.Now().Add(-10 * time.Minute)
	s := model.Sensor{GreenhouseID: g.ID, Name: "温度", Type: "temperature", Unit: "°C", Status: "offline", LastReportedAt: &stale}
	db.Create(&s)
	now := time.Now()
	if err = NewSensorRepository(db).AddReading(&model.SensorReading{SensorID: s.ID, Value: 25, RecordedAt: now}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewSensorRepository(db).Get(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != "online" || reloaded.LastReportedAt == nil || !reloaded.LastReportedAt.Equal(now) {
		t.Fatalf("写入读数后应恢复在线并更新心跳：%#v", reloaded)
	}
}
