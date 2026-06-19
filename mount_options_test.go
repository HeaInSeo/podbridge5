package podbridge5

import (
	"testing"

	"go.podman.io/podman/v6/pkg/specgen"
)

func TestWithFileBindingsInitializesEnvAndMountsFiles(t *testing.T) {
	spec := &specgen.SpecGenerator{}

	err := WithFileBindings("/inputs", map[string]string{
		"SAMPLE_A": "/host/a.fastq",
		"SAMPLE_B": "/host/b.fastq",
	})(spec)
	if err != nil {
		t.Fatalf("WithFileBindings returned error: %v", err)
	}

	if spec.WorkDir != "/inputs" {
		t.Fatalf("workdir mismatch: %q", spec.WorkDir)
	}
	if spec.Env["SAMPLE_A"] != "/inputs/SAMPLE_A" {
		t.Fatalf("SAMPLE_A env mismatch: %#v", spec.Env)
	}
	if spec.Env["SAMPLE_B"] != "/inputs/SAMPLE_B" {
		t.Fatalf("SAMPLE_B env mismatch: %#v", spec.Env)
	}
	if len(spec.Mounts) != 2 {
		t.Fatalf("expected two mounts, got %#v", spec.Mounts)
	}
	for _, mount := range spec.Mounts {
		if len(mount.Options) != 1 || mount.Options[0] != "ro" {
			t.Fatalf("mount should be read-only: %#v", mount)
		}
	}
}

func TestWithFileBindingsPreservesExistingEnv(t *testing.T) {
	spec := &specgen.SpecGenerator{}
	spec.Env = map[string]string{"EXISTING": "value"}

	if err := WithFileBindings("/data", map[string]string{"INPUT": "/host/input.txt"})(spec); err != nil {
		t.Fatalf("WithFileBindings returned error: %v", err)
	}

	if spec.Env["EXISTING"] != "value" {
		t.Fatalf("existing env was overwritten: %#v", spec.Env)
	}
	if spec.Env["INPUT"] != "/data/INPUT" {
		t.Fatalf("INPUT env mismatch: %#v", spec.Env)
	}
}
