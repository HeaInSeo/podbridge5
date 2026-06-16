package podbridge5

import (
	"bytes"
	"reflect"
	"testing"

	"go.podman.io/buildah"
	"go.podman.io/buildah/define"
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

func TestDefaultUserNamespaceStoreOptions(t *testing.T) {
	got := DefaultUserNamespaceStoreOptions()
	if got.RunRoot != DefaultUserNamespaceRunRoot {
		t.Fatalf("unexpected runroot: %q", got.RunRoot)
	}
	if got.GraphRoot != DefaultUserNamespaceGraphRoot {
		t.Fatalf("unexpected graphroot: %q", got.GraphRoot)
	}
	if got.GraphDriverName != "overlay" {
		t.Fatalf("unexpected graph driver: %q", got.GraphDriverName)
	}
	if got.PullOptions["enable_partial_images"] != "true" {
		t.Fatalf("expected partial image pulls to be enabled")
	}
}

func TestStoreOptions(t *testing.T) {
	opts := DefaultUserNamespaceStoreOptions()
	for _, applyOpt := range []StoreOption{
		WithStoreRoots("/run/custom", "/graph/custom"),
		WithStoreDriver("vfs"),
		WithFuseOverlayfsMountProgram("/custom/fuse-overlayfs"),
		WithPartialImagePulls(false),
	} {
		if err := applyOpt(&opts); err != nil {
			t.Fatalf("unexpected option error: %v", err)
		}
	}
	if opts.RunRoot != "/run/custom" || opts.GraphRoot != "/graph/custom" {
		t.Fatalf("unexpected roots: %q %q", opts.RunRoot, opts.GraphRoot)
	}
	if opts.GraphDriverName != "vfs" {
		t.Fatalf("unexpected driver: %q", opts.GraphDriverName)
	}
	if !reflect.DeepEqual(opts.GraphDriverOptions, []string{"overlay.mount_program=/custom/fuse-overlayfs"}) {
		t.Fatalf("unexpected graph driver options: %v", opts.GraphDriverOptions)
	}
	if _, ok := opts.PullOptions["enable_partial_images"]; ok {
		t.Fatal("partial image pulls should be disabled")
	}
}

func TestUserNamespaceImageBuildOptions(t *testing.T) {
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef:        "registry.example.com/team/tool:latest",
		ContextDirectory: "/workspace",
		CacheRef:         "registry.example.com/team/tool-cache:latest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ContextDirectory != "/workspace" {
		t.Fatalf("unexpected context directory: %q", got.ContextDirectory)
	}
	if got.Isolation != define.IsolationChroot {
		t.Fatalf("unexpected isolation: %v", got.Isolation)
	}
	if got.Runtime != DefaultUserNamespaceRuntime {
		t.Fatalf("unexpected runtime: %q", got.Runtime)
	}
	if !got.Layers {
		t.Fatal("expected layers to be enabled")
	}
	if len(got.CacheFrom) != 1 || len(got.CacheTo) != 1 {
		t.Fatalf("expected cache refs, got from=%d to=%d", len(got.CacheFrom), len(got.CacheTo))
	}
	if got.CacheFrom[0].String() != "registry.example.com/team/tool-cache:latest" {
		t.Fatalf("unexpected cache ref: %q", got.CacheFrom[0].String())
	}
}

func TestDefaultUserNamespaceBuildEnvironment(t *testing.T) {
	got := DefaultUserNamespaceBuildEnvironment()
	if got[ContainersUserNamespaceConfiguredEnv] != "done" {
		t.Fatalf("expected user namespace configured marker")
	}
	if got[ContainersRootlessUIDEnv] != "1000" || got[ContainersRootlessGIDEnv] != "1000" {
		t.Fatalf("unexpected rootless id env: uid=%q gid=%q", got[ContainersRootlessUIDEnv], got[ContainersRootlessGIDEnv])
	}
	if got[BuildahIsolationEnv] != "chroot" {
		t.Fatalf("unexpected isolation env: %q", got[BuildahIsolationEnv])
	}
	if got[BuildahRuntimeEnv] != DefaultUserNamespaceRuntime {
		t.Fatalf("unexpected runtime env: %q", got[BuildahRuntimeEnv])
	}
	if got[HomeEnv] != DefaultUserNamespaceHome {
		t.Fatalf("unexpected home env: %q", got[HomeEnv])
	}
	if got[XDGConfigHomeEnv] != DefaultUserNamespaceHome+"/.config" {
		t.Fatalf("unexpected config home env: %q", got[XDGConfigHomeEnv])
	}
}

func TestDefaultUserNamespaceBuildCapabilities(t *testing.T) {
	got := DefaultUserNamespaceBuildCapabilities()
	want := []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "SETFCAP", "SYS_CHROOT"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected capabilities: got %v want %v", got, want)
	}

	got[0] = "CHANGED"
	if DefaultUserNamespaceBuildCapabilities()[0] != "CHOWN" {
		t.Fatal("capabilities should return a fresh slice")
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
	if got.DryRun != DefaultAddAndCopyDryRun {
		t.Fatalf("unexpected dry run default: got %v want %v", got.DryRun, DefaultAddAndCopyDryRun)
	}
	if got.Hasher == nil {
		t.Fatal("expected hasher to be set")
	}
}

func TestNewDefaultAddAndCopyOptionsWithDryRun(t *testing.T) {
	got := newDefaultAddAndCopyOptionsWithDryRun(nil, true)
	if !got.DryRun {
		t.Fatal("expected dry run override to be true")
	}
}
