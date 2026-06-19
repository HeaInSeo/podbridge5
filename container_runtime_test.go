package podbridge5

import (
	"context"
	"errors"
	"testing"

	"go.podman.io/podman/v6/libpod/define"
	entitiesTypes "go.podman.io/podman/v6/pkg/domain/entities/types"
	"go.podman.io/podman/v6/pkg/specgen"
)

type fakeContainerRuntime struct {
	containerExists    bool
	containerExistsErr error
	ensureImageErr     error
	createResp         *entitiesTypes.ContainerCreateResponse
	createErr          error
	startErr           error
	removeErr          error
	inspectResp        *define.InspectContainerData
	inspectErr         error
	startedID          string
	removedID          string
	ensuredImage       string
	createdSpec        *specgen.SpecGenerator
}

func (f *fakeContainerRuntime) ContainerExists(context.Context, string) (bool, error) {
	return f.containerExists, f.containerExistsErr
}

func (f *fakeContainerRuntime) EnsureImage(_ context.Context, imageRef string) error {
	f.ensuredImage = imageRef
	return f.ensureImageErr
}

func (f *fakeContainerRuntime) CreateContainer(_ context.Context, spec *specgen.SpecGenerator) (*entitiesTypes.ContainerCreateResponse, error) {
	f.createdSpec = spec
	if f.createResp != nil {
		return f.createResp, f.createErr
	}
	return &entitiesTypes.ContainerCreateResponse{ID: "generated-id"}, f.createErr
}

func (f *fakeContainerRuntime) StartContainer(_ context.Context, containerID string) error {
	f.startedID = containerID
	return f.startErr
}

func (f *fakeContainerRuntime) RemoveContainer(_ context.Context, containerID string) error {
	f.removedID = containerID
	return f.removeErr
}

func (f *fakeContainerRuntime) InspectContainer(context.Context, string) (*define.InspectContainerData, error) {
	return f.inspectResp, f.inspectErr
}

func TestContainerStatusFromInspectState(t *testing.T) {
	tests := []struct {
		name  string
		state *define.InspectContainerState
		want  ContainerStatus
	}{
		{name: "nil state", state: nil, want: Created},
		{name: "running", state: &define.InspectContainerState{Running: true}, want: Running},
		{name: "paused", state: &define.InspectContainerState{Paused: true}, want: Paused},
		{name: "dead", state: &define.InspectContainerState{Dead: true}, want: Dead},
		{name: "created string", state: &define.InspectContainerState{Status: "created"}, want: Created},
		{name: "configured string", state: &define.InspectContainerState{Status: "configured"}, want: Created},
		{name: "exited zero", state: &define.InspectContainerState{ExitCode: 0}, want: Exited},
		{name: "exited nonzero", state: &define.InspectContainerState{ExitCode: 2}, want: ExitedErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := containerStatusFromInspectState(tc.state); got != tc.want {
				t.Fatalf("containerStatusFromInspectState() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCreateContainerWithRuntime_CreatesNewContainer(t *testing.T) {
	runtime := &fakeContainerRuntime{createResp: &entitiesTypes.ContainerCreateResponse{ID: "new-container", Warnings: []string{"warn"}}}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	got, err := createContainerWithRuntime(context.Background(), runtime, spec)
	if err != nil {
		t.Fatalf("createContainerWithRuntime() error = %v", err)
	}
	if got.ID != "new-container" || got.Name != "demo-container" || got.Status != Created {
		t.Fatalf("unexpected result: %+v", got)
	}
	if runtime.ensuredImage != "docker.io/library/alpine:latest" {
		t.Fatalf("EnsureImage called with %q", runtime.ensuredImage)
	}
	if runtime.createdSpec != spec {
		t.Fatalf("CreateContainer did not receive original spec")
	}
}

func TestCreateContainerWithRuntime_ExistingContainerReturnsError(t *testing.T) {
	runtime := &fakeContainerRuntime{
		containerExists: true,
		inspectResp:     &define.InspectContainerData{ID: "existing-id", Image: "docker.io/library/busybox:latest", State: &define.InspectContainerState{Running: true}},
	}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	got, err := createContainerWithRuntime(context.Background(), runtime, spec)
	if err == nil {
		t.Fatalf("createContainerWithRuntime() error = nil, want existing container error")
	}
	if !errors.Is(err, ErrContainerAlreadyExists) {
		t.Fatalf("createContainerWithRuntime() error = %v, want ErrContainerAlreadyExists", err)
	}
	if got != nil {
		t.Fatalf("expected nil result, got %+v", got)
	}
	if runtime.ensuredImage != "" {
		t.Fatalf("EnsureImage should not be called for existing container")
	}
}

func TestStartContainerWithRuntime_StartsCreatedContainer(t *testing.T) {
	runtime := &fakeContainerRuntime{createResp: &entitiesTypes.ContainerCreateResponse{ID: "start-id"}}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	id, err := startContainerWithRuntime(context.Background(), runtime, spec)
	if err != nil {
		t.Fatalf("startContainerWithRuntime() error = %v", err)
	}
	if id != "start-id" {
		t.Fatalf("startContainerWithRuntime() id = %q, want start-id", id)
	}
	if runtime.startedID != "start-id" {
		t.Fatalf("StartContainer called with %q", runtime.startedID)
	}
}

func TestStartContainerWithRuntime_PropagatesStartError(t *testing.T) {
	startErr := errors.New("boom")
	runtime := &fakeContainerRuntime{createResp: &entitiesTypes.ContainerCreateResponse{ID: "start-id"}, startErr: startErr}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	_, err = startContainerWithRuntime(context.Background(), runtime, spec)
	if err == nil {
		t.Fatalf("startContainerWithRuntime() error = nil, want non-nil")
	}
	if !errors.Is(err, startErr) {
		t.Fatalf("startContainerWithRuntime() error = %v, want error wrapping %v", err, startErr)
	}
	if runtime.removedID != "start-id" {
		t.Fatalf("RemoveContainer called with %q, want start-id", runtime.removedID)
	}
}

func TestStartContainerWithRuntimeRejectsInvalidInput(t *testing.T) {
	runtime := &fakeContainerRuntime{}
	if _, err := startContainerWithRuntime(nil, runtime, &specgen.SpecGenerator{}); err == nil {
		t.Fatal("expected nil context error")
	}
	if _, err := startContainerWithRuntime(context.Background(), runtime, nil); err == nil {
		t.Fatal("expected nil spec error")
	}
}

func TestCreateContainerWithRuntimeFailPaths(t *testing.T) {
	validSpec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	tests := []struct {
		name    string
		runtime *fakeContainerRuntime
		spec    *specgen.SpecGenerator
		wantErr error
	}{
		{
			name:    "missing image and name",
			runtime: &fakeContainerRuntime{},
			spec:    &specgen.SpecGenerator{},
		},
		{
			name:    "exists check failure",
			runtime: &fakeContainerRuntime{containerExistsErr: errors.New("exists denied")},
			spec:    validSpec,
		},
		{
			name:    "existing inspect failure",
			runtime: &fakeContainerRuntime{containerExists: true, inspectErr: errors.New("inspect denied")},
			spec:    validSpec,
		},
		{
			name:    "ensure image failure",
			runtime: &fakeContainerRuntime{ensureImageErr: errors.New("pull denied")},
			spec:    validSpec,
		},
		{
			name:    "create failure",
			runtime: &fakeContainerRuntime{createErr: errors.New("create denied")},
			spec:    validSpec,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := createContainerWithRuntime(context.Background(), tc.runtime, tc.spec)
			if err == nil {
				t.Fatal("expected error")
			}
			if got != nil {
				t.Fatalf("expected nil result, got %+v", got)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want wrapping %v", err, tc.wantErr)
			}
		})
	}
}

func TestInspectContainerWithRuntime(t *testing.T) {
	want := &define.InspectContainerData{ID: "container-id"}
	runtime := &fakeContainerRuntime{inspectResp: want}

	got, err := inspectContainerWithRuntime(context.Background(), runtime, "container-id")
	if err != nil {
		t.Fatalf("inspectContainerWithRuntime returned error: %v", err)
	}
	if got != want {
		t.Fatalf("inspect result pointer mismatch")
	}

	runtime.inspectErr = errors.New("inspect denied")
	if _, err := inspectContainerWithRuntime(context.Background(), runtime, "container-id"); err == nil {
		t.Fatal("expected inspect error")
	}
}
