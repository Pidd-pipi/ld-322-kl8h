package constants

import "time"

const (
	APIPrefix       = "/api/v1"
	HealthPath      = "/healthz"
	WebSocketPath   = "/ws"
	DefaultPage     = 1
	DefaultPageSize = 100
	MaxPageSize     = 500
	StatusOnline    = "online"
	StatusOffline   = "offline"
	// OfflineThreshold 传感器超过该时长没有上报即判定为离线
	OfflineThreshold = 5 * time.Minute
	StatusOff        = "off"
	StatusOn         = "on"
	AlertPending     = "pending"
	AlertHandled     = "handled"
	RoleAdmin        = "admin"
	SuccessMessage   = "ok"
	EventAlert       = "alert.created"
	EventDevice      = "device.updated"
	EventSensor      = "sensor.updated"
)
