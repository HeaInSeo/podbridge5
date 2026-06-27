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
