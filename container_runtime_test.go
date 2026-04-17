package podbridge5

import (
	"context"
	"errors"
	"testing"

	"github.com/containers/podman/v5/libpod/define"
	entitiesTypes "github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/specgen"
)

type fakeContainerRuntime struct {
	containerExists    bool
	containerExistsErr error
	ensureImageErr     error
	createResp         *entitiesTypes.ContainerCreateResponse
	createErr          error
	startErr           error
	inspectResp        *define.InspectContainerData
	inspectErr         error
	startedID          string
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

func TestCreateContainerWithRuntime_ReusesExistingContainer(t *testing.T) {
	runtime := &fakeContainerRuntime{
		containerExists: true,
		inspectResp:     &define.InspectContainerData{ID: "existing-id", State: &define.InspectContainerState{Running: true}},
	}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	got, err := createContainerWithRuntime(context.Background(), runtime, spec)
	if err != nil {
		t.Fatalf("createContainerWithRuntime() error = %v", err)
	}
	if got.ID != "existing-id" || got.Status != Running {
		t.Fatalf("unexpected reused result: %+v", got)
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
	runtime := &fakeContainerRuntime{createResp: &entitiesTypes.ContainerCreateResponse{ID: "start-id"}, startErr: errors.New("boom")}
	spec, err := NewSpec(WithImageName("docker.io/library/alpine:latest"), WithName("demo-container"))
	if err != nil {
		t.Fatalf("NewSpec() error = %v", err)
	}

	_, err = startContainerWithRuntime(context.Background(), runtime, spec)
	if err == nil || err.Error() != "boom" {
		t.Fatalf("startContainerWithRuntime() error = %v, want boom", err)
	}
}
