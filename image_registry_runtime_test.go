//go:build runtime

package podbridge5

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	imageTypes "go.podman.io/image/v5/types"
)

// startTestRegistry starts a local, plain-HTTP `registry:2` container published
// on the host at hostPort and returns its container ID. The caller is
// responsible for removing the container.
func startTestRegistry(t *testing.T, ctx context.Context, hostPort uint16) (containerID string) {
	t.Helper()

	spec, err := NewSpec(
		WithImageName("docker.io/library/registry:2"),
		WithName("test-registry-"+uuid.New().String()),
		WithPortMapping(5000, hostPort),
	)
	if err != nil {
		t.Fatalf("NewSpec() for test registry failed: %v", err)
	}

	id, err := StartContainer(ctx, spec)
	if err != nil {
		t.Fatalf("StartContainer() for test registry failed: %v", err)
	}
	return id
}

// waitForRegistry retries fn until the local registry has finished starting
// up and is accepting connections.
func waitForRegistry(ctx context.Context, fn func() error) error {
	return withRetry(ctx, 10, 500*time.Millisecond, fn)
}

func TestPushImageRealAdapterRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	const hostPort = uint16(15050)
	registryID := startTestRegistry(t, ctx, hostPort)
	t.Cleanup(func() { cleanupContainer(t, ctx, registryID) })

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
	destination := fmt.Sprintf("localhost:%d/podbridge5-test-push:latest", hostPort)
	imageID, _, err := BuildDockerfileContent(context.Background(), store, dockerfileContent, destination)
	if err != nil {
		t.Fatalf("BuildDockerfileContent() failed: %v", err)
	}

	sysCtx := &imageTypes.SystemContext{DockerInsecureSkipTLSVerify: imageTypes.OptionalBoolTrue}
	if err := waitForRegistry(ctx, func() error {
		_, err := pushImageWithSystemContextRuntime(context.Background(), realImageBuildRuntime{}, store, destination, destination, sysCtx)
		return err
	}); err != nil {
		t.Fatalf("pushImageWithSystemContextRuntime() failed: %v", err)
	}
	t.Logf("pushed image %s to %s", imageID, destination)
}

func TestBuildAndPushUserNamespaceRuntime(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	const hostPort = uint16(15051)
	registryID := startTestRegistry(t, ctx, hostPort)
	t.Cleanup(func() { cleanupContainer(t, ctx, registryID) })

	destination := fmt.Sprintf("localhost:%d/podbridge5-test-buildpush-userns:latest", hostPort)
	storageRoot := t.TempDir()
	cfg := UserNamespaceBuildConfig{
		OutputRef:             destination,
		StorageMode:           StorageVFS,
		RunRoot:               storageRoot + "/run",
		GraphRoot:             storageRoot + "/graph",
		InsecureSkipTLSVerify: true,
	}
	const dockerfileContent = "FROM docker.io/library/alpine:latest\nCMD [\"true\"]\n"

	var imageID, digestStr string
	if err := waitForRegistry(context.Background(), func() error {
		var pushErr error
		imageID, digestStr, pushErr = BuildAndPushUserNamespace(context.Background(), cfg, dockerfileContent)
		return pushErr
	}); err != nil {
		t.Fatalf("BuildAndPushUserNamespace() failed: %v", err)
	}
	if imageID == "" {
		t.Fatal("BuildAndPushUserNamespace() returned empty image ID")
	}
	t.Logf("built+pushed image %s digest=%s to %s", imageID, digestStr, destination)
}
