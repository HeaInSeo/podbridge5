package podbridge5

import (
	"testing"
	"time"
)

func TestDefaultHealthcheckPolicy(t *testing.T) {
	interval, retries, timeout, startPeriod := DefaultHealthcheckPolicy()
	if interval != "30s" {
		t.Fatalf("unexpected interval: %q", interval)
	}
	if retries != 3 {
		t.Fatalf("unexpected retries: %d", retries)
	}
	if timeout != "5s" {
		t.Fatalf("unexpected timeout: %q", timeout)
	}
	if startPeriod != "0s" {
		t.Fatalf("unexpected start period: %q", startPeriod)
	}
}

func TestParsePolicyDuration(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		min       time.Duration
		allowZero bool
		want      time.Duration
		wantErr   bool
	}{
		{name: "valid", fieldName: "field", value: "5s", min: time.Second, want: 5 * time.Second},
		{name: "zero allowed", fieldName: "field", value: "0", min: time.Second, allowZero: true, want: 0},
		{name: "below minimum", fieldName: "field", value: "500ms", min: time.Second, wantErr: true},
		{name: "invalid", fieldName: "field", value: "abc", min: time.Second, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePolicyDuration(tc.fieldName, tc.value, tc.min, tc.allowZero)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected duration: got %s want %s", got, tc.want)
			}
		})
	}
}

func TestHealthcheckDurationHelpers(t *testing.T) {
	interval, err := parseHealthcheckInterval("disable")
	if err != nil {
		t.Fatalf("unexpected interval error: %v", err)
	}
	if interval != 0 {
		t.Fatalf("unexpected interval: %s", interval)
	}

	timeout, err := parseHealthcheckTimeout("5s")
	if err != nil {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if timeout != 5*time.Second {
		t.Fatalf("unexpected timeout: %s", timeout)
	}

	if _, err := parseHealthcheckTimeout("500ms"); err == nil {
		t.Fatal("expected timeout validation error")
	}

	startPeriod, err := parseHealthcheckStartPeriod("0s")
	if err != nil {
		t.Fatalf("unexpected start period error: %v", err)
	}
	if startPeriod != 0 {
		t.Fatalf("unexpected start period: %s", startPeriod)
	}

	if _, err := parseHealthcheckStartPeriod("-1s"); err == nil {
		t.Fatal("expected start period validation error")
	}
}
