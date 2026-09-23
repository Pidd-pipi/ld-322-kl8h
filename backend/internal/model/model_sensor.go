package model

import (
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
)

type Sensor struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	GreenhouseID   uint       `gorm:"index" json:"greenhouseId"`
	Name           string     `gorm:"type:varchar(100)" json:"name"`
	Type           string     `gorm:"type:varchar(50);index" json:"type"`
	Unit           string     `gorm:"type:varchar(32)" json:"unit"`
	Status         string     `gorm:"type:varchar(32)" json:"status"`
	LastReportedAt *time.Time `gorm:"index" json:"lastReportedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	Threshold      Threshold  `json:"threshold,omitempty"`
}

// RefreshStatus 依据最近一次上报时间推导在线状态：
// 五分钟内有上报为在线，否则离线；从未上报同样视为离线。
func (s *Sensor) RefreshStatus(now time.Time) {
	s.Status = s.Online(now)
}

// Online 返回给定时刻传感器应处的在线状态。
func (s *Sensor) Online(now time.Time) string {
	if s.LastReportedAt == nil || now.Sub(*s.LastReportedAt) > constants.SensorOfflineThreshold {
		return constants.StatusOffline
	}
	return constants.StatusOnline
}

// RefreshSensorsStatus 批量刷新传感器在线状态，便于在查询出口统一调用。
func RefreshSensorsStatus(sensors []Sensor, now time.Time) {
	for i := range sensors {
		sensors[i].RefreshStatus(now)
	}
}

type SensorReading struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SensorID   uint      `gorm:"index" json:"sensorId"`
	Value      float64   `json:"value"`
	RecordedAt time.Time `gorm:"index" json:"recordedAt"`
	Sensor     Sensor    `json:"sensor,omitempty"`
}
type Threshold struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SensorID  uint      `gorm:"uniqueIndex" json:"sensorId"`
	MinValue  float64   `json:"minValue"`
	MaxValue  float64   `json:"maxValue"`
	UpdatedAt time.Time `json:"updatedAt"`
}
