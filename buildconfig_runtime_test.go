package podbridge5

import (
	"errors"
	"reflect"
	"testing"

	"github.com/containers/buildah"
)

type fakeConfiguredBuilder struct {
	fakeImageBuilder
	workDir string
	cmd     []string
}

func (f *fakeConfiguredBuilder) SetWorkDir(workDir string) {
	f.workDir = workDir
}

func (f *fakeConfiguredBuilder) SetCmd(cmd []string) {
	f.cmd = append([]string(nil), cmd...)
}

func TestApplyImageConfigToBuilder(t *testing.T) {
	builder := &fakeConfiguredBuilder{}
	config := ImageConfig{
		Directories:     []string{"/app", "/data"},
		ScriptMap:       map[string][]string{"/app": {"executor.sh"}},
		PermissionFiles: []string{"/app/executor.sh"},
		WorkDir:         "/app",
		CMD:             []string{"/bin/sh", "-c", "/app/executor.sh"},
	}
	if err := applyImageConfigToBuilder(builder, config); err != nil {
		t.Fatalf("applyImageConfigToBuilder() error = %v", err)
	}
	if builder.workDir != "/app" {
		t.Fatalf("workDir = %q", builder.workDir)
	}
	if !reflect.DeepEqual(builder.cmd, config.CMD) {
		t.Fatalf("cmd = %#v, want %#v", builder.cmd, config.CMD)
	}
	if len(builder.runCommands) < 3 {
		t.Fatalf("expected builder run commands, got %#v", builder.runCommands)
	}
	if len(builder.addCalls) != 1 {
		t.Fatalf("add calls = %d, want 1", len(builder.addCalls))
	}
}

func TestApplyContainerConfigToBuilder(t *testing.T) {
	builder := &fakeConfiguredBuilder{}
	config := ContainerConfig{
		Directories:     []string{"/work"},
		ScriptMap:       map[string][]string{"/work": {"user_script.sh"}},
		PermissionFiles: []string{"/work/user_script.sh"},
		WorkDir:         "/work",
		Cmd:             []string{"sleep", "infinity"},
	}
	if err := applyContainerConfigToBuilder(builder, config); err != nil {
		t.Fatalf("applyContainerConfigToBuilder() error = %v", err)
	}
	if builder.workDir != "/work" {
		t.Fatalf("workDir = %q", builder.workDir)
	}
	if !reflect.DeepEqual(builder.cmd, config.Cmd) {
		t.Fatalf("cmd = %#v, want %#v", builder.cmd, config.Cmd)
	}
}

func TestApplyImageConfigToBuilderPropagatesError(t *testing.T) {
	builder := &fakeConfiguredBuilder{fakeImageBuilder: fakeImageBuilder{runErr: errors.New("run failed")}}
	config := ImageConfig{Directories: []string{"/app"}}
	err := applyImageConfigToBuilder(builder, config)
	if err == nil || err.Error() != "failed to create directories: failed to create directory /app: run failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyContainerConfigToBuilderPropagatesCopyError(t *testing.T) {
	builder := &fakeConfiguredBuilder{fakeImageBuilder: fakeImageBuilder{addErr: errors.New("add failed")}}
	config := ContainerConfig{ScriptMap: map[string][]string{"/app": {"executor.sh"}}}
	err := applyContainerConfigToBuilder(builder, config)
	if err == nil || err.Error() != "failed to copy scripts: failed to copy script executor.sh to /app: add failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

var _ configuredBuilder = (*fakeConfiguredBuilder)(nil)
var _ imageBuilder = (*buildah.Builder)(nil)
