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
	StatusOff       = "off"
	StatusOn        = "on"
	AlertPending    = "pending"
	AlertHandled    = "handled"
	RoleAdmin       = "admin"
	SuccessMessage  = "ok"
	EventAlert      = "alert.created"
	EventDevice     = "device.updated"
)

// SensorOfflineThreshold 传感器超过该时长未上报即视为离线。
const SensorOfflineThreshold = 5 * time.Minute
