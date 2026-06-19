package podbridge5

import (
	"context"
	"strings"
	"testing"
)

func TestNewConnection5RejectsEmptyIPCName(t *testing.T) {
	if _, err := NewConnection5(context.Background(), " "); err == nil {
		t.Fatal("expected empty ipc name error")
	}
}

func TestCurrentRuntimeConnectionTarget(t *testing.T) {
	got := currentRuntimeConnectionTarget()
	if strings.TrimSpace(got.URI) == "" {
		t.Fatalf("runtime connection URI must not be empty: %#v", got)
	}
}

func TestCreateInitContainerRejectsInvalidOverlayArgsBeforeStart(t *testing.T) {
	if _, err := CreateInitContainer(context.Background(), "pod-id", "/lower", "/bad,path", "/work", "/target"); err == nil {
		t.Fatal("expected invalid upperdir error")
	}
	if _, err := CreateInitContainer(context.Background(), "pod-id", "/lower", "/upper", "/work", " "); err == nil {
		t.Fatal("expected invalid target error")
	}
}

func TestBuildConfigCreateImageRejectsUninitializedRuntime(t *testing.T) {
	initMu.Lock()
	oldCtx := pbCtx
	oldStore := pbStore
	oldInitialized := initialized
	pbCtx = nil
	pbStore = nil
	initialized = false
	initMu.Unlock()
	t.Cleanup(func() {
		initMu.Lock()
		pbCtx = oldCtx
		pbStore = oldStore
		initialized = oldInitialized
		initMu.Unlock()
	})

	_, _, err := NewConfig("alpine").CreateImage()
	if err == nil {
		t.Fatal("expected uninitialized runtime error")
	}
	if !strings.Contains(err.Error(), "pbCtx is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}
