//go:build runtime

package podbridge5

import (
	"context"
	"errors"
	"testing"
)

// resetRuntimeSingleton clears the package-level runtime singleton so each
// subtest starts from a known, uninitialized state regardless of what ran
// before it in the same test binary.
func resetRuntimeSingleton(t *testing.T) {
	t.Helper()
	if err := Shutdown(); err != nil && !errors.Is(err, ErrRuntimeNotInitialized) {
		t.Fatalf("Shutdown() during reset failed: %v", err)
	}
}

func TestInit(t *testing.T) {
	resetRuntimeSingleton(t)
	t.Cleanup(func() { resetRuntimeSingleton(t) })

	if err := Init(); err != nil {
		t.Fatalf("Init() failed: %v", err)
	}

	// Init() is idempotent once the singleton is initialized.
	if err := Init(); err != nil {
		t.Fatalf("second Init() call failed: %v", err)
	}
}

func TestInitWithContext(t *testing.T) {
	resetRuntimeSingleton(t)
	t.Cleanup(func() { resetRuntimeSingleton(t) })

	ctx, err := InitWithContext(context.Background())
	if err != nil {
		t.Fatalf("InitWithContext() failed: %v", err)
	}
	if ctx == nil {
		t.Fatal("InitWithContext() returned nil context")
	}

	// Calling it again on an already-initialized singleton returns the same
	// stored context rather than reconnecting.
	ctx2, err := InitWithContext(context.Background())
	if err != nil {
		t.Fatalf("second InitWithContext() call failed: %v", err)
	}
	if ctx2 != ctx {
		t.Fatal("expected second InitWithContext() to return the cached context")
	}
}

func TestReexecIfNeeded(t *testing.T) {
	// Under a normal `go test` binary (no buildah reexec wrapper argv0
	// markers), this should not trigger a reexec and must simply return
	// false without panicking.
	if got := ReexecIfNeeded(); got {
		t.Fatalf("ReexecIfNeeded() = true, want false in a normal test process")
	}
}
