package model

import (
	"testing"
	"time"

	"github.com/cygreenenv/greenhouse-panel/internal/constants"
)

func TestSensorResolveStatus(t *testing.T) {
	now := time.Now()
	fresh := now.Add(-constants.OfflineThreshold + time.Minute)
	stale := now.Add(-constants.OfflineThreshold - time.Minute)

	cases := []struct {
		name   string
		sensor Sensor
		online bool
	}{
		{"从未上报为离线", Sensor{LastReportedAt: nil}, false},
		{"五分钟内上报为在线", Sensor{LastReportedAt: &fresh}, true},
		{"超过五分钟未上报为离线", Sensor{LastReportedAt: &stale}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sensor := tc.sensor
			if got := sensor.IsOnline(now); got != tc.online {
				t.Fatalf("IsOnline = %v, want %v", got, tc.online)
			}
			sensor.ResolveStatus(now)
			want := constants.StatusOffline
			if tc.online {
				want = constants.StatusOnline
			}
			if sensor.Status != want {
				t.Fatalf("Status = %q, want %q", sensor.Status, want)
			}
		})
	}
}
