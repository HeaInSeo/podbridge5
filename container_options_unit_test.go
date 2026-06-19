package podbridge5

import (
	"strings"
	"testing"

	"go.podman.io/podman/v6/pkg/specgen"
)

func TestNewSpecAppliesContainerOptions(t *testing.T) {
	spec, err := NewSpec(
		WithImageName("docker.io/library/alpine:latest"),
		WithName("unit-container"),
		WithTerminal(true),
		WithPod("pod-id"),
		WithSysAdmin(),
		WithUnconfinedSeccomp(),
		WithEnv("ONE", "1"),
		WithEnvs(map[string]string{"TWO": "2"}),
		WithCommand([]string{"sh", "-c", "echo ok"}),
	)
	if err != nil {
		t.Fatalf("NewSpec returned error: %v", err)
	}

	if spec.Image != "docker.io/library/alpine:latest" {
		t.Fatalf("image mismatch: %q", spec.Image)
	}
	if spec.Name != "unit-container" {
		t.Fatalf("name mismatch: %q", spec.Name)
	}
	if spec.Terminal == nil || !*spec.Terminal {
		t.Fatalf("terminal flag was not set")
	}
	if spec.Pod != "pod-id" {
		t.Fatalf("pod mismatch: %q", spec.Pod)
	}
	if len(spec.CapAdd) != 1 || spec.CapAdd[0] != "SYS_ADMIN" {
		t.Fatalf("capabilities mismatch: %#v", spec.CapAdd)
	}
	if spec.SeccompPolicy != "unconfined" {
		t.Fatalf("seccomp policy mismatch: %q", spec.SeccompPolicy)
	}
	if spec.Env["ONE"] != "1" || spec.Env["TWO"] != "2" {
		t.Fatalf("environment mismatch: %#v", spec.Env)
	}
	if len(spec.Command) != 3 || spec.Command[0] != "sh" {
		t.Fatalf("command mismatch: %#v", spec.Command)
	}
}

func TestNewSpecPropagatesOptionError(t *testing.T) {
	_, err := NewSpec(WithHealthChecker("not-a-cmd-shell", "30s", 3, "5s", "0s"))
	if err == nil {
		t.Fatal("expected invalid healthcheck error")
	}
	if !strings.Contains(err.Error(), "invalid healthcheck config") {
		t.Fatalf("expected wrapped healthcheck error, got %v", err)
	}
}

func TestWithHealthCheckerSetsConfig(t *testing.T) {
	spec := &specgen.SpecGenerator{}
	err := WithHealthChecker(DefaultHealthcheckCommand(), "30s", 3, "5s", "0s")(spec)
	if err != nil {
		t.Fatalf("WithHealthChecker returned error: %v", err)
	}
	if spec.HealthConfig == nil {
		t.Fatal("health config was not set")
	}
	if len(spec.HealthConfig.Test) != 2 || spec.HealthConfig.Test[1] != ContainerHealthcheckPath {
		t.Fatalf("healthcheck command mismatch: %#v", spec.HealthConfig.Test)
	}
}
