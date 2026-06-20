//go:build runtime

package podbridge5

import (
	"context"
	"os"
	"testing"

	"go.podman.io/podman/v6/pkg/bindings/images"
)

func TestNewStoreRuntime(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})
}

func TestNewBuilderWithCapabilitiesRuntime(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})

	builder, err := NewBuilder(context.Background(), store, WithFromImage("docker.io/library/alpine:latest"), WithCapabilities())
	if err != nil {
		t.Fatalf("NewBuilder() with WithCapabilities() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := builder.Delete(); err != nil {
			t.Logf("warning: delete builder: %v", err)
		}
	})

	if len(builder.Capabilities) == 0 {
		t.Fatal("expected non-empty capabilities on the builder")
	}
}

func TestNewBuilderPrivateRuntime(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})

	builder, err := newBuilder(context.Background(), store, "docker.io/library/alpine:latest")
	if err != nil {
		t.Fatalf("newBuilder() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := builder.Delete(); err != nil {
			t.Logf("warning: delete builder: %v", err)
		}
	})
}

func TestSaveImageAndExportImageRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	const imageRef = "docker.io/library/alpine:latest"
	if _, err := images.Pull(ctx, imageRef, &images.PullOptions{}); err != nil {
		t.Fatalf("images.Pull() failed: %v", err)
	}

	tmpDir := t.TempDir()
	if err := saveImage(ctx, tmpDir, "alpine", imageRef, false); err != nil {
		t.Fatalf("saveImage() failed: %v", err)
	}

	archivePath := imageArchivePath(tmpDir, "alpine", false)
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatalf("expected archive file at %s: %v", archivePath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty archive at %s", archivePath)
	}
}

func TestBuildImageFromDockerfileRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	id, err := buildImageFromDockerfile(ctx, "./Dockerfile")
	if err != nil {
		t.Fatalf("buildImageFromDockerfile() failed: %v", err)
	}
	if id == "" {
		t.Fatal("buildImageFromDockerfile() returned empty image ID")
	}
}

func TestBuildDockerfileContentRealAdapterRuntime(t *testing.T) {
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore() failed: %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(store, false); err != nil {
			t.Logf("warning: shutdown store: %v", err)
		}
	})

	const dockerfileContent = "FROM docker.io/library/alpine:latest\nCMD [\"true\"]\n"
	imageID, _, err := BuildDockerfileContent(context.Background(), store, dockerfileContent, "localhost/podbridge5-test-builddockerfilecontent:latest")
	if err != nil {
		t.Fatalf("BuildDockerfileContent() failed: %v", err)
	}
	if imageID == "" {
		t.Fatal("BuildDockerfileContent() returned empty image ID")
	}
}
