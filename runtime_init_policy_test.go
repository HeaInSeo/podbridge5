package podbridge5

import (
	"errors"
	"os"
	"testing"
)

func TestResolveRuntimeConnectionTargetUsesContainerHost(t *testing.T) {
	target := resolveRuntimeConnectionTarget(1001, " unix:///tmp/podman.sock ", "/run/user/1001", true)
	if target.URI != "unix:///tmp/podman.sock" {
		t.Fatalf("unexpected uri: %q", target.URI)
	}
	if target.Source != "CONTAINER_HOST" {
		t.Fatalf("unexpected source: %q", target.Source)
	}
	if target.RuntimeDir != "" {
		t.Fatalf("expected empty runtime dir for CONTAINER_HOST override, got %q", target.RuntimeDir)
	}
}

func TestResolveRuntimeConnectionTargetUsesXDGRuntimeDir(t *testing.T) {
	target := resolveRuntimeConnectionTarget(1001, "", "/tmp/runtime-dir", true)
	if target.URI != "unix:/tmp/runtime-dir/podman/podman.sock" {
		t.Fatalf("unexpected uri: %q", target.URI)
	}
	if target.Source != "XDG_RUNTIME_DIR" {
		t.Fatalf("unexpected source: %q", target.Source)
	}
	if target.RuntimeDir != "/tmp/runtime-dir" {
		t.Fatalf("unexpected runtime dir: %q", target.RuntimeDir)
	}
}

func TestResolveRuntimeConnectionTargetUsesRootlessDefault(t *testing.T) {
	target := resolveRuntimeConnectionTarget(4242, "", "", true)
	if target.URI != "unix:/run/user/4242/podman/podman.sock" {
		t.Fatalf("unexpected uri: %q", target.URI)
	}
	if target.Source != "rootless-default" {
		t.Fatalf("unexpected source: %q", target.Source)
	}
}

func TestResolveRuntimeConnectionTargetUsesRootDefault(t *testing.T) {
	target := resolveRuntimeConnectionTarget(0, "", "", false)
	if target.URI != "unix:/run/podman/podman.sock" {
		t.Fatalf("unexpected uri: %q", target.URI)
	}
	if target.Source != "root-default" {
		t.Fatalf("unexpected source: %q", target.Source)
	}
}

func TestWrapRuntimeErrorsAreClassifiable(t *testing.T) {
	connErr := wrapRuntimeConnectionError(runtimeConnectionTarget{URI: "unix:/run/podman/podman.sock", Source: "root-default"}, os.ErrNotExist)
	if !errors.Is(connErr, ErrRuntimeConnectionUnavailable) {
		t.Fatalf("expected ErrRuntimeConnectionUnavailable")
	}
	if !errors.Is(connErr, os.ErrNotExist) {
		t.Fatalf("expected wrapped os.ErrNotExist")
	}

	storeErr := wrapRuntimeStoreError(os.ErrPermission)
	if !errors.Is(storeErr, ErrRuntimeStoreUnavailable) {
		t.Fatalf("expected ErrRuntimeStoreUnavailable")
	}
	if !errors.Is(storeErr, os.ErrPermission) {
		t.Fatalf("expected wrapped os.ErrPermission")
	}
}

func TestShutdownWithoutInit(t *testing.T) {
	initMu.Lock()
	pbStore = nil
	pbCtx = nil
	initialized = false
	initMu.Unlock()

	if err := Shutdown(); !errors.Is(err, ErrRuntimeNotInitialized) {
		t.Fatalf("expected ErrRuntimeNotInitialized, got %v", err)
	}
}
