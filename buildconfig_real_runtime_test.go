//go:build runtime

package podbridge5

import (
	"context"
	"testing"
)

func TestBuildConfigCreateImageWithDockerfileRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})

	cfg := NewConfig("docker.io/library/ubuntu:latest")
	cfg.Image.ImageSavePath = t.TempDir()

	builder, imageID, err := cfg.CreateImageWithDockerfile(ctx, store)
	if builder != nil {
		t.Cleanup(func() {
			if err := builder.Delete(); err != nil {
				t.Logf("warning: delete builder: %v", err)
			}
		})
	}
	if err != nil {
		t.Fatalf("CreateImageWithDockerfile() failed: %v", err)
	}
	if imageID == "" {
		t.Fatal("CreateImageWithDockerfile() returned empty image ID")
	}
}

func TestContainerConfigSetupContainerRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})

	builder, err := newBuilder(ctx, store, "docker.io/library/ubuntu:latest")
	if err != nil {
		t.Fatalf("newBuilder() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := builder.Delete(); err != nil {
			t.Logf("warning: delete builder: %v", err)
		}
	})

	cfg := NewConfig("docker.io/library/ubuntu:latest")
	if err := cfg.Container.SetupContainer(builder); err != nil {
		t.Fatalf("SetupContainer() failed: %v", err)
	}
}
