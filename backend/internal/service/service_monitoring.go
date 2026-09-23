package service

import (
	"fmt"
	"github.com/cygreenenv/greenhouse-panel/internal/constants"
	"github.com/cygreenenv/greenhouse-panel/internal/model"
	"github.com/cygreenenv/greenhouse-panel/internal/repository"
	ws "github.com/cygreenenv/greenhouse-panel/internal/websocket"
	"log/slog"
	"math/rand"
	"time"
)

type MonitoringService struct {
	greenhouseRepo *repository.GreenhouseRepository
	sensorRepo     *repository.SensorRepository
	alertRepo      *repository.AlertRepository
	logger         *slog.Logger
	hub            *ws.Hub
}

func NewMonitoringService(g *repository.GreenhouseRepository, s *repository.SensorRepository, a *repository.AlertRepository, l *slog.Logger, h *ws.Hub) *MonitoringService {
	return &MonitoringService{g, s, a, l, h}
}

// resolveGreenhouseStatus 依据每个传感器最近上报时间刷新在线状态，并返回离线传感器数量。
func resolveGreenhouseStatus(g *model.Greenhouse) int {
	offline := 0
	now := time.Now()
	for i := range g.Sensors {
		g.Sensors[i].ResolveStatus(now)
		if g.Sensors[i].Status == constants.StatusOffline {
			offline++
		}
	}
	return offline
}
func (s *MonitoringService) ListGreenhouses() ([]model.Greenhouse, error) {
	rows, err := s.greenhouseRepo.List()
	if err != nil {
		return nil, err
	}
	for i := range rows {
		resolveGreenhouseStatus(&rows[i])
	}
	return rows, nil
}
func (s *MonitoringService) Detail(id uint) (*model.Greenhouse, error) {
	g, err := s.greenhouseRepo.Get(id)
	if err != nil {
		return nil, err
	}
	resolveGreenhouseStatus(g)
	return g, nil
}
func (s *MonitoringService) Ingest(sensorID uint, value float64) (*model.SensorReading, *model.Alert, error) {
	sensor, err := s.sensorRepo.Get(sensorID)
	if err != nil {
		return nil, nil, err
	}
	wasOffline := !sensor.IsOnline(time.Now())
	reading := &model.SensorReading{SensorID: sensorID, Value: value, RecordedAt: time.Now()}
	if err = s.sensorRepo.AddReading(reading); err != nil {
		return nil, nil, err
	}
	sensor.LastReportedAt = &reading.RecordedAt
	sensor.ResolveStatus(reading.RecordedAt)
	// 重新上报立即恢复在线，并通知前端刷新状态
	if wasOffline {
		s.hub.Broadcast(constants.EventSensor, sensor)
	}
	var alert *model.Alert
	if value < sensor.Threshold.MinValue || value > sensor.Threshold.MaxValue {
		level := "warning"
		if value < sensor.Threshold.MinValue*.8 || value > sensor.Threshold.MaxValue*1.2 {
			level = "critical"
		}
		alert = &model.Alert{GreenhouseID: sensor.GreenhouseID, SensorID: sensor.ID, Level: level, Message: fmt.Sprintf("%s 当前值 %.2f%s 超出阈值 [%.2f, %.2f]", constants.SensorLabels[sensor.Type], value, sensor.Unit, sensor.Threshold.MinValue, sensor.Threshold.MaxValue), Value: value, Status: constants.AlertPending}
		if err = s.alertRepo.Create(alert); err != nil {
			return nil, nil, err
		}
		s.hub.Broadcast(constants.EventAlert, alert)
	}
	s.hub.Broadcast("reading.created", reading)
	return reading, alert, nil
}
func (s *MonitoringService) History(greenhouseID uint, types []string, start, end time.Time) ([]model.SensorReading, error) {
	return s.sensorRepo.History(greenhouseID, types, start, end)
}
func (s *MonitoringService) Latest(greenhouseID uint) ([]model.SensorReading, error) {
	rows, err := s.sensorRepo.LatestForGreenhouse(greenhouseID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	for i := range rows {
		rows[i].Sensor.ResolveStatus(now)
	}
	return rows, nil
}
func (s *MonitoringService) UpdateThreshold(sensorID uint, min, max float64) (*model.Threshold, error) {
	return s.sensorRepo.UpdateThreshold(sensorID, min, max)
}
func (s *MonitoringService) Simulate(greenhouseID uint) (int, error) {
	g, err := s.greenhouseRepo.Get(greenhouseID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, sensor := range g.Sensors {
		base := (sensor.Threshold.MinValue + sensor.Threshold.MaxValue) / 2
		span := (sensor.Threshold.MaxValue - sensor.Threshold.MinValue) / 3
		value := base + (rand.Float64()-.5)*span
		if _, _, err = s.Ingest(sensor.ID, value); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
func (s *MonitoringService) CreateGreenhouse(row *model.Greenhouse) error {
	return s.greenhouseRepo.Create(row)
}
func (s *MonitoringService) CreateSensor(greenhouseID uint, name, sensorType string, min, max float64) (*model.Sensor, error) {
	sensor := &model.Sensor{GreenhouseID: greenhouseID, Name: name, Type: sensorType, Unit: constants.SensorUnits[sensorType], Status: constants.StatusOffline}
	threshold := &model.Threshold{MinValue: min, MaxValue: max}
	if err := s.sensorRepo.Create(sensor, threshold); err != nil {
		return nil, err
	}
	sensor.Threshold = *threshold
	return sensor, nil
}
