package podbridge5

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVolumeRemoveErrorWrapsCause(t *testing.T) {
	cause := errors.New("busy")
	err := &VolumeRemoveError{Name: "data", Cause: cause}

	if !errors.Is(err, cause) {
		t.Fatalf("expected errors.Is to match wrapped cause")
	}
	if got := err.Error(); got != "remove volume data: busy" {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestIsNotFoundErr(t *testing.T) {
	for _, err := range []error{
		errors.New("volume not found"),
		errors.New("server returned 404"),
	} {
		if !isNotFoundErr(err) {
			t.Fatalf("expected not-found error for %v", err)
		}
	}
	if isNotFoundErr(nil) {
		t.Fatal("nil must not be treated as not-found")
	}
	if isNotFoundErr(errors.New("permission denied")) {
		t.Fatal("permission error must not be treated as not-found")
	}
}

func TestWithRetryReturnsContextCancellationDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := withRetry(ctx, 3, time.Hour, func() error {
		calls++
		cancel()
		return errors.New("first failure")
	})
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !strings.Contains(err.Error(), "retry:") || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected wrapped context cancellation, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one attempt before cancellation, got %d", calls)
	}
}

func TestWithNamedVolumeValidationAndIdempotency(t *testing.T) {
	spec, err := NewSpec(WithNamedVolume(" data ", "/data", "sub", "rw"))
	if err != nil {
		t.Fatalf("WithNamedVolume returned error: %v", err)
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("expected one volume, got %#v", spec.Volumes)
	}
	if spec.Volumes[0].Name != "data" || spec.Volumes[0].Dest != "/data" || spec.Volumes[0].SubPath != "sub" {
		t.Fatalf("volume mismatch: %#v", spec.Volumes[0])
	}

	if err := WithNamedVolume("data", "/data", "ignored")(spec); err != nil {
		t.Fatalf("idempotent same volume returned error: %v", err)
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("idempotent call appended duplicate volume: %#v", spec.Volumes)
	}

	if err := WithNamedVolume("other", "/data", "")(spec); err == nil {
		t.Fatal("expected conflicting destination error")
	}
	if err := WithNamedVolume(" ", "/empty", "")(spec); err == nil {
		t.Fatal("expected empty volume name error")
	}
}
