package podbridge5

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.podman.io/podman/v6/pkg/specgen"
)

type fakeVolumeContainerRuntime struct {
	ensureErr error
	createErr error
	startErr  error
	stopErr   error
	removeErr error

	ensuredImage string
	createdSpec  *specgen.SpecGenerator
	startedID    string
	stoppedID    string
	removedID    string
}

func (f *fakeVolumeContainerRuntime) EnsureImage(_ context.Context, imageRef string) error {
	f.ensuredImage = imageRef
	return f.ensureErr
}

func (f *fakeVolumeContainerRuntime) CreateContainer(_ context.Context, spec *specgen.SpecGenerator) (string, error) {
	f.createdSpec = spec
	if f.createErr != nil {
		return "", f.createErr
	}
	return "container-id", nil
}

func (f *fakeVolumeContainerRuntime) StartContainer(_ context.Context, containerID string) error {
	f.startedID = containerID
	return f.startErr
}

func (f *fakeVolumeContainerRuntime) StopContainer(_ context.Context, containerID string) error {
	f.stoppedID = containerID
	return f.stopErr
}

func (f *fakeVolumeContainerRuntime) RemoveContainer(_ context.Context, containerID string) error {
	f.removedID = containerID
	return f.removeErr
}

func TestNewVolumeWriterSpec(t *testing.T) {
	spec, err := newVolumeWriterSpec("demo-volume", "/data")
	if err != nil {
		t.Fatalf("newVolumeWriterSpec() error = %v", err)
	}

	if spec.Image != volumeTransferImage {
		t.Fatalf("spec.Image = %q, want %q", spec.Image, volumeTransferImage)
	}
	if spec.Name == volumeWriterContainerNamePrefix || spec.Name == "" {
		t.Fatalf("spec.Name = %q, want unique temp container name", spec.Name)
	}
	if got := spec.Env["MOUNT"]; got != "/data" {
		t.Fatalf("spec.Env[MOUNT] = %q, want /data", got)
	}
	if len(spec.Command) != 3 {
		t.Fatalf("len(spec.Command) = %d, want 3", len(spec.Command))
	}
	if spec.Command[2] != "mkdir -p \"$MOUNT\"; exec tail -f /dev/null" {
		t.Fatalf("spec.Command[2] = %q", spec.Command[2])
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("len(spec.Volumes) = %d, want 1", len(spec.Volumes))
	}
	if spec.Volumes[0].Name != "demo-volume" || spec.Volumes[0].Dest != "/data" {
		t.Fatalf("unexpected volume mapping: %+v", spec.Volumes[0])
	}
}

func TestNewVolumeReaderSpec(t *testing.T) {
	spec, err := newVolumeReaderSpec("demo-volume", "/cache")
	if err != nil {
		t.Fatalf("newVolumeReaderSpec() error = %v", err)
	}

	if spec.Image != volumeTransferImage {
		t.Fatalf("spec.Image = %q, want %q", spec.Image, volumeTransferImage)
	}
	if spec.Name == volumeReaderContainerNamePrefix || spec.Name == "" {
		t.Fatalf("spec.Name = %q, want unique temp container name", spec.Name)
	}
	if len(spec.Command) != 3 {
		t.Fatalf("len(spec.Command) = %d, want 3", len(spec.Command))
	}
	if spec.Command[2] != "mkdir -p /data && sleep infinity" {
		t.Fatalf("spec.Command[2] = %q", spec.Command[2])
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("len(spec.Volumes) = %d, want 1", len(spec.Volumes))
	}
	if spec.Volumes[0].Name != "demo-volume" || spec.Volumes[0].Dest != "/cache" {
		t.Fatalf("unexpected volume mapping: %+v", spec.Volumes[0])
	}
}

func TestUniqueTempContainerName(t *testing.T) {
	first := uniqueTempContainerName("tmp")
	second := uniqueTempContainerName("tmp")
	if first == second {
		t.Fatalf("expected unique names, got %q and %q", first, second)
	}
}

func TestStartVolumeContainerWithRuntime(t *testing.T) {
	spec, err := newVolumeWriterSpec("demo-volume", "/data")
	if err != nil {
		t.Fatalf("newVolumeWriterSpec() error = %v", err)
	}
	runtime := &fakeVolumeContainerRuntime{}

	containerID, cleanup, err := startVolumeContainer(context.Background(), runtime, spec)
	if err != nil {
		t.Fatalf("startVolumeContainer returned error: %v", err)
	}
	if containerID != "container-id" {
		t.Fatalf("containerID = %q", containerID)
	}
	if cleanup == nil {
		t.Fatal("expected cleanup function")
	}
	if runtime.ensuredImage != volumeTransferImage || runtime.createdSpec != spec || runtime.startedID != "container-id" {
		t.Fatalf("runtime calls mismatch: %#v", runtime)
	}

	cleanup()
	if runtime.stoppedID != "container-id" || runtime.removedID != "container-id" {
		t.Fatalf("cleanup did not stop/remove container: %#v", runtime)
	}
}

func TestStartVolumeContainerWithRuntimeFailPaths(t *testing.T) {
	spec, err := newVolumeWriterSpec("demo-volume", "/data")
	if err != nil {
		t.Fatalf("newVolumeWriterSpec() error = %v", err)
	}

	tests := []struct {
		name      string
		runtime   *fakeVolumeContainerRuntime
		want      string
		wantClean bool
	}{
		{
			name:    "ensure image failure",
			runtime: &fakeVolumeContainerRuntime{ensureErr: errors.New("pull denied")},
			want:    "ensure image",
		},
		{
			name:    "create failure",
			runtime: &fakeVolumeContainerRuntime{createErr: errors.New("create denied")},
			want:    "create container",
		},
		{
			name:      "start failure cleans up",
			runtime:   &fakeVolumeContainerRuntime{startErr: errors.New("start denied")},
			want:      "start container",
			wantClean: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			containerID, cleanup, err := startVolumeContainer(context.Background(), tc.runtime, spec)
			if err == nil {
				t.Fatal("expected error")
			}
			if containerID != "" || cleanup != nil {
				t.Fatalf("expected no container/cleanup, got id=%q cleanupSet=%t", containerID, cleanup != nil)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
			if tc.wantClean && (tc.runtime.stoppedID != "container-id" || tc.runtime.removedID != "container-id") {
				t.Fatalf("start failure did not cleanup: %#v", tc.runtime)
			}
		})
	}
}
