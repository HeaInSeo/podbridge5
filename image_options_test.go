package podbridge5

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
)

func TestDefaultImageBuildOptions(t *testing.T) {
	got := DefaultImageBuildOptions("registry.example.com/team/tool:latest")
	if got.ContextDirectory != "." {
		t.Fatalf("unexpected context directory: %q", got.ContextDirectory)
	}
	if got.PullPolicy != define.PullIfMissing {
		t.Fatalf("unexpected pull policy: %v", got.PullPolicy)
	}
	if got.Isolation != define.IsolationOCI {
		t.Fatalf("unexpected isolation: %v", got.Isolation)
	}
	if got.Output != "registry.example.com/team/tool:latest" {
		t.Fatalf("unexpected output: %q", got.Output)
	}
	if got.OutputFormat != buildah.Dockerv2ImageManifest {
		t.Fatalf("unexpected output format: %q", got.OutputFormat)
	}
	if got.SystemContext == nil {
		t.Fatal("expected system context")
	}
}

func TestNormalizePushDestination(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		wantError bool
	}{
		{name: "docker transport already present", input: "docker://registry.example.com/app:latest", want: "docker://registry.example.com/app:latest"},
		{name: "adds docker transport", input: "registry.example.com/app:latest", want: "docker://registry.example.com/app:latest"},
		{name: "trimmed input", input: "  registry.example.com/app:latest  ", want: "docker://registry.example.com/app:latest"},
		{name: "empty", input: "   ", wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizePushDestination(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected destination: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestNewBuilderOptions(t *testing.T) {
	caps := []string{"CAP_NET_BIND_SERVICE", "CAP_SYS_ADMIN"}
	got, err := newBuilderOptions("docker.io/library/alpine:latest", caps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FromImage != "docker.io/library/alpine:latest" {
		t.Fatalf("unexpected from image: %q", got.FromImage)
	}
	if got.Isolation != define.IsolationOCI {
		t.Fatalf("unexpected isolation: %v", got.Isolation)
	}
	if got.CommonBuildOpts == nil {
		t.Fatal("expected common build options")
	}
	if got.SystemContext == nil {
		t.Fatal("expected system context")
	}
	if !reflect.DeepEqual(got.Capabilities, caps) {
		t.Fatalf("unexpected capabilities: got %v want %v", got.Capabilities, caps)
	}

	caps[0] = "CHANGED"
	if got.Capabilities[0] != "CAP_NET_BIND_SERVICE" {
		t.Fatal("builder options should copy capabilities slice")
	}
}

func TestNewBuilderOptionsRejectsEmptyBaseImage(t *testing.T) {
	if _, err := newBuilderOptions("   ", nil); err == nil {
		t.Fatal("expected error for empty base image")
	}
}

func TestNewDefaultAddAndCopyOptions(t *testing.T) {
	var hasher bytes.Buffer
	got := newDefaultAddAndCopyOptions(&hasher)
	if got.Chmod != "0o755" {
		t.Fatalf("unexpected chmod: %q", got.Chmod)
	}
	if got.Chown != "0:0" {
		t.Fatalf("unexpected chown: %q", got.Chown)
	}
	if got.ContextDir != "." {
		t.Fatalf("unexpected context dir: %q", got.ContextDir)
	}
	if got.DryRun {
		t.Fatal("expected dry run to be false")
	}
	if got.Hasher == nil {
		t.Fatal("expected hasher to be set")
	}
}
