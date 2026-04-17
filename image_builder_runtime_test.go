package podbridge5

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/containers/buildah"
	"github.com/containers/storage"
)

type fakeImageBuilder struct {
	runCommands [][]string
	addCalls    []fakeAddCall
	runErr      error
	addErr      error
}

type fakeAddCall struct {
	dest string
	src  []string
}

func (f *fakeImageBuilder) Run(command []string, _ buildah.RunOptions) error {
	f.runCommands = append(f.runCommands, append([]string(nil), command...))
	return f.runErr
}

func (f *fakeImageBuilder) Add(dest string, _ bool, _ buildah.AddAndCopyOptions, src ...string) error {
	f.addCalls = append(f.addCalls, fakeAddCall{dest: dest, src: append([]string(nil), src...)})
	return f.addErr
}

type fakeImageBuilderFactoryRuntime struct {
	caps       []string
	capsErr    error
	newBuilder *buildah.Builder
	newErr     error
	options    *buildah.BuilderOptions
}

func (f *fakeImageBuilderFactoryRuntime) Capabilities() ([]string, error) {
	return append([]string(nil), f.caps...), f.capsErr
}

func (f *fakeImageBuilderFactoryRuntime) NewBuilder(_ context.Context, _ storage.Store, options buildah.BuilderOptions) (*buildah.Builder, error) {
	copy := options
	f.options = &copy
	return f.newBuilder, f.newErr
}

func TestCreateDirectoriesWithRuntime(t *testing.T) {
	builder := &fakeImageBuilder{}
	if err := createDirectoriesWithRuntime(builder, []string{"/app", "/data"}); err != nil {
		t.Fatalf("createDirectoriesWithRuntime() error = %v", err)
	}
	want := [][]string{{"mkdir", "-p", "/app"}, {"mkdir", "-p", "/data"}}
	if !reflect.DeepEqual(builder.runCommands, want) {
		t.Fatalf("run commands = %#v, want %#v", builder.runCommands, want)
	}
}

func TestSetFilePermissionsWithRuntime(t *testing.T) {
	builder := &fakeImageBuilder{}
	if err := setFilePermissionsWithRuntime(builder, []string{"/app/executor.sh", "/app/healthcheck.sh"}); err != nil {
		t.Fatalf("setFilePermissionsWithRuntime() error = %v", err)
	}
	want := [][]string{{"chmod", "777", "/app/executor.sh", "/app/healthcheck.sh"}}
	if !reflect.DeepEqual(builder.runCommands, want) {
		t.Fatalf("run commands = %#v, want %#v", builder.runCommands, want)
	}
}

func TestInstallDependenciesWithRuntime(t *testing.T) {
	builder := &fakeImageBuilder{}
	if err := installDependenciesWithRuntime(builder); err != nil {
		t.Fatalf("installDependenciesWithRuntime() error = %v", err)
	}
	want := [][]string{{ContainerInstallPath}}
	if !reflect.DeepEqual(builder.runCommands, want) {
		t.Fatalf("run commands = %#v, want %#v", builder.runCommands, want)
	}
}

func TestCopyScriptsWithRuntime(t *testing.T) {
	builder := &fakeImageBuilder{}
	options := newDefaultAddAndCopyOptions(nil)
	scripts := map[string][]string{"/app": {"executor.sh", "healthcheck.sh"}, "/data": {"seed.txt"}}
	if err := copyScriptsWithRuntime(builder, options, scripts); err != nil {
		t.Fatalf("copyScriptsWithRuntime() error = %v", err)
	}
	if len(builder.addCalls) != 3 {
		t.Fatalf("add calls = %d, want 3", len(builder.addCalls))
	}
}

func TestNewBuilderWithRuntime(t *testing.T) {
	runtime := &fakeImageBuilderFactoryRuntime{caps: []string{"CAP_SYS_ADMIN"}, newBuilder: &buildah.Builder{}}
	builder, err := newBuilderWithRuntime(context.Background(), runtime, nil, "docker.io/library/alpine:latest")
	if err != nil {
		t.Fatalf("newBuilderWithRuntime() error = %v", err)
	}
	if builder == nil {
		t.Fatal("expected builder")
	}
	if runtime.options == nil {
		t.Fatal("expected builder options to be captured")
	}
	if runtime.options.FromImage != "docker.io/library/alpine:latest" {
		t.Fatalf("FromImage = %q", runtime.options.FromImage)
	}
	if !reflect.DeepEqual(runtime.options.Capabilities, []string{"CAP_SYS_ADMIN"}) {
		t.Fatalf("Capabilities = %#v", runtime.options.Capabilities)
	}
}

func TestNewBuilderWithRuntime_PropagatesCapabilityError(t *testing.T) {
	runtime := &fakeImageBuilderFactoryRuntime{capsErr: errors.New("caps failed")}
	_, err := newBuilderWithRuntime(context.Background(), runtime, nil, "docker.io/library/alpine:latest")
	if err == nil || err.Error() != "failed to get capabilities: caps failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDirectoriesWithRuntime_PropagatesRunError(t *testing.T) {
	builder := &fakeImageBuilder{runErr: errors.New("run failed")}
	err := createDirectoriesWithRuntime(builder, []string{"/app"})
	if err == nil || err.Error() != "failed to create directory /app: run failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyScriptsWithRuntime_PropagatesAddError(t *testing.T) {
	builder := &fakeImageBuilder{addErr: errors.New("add failed")}
	err := copyScriptsWithRuntime(builder, newDefaultAddAndCopyOptions(nil), map[string][]string{"/app": {"executor.sh"}})
	want := fmt.Sprintf("failed to copy script %s to %s: %s", "executor.sh", "/app", "add failed")
	if err == nil || err.Error() != want {
		t.Fatalf("unexpected error: %v", err)
	}
}
