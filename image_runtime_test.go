package podbridge5

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/containers/buildah/define"
	"github.com/containers/storage"
)

type fakeImageBuildRuntime struct {
	buildID         string
	buildDigest     string
	buildErr        error
	buildDockerfile string
	buildOutput     string
	pushDigest      string
	pushErr         error
	pushImageRef    string
	pushDestination string
}

func (f *fakeImageBuildRuntime) BuildDockerfiles(_ context.Context, _ storage.Store, options define.BuildOptions, dockerfilePath string) (string, string, error) {
	f.buildDockerfile = dockerfilePath
	f.buildOutput = options.Output
	return f.buildID, f.buildDigest, f.buildErr
}

func (f *fakeImageBuildRuntime) PushImage(_ context.Context, _ storage.Store, imageRef, normalizedDestination string) (string, error) {
	f.pushImageRef = imageRef
	f.pushDestination = normalizedDestination
	return f.pushDigest, f.pushErr
}

func TestWriteDockerfileTempFile(t *testing.T) {
	path, cleanup, err := writeDockerfileTempFile("FROM scratch\n")
	if err != nil {
		t.Fatalf("writeDockerfileTempFile() error = %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != "FROM scratch\n" {
		t.Fatalf("temp Dockerfile content = %q", string(data))
	}
}

func TestBuildDockerfileContentWithRuntime(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildID: "img-1", buildDigest: "sha256:abc"}
	imageID, digestStr, err := buildDockerfileContentWithRuntime(context.Background(), runtime, nil, "FROM scratch\n", "example.com/demo:latest")
	if err != nil {
		t.Fatalf("buildDockerfileContentWithRuntime() error = %v", err)
	}
	if imageID != "img-1" || digestStr != "sha256:abc" {
		t.Fatalf("unexpected build result: %q %q", imageID, digestStr)
	}
	if runtime.buildOutput != "example.com/demo:latest" {
		t.Fatalf("build output = %q", runtime.buildOutput)
	}
	if runtime.buildDockerfile == "" {
		t.Fatal("expected temp Dockerfile path to be passed")
	}
}

func TestPushImageWithRuntime(t *testing.T) {
	runtime := &fakeImageBuildRuntime{pushDigest: "sha256:def"}
	digestStr, err := pushImageWithRuntime(context.Background(), runtime, nil, "example.com/demo:latest", "example.com/demo:latest")
	if err != nil {
		t.Fatalf("pushImageWithRuntime() error = %v", err)
	}
	if digestStr != "sha256:def" {
		t.Fatalf("pushImageWithRuntime() = %q", digestStr)
	}
	if runtime.pushDestination != "docker://example.com/demo:latest" {
		t.Fatalf("normalized destination = %q", runtime.pushDestination)
	}
}

func TestBuildAndPushDockerfileContentWithRuntime(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildID: "img-2", buildDigest: "sha256:build", pushDigest: "sha256:push"}
	imageID, digestStr, err := buildAndPushDockerfileContentWithRuntime(context.Background(), runtime, nil, "FROM scratch\n", "example.com/demo:latest")
	if err != nil {
		t.Fatalf("buildAndPushDockerfileContentWithRuntime() error = %v", err)
	}
	if imageID != "img-2" || digestStr != "sha256:push" {
		t.Fatalf("unexpected build+push result: %q %q", imageID, digestStr)
	}
}

func TestBuildDockerfileContentWithRuntime_PropagatesBuildError(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildErr: errors.New("build failed")}
	_, _, err := buildDockerfileContentWithRuntime(context.Background(), runtime, nil, "FROM scratch\n", "example.com/demo:latest")
	if err == nil || err.Error() != "imagebuildah.BuildDockerfiles: build failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPushImageWithRuntime_PropagatesPushError(t *testing.T) {
	runtime := &fakeImageBuildRuntime{pushErr: errors.New("push failed")}
	_, err := pushImageWithRuntime(context.Background(), runtime, nil, "example.com/demo:latest", "example.com/demo:latest")
	if err == nil || err.Error() != "push image \"example.com/demo:latest\" to \"example.com/demo:latest\": push failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
