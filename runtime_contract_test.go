package podbridge5

import (
	"reflect"
	"testing"
	"time"

	"github.com/containers/image/v5/manifest"
)

func TestDefaultExecutorCommand(t *testing.T) {
	got := DefaultExecutorCommand()
	want := []string{"/bin/sh", "-c", ContainerExecutorPath}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command: got %v want %v", got, want)
	}
}

func TestParseHealthcheckConfig(t *testing.T) {
	tests := []struct {
		name        string
		inCmd       string
		interval    string
		retries     uint
		timeout     string
		startPeriod string
		expectErr   bool
		expected    *manifest.Schema2HealthConfig
	}{
		{
			name:        "valid healthcheck with default contract",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "30s",
			retries:     3,
			timeout:     "5s",
			startPeriod: "0s",
			expected: &manifest.Schema2HealthConfig{
				Test:        []string{"CMD-SHELL", ContainerHealthcheckPath},
				Interval:    30 * time.Second,
				Retries:     3,
				Timeout:     5 * time.Second,
				StartPeriod: 0,
			},
		},
		{
			name:        "disabled interval",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "disable",
			retries:     2,
			timeout:     "10s",
			startPeriod: "5s",
			expected: &manifest.Schema2HealthConfig{
				Test:        []string{"CMD-SHELL", ContainerHealthcheckPath},
				Interval:    0,
				Retries:     2,
				Timeout:     10 * time.Second,
				StartPeriod: 5 * time.Second,
			},
		},
		{
			name:        "invalid command",
			inCmd:       ContainerHealthcheckPath,
			interval:    "30s",
			retries:     3,
			timeout:     "5s",
			startPeriod: "0s",
			expectErr:   true,
		},
		{
			name:        "invalid interval",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "abc",
			retries:     3,
			timeout:     "5s",
			startPeriod: "0s",
			expectErr:   true,
		},
		{
			name:        "invalid timeout",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "30s",
			retries:     3,
			timeout:     "500ms",
			startPeriod: "0s",
			expectErr:   true,
		},
		{
			name:        "negative start period",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "30s",
			retries:     3,
			timeout:     "5s",
			startPeriod: "-1s",
			expectErr:   true,
		},
		{
			name:        "zero retries",
			inCmd:       DefaultHealthcheckCommand(),
			interval:    "30s",
			retries:     0,
			timeout:     "5s",
			startPeriod: "0s",
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseHealthcheckConfig(tc.inCmd, tc.interval, tc.retries, tc.timeout, tc.startPeriod)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("mismatch: got %+v want %+v", got, tc.expected)
			}
		})
	}
}
