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
	LastReportedAt *time.Time `gorm:"index" json:"lastReportedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	Threshold      Threshold  `json:"threshold,omitempty"`
}

// IsOnline 根据最近一次上报时间判断在线状态：从未上报或超过离线阈值未上报均为离线。
func (s Sensor) IsOnline(now time.Time) bool {
	return s.LastReportedAt != nil && now.Sub(*s.LastReportedAt) <= constants.OfflineThreshold
}

// ResolveStatus 依据最近上报时间刷新内存中的状态字段，保证返回给前端的状态与时效一致。
func (s *Sensor) ResolveStatus(now time.Time) {
	if s.IsOnline(now) {
		s.Status = constants.StatusOnline
	} else {
		s.Status = constants.StatusOffline
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
