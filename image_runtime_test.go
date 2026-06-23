package podbridge5

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"go.podman.io/buildah/define"
	imageTypes "go.podman.io/image/v5/types"
	"go.podman.io/storage"
)

type fakeImageBuildRuntime struct {
	buildID         string
	buildDigest     string
	buildErr        error
	buildDockerfile string
	buildOutput     string
	buildOptions    define.BuildOptions
	pushDigest      string
	pushErr         error
	pushImageRef    string
	pushDestination string
	pushCertDir     string
	pushSecureTLS   bool
}

func (f *fakeImageBuildRuntime) BuildDockerfiles(_ context.Context, _ storage.Store, options define.BuildOptions, dockerfilePath string) (buildID, buildRef string, buildErr error) {
	f.buildDockerfile = dockerfilePath
	f.buildOutput = options.Output
	f.buildOptions = options
	return f.buildID, f.buildDigest, f.buildErr
}

func (f *fakeImageBuildRuntime) PushImage(_ context.Context, _ storage.Store, imageRef, normalizedDestination string, sysCtx *imageTypes.SystemContext) (string, error) {
	f.pushImageRef = imageRef
	f.pushDestination = normalizedDestination
	if sysCtx != nil {
		f.pushCertDir = sysCtx.DockerPerHostCertDirPath
	}
	f.pushSecureTLS = true
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

func TestBuildDockerfileContentWithOptionsRuntime(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildID: "img-1", buildDigest: "sha256:abc"}
	buildOpts := DefaultImageBuildOptions("example.com/demo:latest")
	buildOpts.ContextDirectory = "/workspace"
	buildOpts.Isolation = define.IsolationChroot
	buildOpts.Runtime = "crun"
	buildOpts.Layers = true

	imageID, digestStr, err := buildDockerfileContentWithOptionsRuntime(context.Background(), runtime, nil, "FROM scratch\n", buildOpts)
	if err != nil {
		t.Fatalf("buildDockerfileContentWithOptionsRuntime() error = %v", err)
	}
	if imageID != "img-1" || digestStr != "sha256:abc" {
		t.Fatalf("unexpected build result: %q %q", imageID, digestStr)
	}
	if runtime.buildOutput != "example.com/demo:latest" {
		t.Fatalf("build output = %q", runtime.buildOutput)
	}
}

func TestBuildDockerfileContentWithOptionsRuntimeRejectsNilContext(t *testing.T) {
	_, _, err := buildDockerfileContentWithOptionsRuntime(nil, &fakeImageBuildRuntime{}, nil, "FROM scratch\n", DefaultImageBuildOptions("example.com/demo:latest"))
	if err == nil || err.Error() != "ctx must not be nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDockerfileContentWithOptionsRuntimeRejectsEmptyDockerfile(t *testing.T) {
	_, _, err := buildDockerfileContentWithOptionsRuntime(context.Background(), &fakeImageBuildRuntime{}, nil, "  \n\t", DefaultImageBuildOptions("example.com/demo:latest"))
	if err == nil || err.Error() != "dockerfileContent must not be empty" {
		t.Fatalf("unexpected error: %v", err)
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

func TestPushImageWithSystemContextRuntimeRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		ctx         context.Context
		imageRef    string
		destination string
		want        string
	}{
		{name: "nil context", ctx: nil, imageRef: "example.com/demo:latest", destination: "example.com/demo:latest", want: "ctx must not be nil"},
		{name: "empty image", ctx: context.Background(), imageRef: " ", destination: "example.com/demo:latest", want: "imageRef must not be empty"},
		{name: "empty destination", ctx: context.Background(), imageRef: "example.com/demo:latest", destination: " ", want: "destination must not be empty"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pushImageWithSystemContextRuntime(tc.ctx, &fakeImageBuildRuntime{}, nil, tc.imageRef, tc.destination, nil)
			if err == nil || err.Error() != tc.want {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func TestBuildAndPushDockerfileContentWithRuntimePropagatesPushError(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildID: "img-2", pushErr: errors.New("push failed")}
	_, _, err := buildAndPushDockerfileContentWithRuntime(context.Background(), runtime, nil, "FROM scratch\n", "example.com/demo:latest")
	if err == nil || !strings.Contains(err.Error(), "push failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAndPushUserNamespaceWithRuntime(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildID: "img-userns", buildDigest: "sha256:build", pushDigest: "sha256:push"}
	var factoryConfig UserNamespaceBuildConfig
	var shutdownCalled bool

	config := UserNamespaceBuildConfig{
		OutputRef:           "example.com/demo:latest",
		ContextDirectory:    "/workspace",
		Runtime:             "crun",
		Isolation:           BuildIsolationOCI,
		StorageMode:         StorageFuseOverlay,
		RegistryCertDirPath: "/etc/containers/certs.d",
	}
	imageID, digestStr, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		runtime,
		func(config UserNamespaceBuildConfig) (storage.Store, error) {
			factoryConfig = config
			return nil, nil
		},
		func(_ storage.Store, force bool) error {
			if force {
				t.Fatal("user namespace build should not force storage shutdown")
			}
			shutdownCalled = true
			return nil
		},
		config,
		"FROM scratch\n",
	)
	if err != nil {
		t.Fatalf("buildAndPushUserNamespaceWithRuntime() error = %v", err)
	}
	if imageID != "img-userns" || digestStr != "sha256:push" {
		t.Fatalf("unexpected result: %q %q", imageID, digestStr)
	}
	if factoryConfig.StorageMode != StorageFuseOverlay {
		t.Fatalf("store factory config storage mode = %q", factoryConfig.StorageMode)
	}
	if runtime.buildOptions.Output != "example.com/demo:latest" {
		t.Fatalf("build output = %q", runtime.buildOptions.Output)
	}
	if runtime.buildOptions.ContextDirectory != "/workspace" {
		t.Fatalf("context directory = %q", runtime.buildOptions.ContextDirectory)
	}
	if runtime.buildOptions.Isolation != define.IsolationOCI {
		t.Fatalf("isolation = %v", runtime.buildOptions.Isolation)
	}
	if runtime.buildOptions.Runtime != "crun" {
		t.Fatalf("runtime = %q", runtime.buildOptions.Runtime)
	}
	if runtime.buildOptions.SystemContext == nil || runtime.buildOptions.SystemContext.DockerPerHostCertDirPath != "/etc/containers/certs.d" {
		t.Fatalf("build cert dir = %#v", runtime.buildOptions.SystemContext)
	}
	if !runtime.buildOptions.Layers {
		t.Fatal("expected layers to be enabled")
	}
	if runtime.pushImageRef != "example.com/demo:latest" || runtime.pushDestination != "docker://example.com/demo:latest" {
		t.Fatalf("unexpected push args: image=%q destination=%q", runtime.pushImageRef, runtime.pushDestination)
	}
	if runtime.pushCertDir != "/etc/containers/certs.d" {
		t.Fatalf("push cert dir = %q", runtime.pushCertDir)
	}
	if !shutdownCalled {
		t.Fatal("expected storage shutdown")
	}
}

func TestBuildAndPushUserNamespaceWithRuntimePropagatesStoreFactoryError(t *testing.T) {
	_, _, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		&fakeImageBuildRuntime{},
		func(UserNamespaceBuildConfig) (storage.Store, error) {
			return nil, errors.New("store failed")
		},
		func(storage.Store, bool) error {
			t.Fatal("shutdown should not be called")
			return nil
		},
		UserNamespaceBuildConfig{OutputRef: "example.com/demo:latest", StorageMode: StorageVFS},
		"FROM scratch\n",
	)
	if err == nil || err.Error() != "store failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAndPushUserNamespaceWithRuntimePropagatesBuildOptionErrorAndShutdown(t *testing.T) {
	var shutdownCalled bool
	_, _, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		&fakeImageBuildRuntime{},
		func(UserNamespaceBuildConfig) (storage.Store, error) {
			return nil, nil
		},
		func(storage.Store, bool) error {
			shutdownCalled = true
			return nil
		},
		UserNamespaceBuildConfig{OutputRef: "example.com/demo:latest", StorageMode: StorageVFS, Isolation: "bad-isolation"},
		"FROM scratch\n",
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported build isolation") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shutdownCalled {
		t.Fatal("expected shutdown after store creation")
	}
}

func TestBuildAndPushUserNamespaceWithRuntimePropagatesPushErrorAndShutdown(t *testing.T) {
	var shutdownCalled bool
	_, _, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		&fakeImageBuildRuntime{buildID: "img-userns", pushErr: errors.New("push failed")},
		func(UserNamespaceBuildConfig) (storage.Store, error) {
			return nil, nil
		},
		func(storage.Store, bool) error {
			shutdownCalled = true
			return nil
		},
		UserNamespaceBuildConfig{OutputRef: "example.com/demo:latest", StorageMode: StorageVFS},
		"FROM scratch\n",
	)
	if err == nil || !strings.Contains(err.Error(), "push failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !shutdownCalled {
		t.Fatal("expected shutdown after push failure")
	}
}

func TestBuildAndPushUserNamespaceWithRuntimePropagatesShutdownError(t *testing.T) {
	_, _, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		&fakeImageBuildRuntime{buildID: "img-userns", pushDigest: "sha256:push"},
		func(UserNamespaceBuildConfig) (storage.Store, error) {
			return nil, nil
		},
		func(storage.Store, bool) error {
			return errors.New("shutdown failed")
		},
		UserNamespaceBuildConfig{OutputRef: "example.com/demo:latest", StorageMode: StorageVFS},
		"FROM scratch\n",
	)
	if err == nil || err.Error() != "shutdown failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAndPushUserNamespaceWithRuntime_RequiresOutputRef(t *testing.T) {
	_, _, err := buildAndPushUserNamespaceWithRuntime(
		context.Background(),
		&fakeImageBuildRuntime{},
		func(UserNamespaceBuildConfig) (storage.Store, error) {
			t.Fatal("store factory should not be called")
			return nil, nil
		},
		func(storage.Store, bool) error {
			t.Fatal("shutdown should not be called")
			return nil
		},
		UserNamespaceBuildConfig{StorageMode: StorageVFS},
		"FROM scratch\n",
	)
	if err == nil || err.Error() != "output ref must not be empty" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildDockerfileContentWithRuntime_PropagatesBuildError(t *testing.T) {
	runtime := &fakeImageBuildRuntime{buildErr: errors.New("build failed")}
	_, _, err := buildDockerfileContentWithRuntime(context.Background(), runtime, nil, "FROM scratch\n", "example.com/demo:latest")
	if err == nil || err.Error() != "imagebuildah.BuildDockerfiles: build failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnnotateBuildahExecutionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "unshare", err: errors.New("unshare(CLONE_NEWUSER): operation not permitted"), want: "user namespace"},
		{name: "setgroups", err: errors.New("write setgroups: permission denied"), want: "supplementary groups"},
		{name: "setcap", err: errors.New("setcap cap_net_bind_service=+ep /usr/bin/tool failed"), want: "CAP_SETFCAP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := annotateBuildahExecutionError(tc.err)
			if got == nil || !strings.Contains(got.Error(), tc.want) {
				t.Fatalf("annotation = %v, want hint containing %q", got, tc.want)
			}
		})
	}
}

func TestClassifyBuildahExecutionError(t *testing.T) {
	tests := []struct {
		err  error
		want BuildahExecutionErrorKind
	}{
		{err: nil, want: BuildahExecutionErrorUnknown},
		{err: errors.New("unshare(CLONE_NEWUSER): operation not permitted"), want: BuildahExecutionErrorUserNamespace},
		{err: errors.New("write setgroups: permission denied"), want: BuildahExecutionErrorSupplementaryGroup},
		{err: errors.New("set file capabilities: operation not permitted"), want: BuildahExecutionErrorFileCapabilities},
		{err: errors.New("unrelated failure"), want: BuildahExecutionErrorUnknown},
	}
	for _, tc := range tests {
		if got := ClassifyBuildahExecutionError(tc.err); got != tc.want {
			t.Fatalf("ClassifyBuildahExecutionError(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestPushImageWithRuntime_PropagatesPushError(t *testing.T) {
	runtime := &fakeImageBuildRuntime{pushErr: errors.New("push failed")}
	_, err := pushImageWithRuntime(context.Background(), runtime, nil, "example.com/demo:latest", "example.com/demo:latest")
	if err == nil || err.Error() != "push image \"example.com/demo:latest\" to \"example.com/demo:latest\": push failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
