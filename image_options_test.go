package podbridge5

import (
	"bytes"
	"os"
	"reflect"
	"strconv"
	"testing"

	"go.podman.io/buildah"
	"go.podman.io/buildah/define"
	imageTypes "go.podman.io/image/v5/types"
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

func TestNewUserNamespaceBuildConfig(t *testing.T) {
	got := NewUserNamespaceBuildConfig("registry.example.com/team/tool:latest", StorageNativeOverlay)
	if got.OutputRef != "registry.example.com/team/tool:latest" {
		t.Fatalf("unexpected output ref: %q", got.OutputRef)
	}
	if got.StorageMode != StorageNativeOverlay {
		t.Fatalf("unexpected storage mode: %q", got.StorageMode)
	}
	if got.Isolation != BuildIsolationOCI {
		t.Fatalf("unexpected isolation: %q", got.Isolation)
	}
	if got.UserNamespaceMode != UserNamespaceModeAuto {
		t.Fatalf("unexpected user namespace mode: %q", got.UserNamespaceMode)
	}
}

func TestNewExternalUserNamespaceBuildConfig(t *testing.T) {
	got := NewExternalUserNamespaceBuildConfig("registry.example.com/team/tool:latest", StorageNativeOverlay)
	if got.OutputRef != "registry.example.com/team/tool:latest" {
		t.Fatalf("unexpected output ref: %q", got.OutputRef)
	}
	if got.StorageMode != StorageNativeOverlay {
		t.Fatalf("unexpected storage mode: %q", got.StorageMode)
	}
	if got.Isolation != BuildIsolationOCI {
		t.Fatalf("unexpected isolation: %q", got.Isolation)
	}
	if got.UserNamespaceMode != UserNamespaceModeExternal {
		t.Fatalf("unexpected user namespace mode: %q", got.UserNamespaceMode)
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

func TestUserNamespaceStoreOptions(t *testing.T) {
	tests := []struct {
		name        string
		mode        StorageMode
		wantDriver  string
		wantOptions []string
	}{
		{name: "vfs", mode: StorageVFS, wantDriver: "vfs"},
		{name: "native overlay", mode: StorageNativeOverlay, wantDriver: "overlay"},
		{name: "fuse overlay", mode: StorageFuseOverlay, wantDriver: "overlay", wantOptions: []string{"overlay.mount_program=/usr/bin/fuse-overlayfs"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := UserNamespaceStoreOptions(UserNamespaceBuildConfig{
				StorageMode: tc.mode,
				RunRoot:     "/run/custom",
				GraphRoot:   "/graph/custom",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.RunRoot != "/run/custom" || got.GraphRoot != "/graph/custom" {
				t.Fatalf("unexpected roots: %q %q", got.RunRoot, got.GraphRoot)
			}
			if got.GraphDriverName != tc.wantDriver {
				t.Fatalf("unexpected driver: got %q want %q", got.GraphDriverName, tc.wantDriver)
			}
			if !reflect.DeepEqual(got.GraphDriverOptions, tc.wantOptions) {
				t.Fatalf("unexpected graph driver options: got %v want %v", got.GraphDriverOptions, tc.wantOptions)
			}
		})
	}
}

func TestUserNamespaceStoreOptionsRequiresExplicitStorageMode(t *testing.T) {
	if _, err := UserNamespaceStoreOptions(UserNamespaceBuildConfig{}); err == nil {
		t.Fatal("expected error for empty storage mode")
	}
}

func TestUserNamespaceStoreOptionsRejectsUnknownStorageMode(t *testing.T) {
	if _, err := UserNamespaceStoreOptions(UserNamespaceBuildConfig{StorageMode: "overlay-maybe"}); err == nil {
		t.Fatal("expected error for unsupported storage mode")
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
	if got.ConfigureNetwork != define.NetworkDefault {
		t.Fatalf("unexpected network configuration: %v", got.ConfigureNetwork)
	}
}

func TestUserNamespaceImageBuildOptionsWithBuildLog(t *testing.T) {
	var buf bytes.Buffer
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
		BuildLog:  &buf,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Out != &buf {
		t.Fatalf("expected Out to be the provided BuildLog writer, got %v", got.Out)
	}
}

func TestUserNamespaceImageBuildOptionsWithoutBuildLog_OutUnset(t *testing.T) {
	// Nil BuildLog must leave Out unset (nil) so Buildah's own default
	// (os.Stdout) applies — preserving prior behavior for existing callers.
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Out != nil {
		t.Fatalf("expected Out to remain nil when BuildLog is not set, got %v", got.Out)
	}
}

func TestUserNamespaceImageBuildOptionsWithNetworkDisabled(t *testing.T) {
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef:            "registry.example.com/team/tool:latest",
		NetworkConfiguration: define.NetworkDisabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ConfigureNetwork != define.NetworkDisabled {
		t.Fatalf("unexpected network configuration: %v", got.ConfigureNetwork)
	}
}

func TestUserNamespaceImageBuildOptionsWithExplicitIsolation(t *testing.T) {
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
		Isolation: BuildIsolationOCI,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Isolation != define.IsolationOCI {
		t.Fatalf("unexpected isolation: %v", got.Isolation)
	}
}

func TestUserNamespaceImageBuildOptionsWithRegistryCertDir(t *testing.T) {
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef:           "registry.example.com/team/tool:latest",
		RegistryCertDirPath: "/etc/containers/certs.d",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.SystemContext == nil {
		t.Fatal("expected system context")
	}
	if got.SystemContext.DockerPerHostCertDirPath != "/etc/containers/certs.d" {
		t.Fatalf("unexpected cert dir: %q", got.SystemContext.DockerPerHostCertDirPath)
	}
}

func TestUserNamespaceImageBuildOptionsRejectsUnknownIsolation(t *testing.T) {
	if _, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
		Isolation: "bad-isolation",
	}); err == nil {
		t.Fatal("expected error for unknown isolation")
	}
}

func TestUserNamespaceImageBuildOptionsRejectsInvalidCacheRef(t *testing.T) {
	if _, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
		CacheRef:  "not a valid reference",
	}); err == nil {
		t.Fatal("expected error for invalid cache ref")
	}
}

func TestBuildIsolationOCIRootless(t *testing.T) {
	got, err := UserNamespaceImageBuildOptions(UserNamespaceBuildConfig{
		OutputRef: "registry.example.com/team/tool:latest",
		Isolation: BuildIsolationOCIRootless,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Isolation != define.IsolationOCIRootless {
		t.Fatalf("unexpected isolation: %v", got.Isolation)
	}
}

func TestDefaultUserNamespaceBuildEnvironment(t *testing.T) {
	got := DefaultUserNamespaceBuildEnvironment()
	if got[ContainersUserNamespaceConfiguredEnv] != "done" {
		t.Fatalf("expected user namespace configured marker")
	}
	if got[ContainersRootlessUIDEnv] != strconv.Itoa(os.Geteuid()) || got[ContainersRootlessGIDEnv] != strconv.Itoa(os.Getegid()) {
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

func TestUserNamespaceBuildEnvironmentWithExplicitIsolation(t *testing.T) {
	got, err := UserNamespaceBuildEnvironment(UserNamespaceBuildConfig{Isolation: BuildIsolationOCI})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[BuildahIsolationEnv] != "oci" {
		t.Fatalf("unexpected isolation env: %q", got[BuildahIsolationEnv])
	}
}

func TestUserNamespaceBuildEnvironmentRejectsUnknownIsolation(t *testing.T) {
	if _, err := UserNamespaceBuildEnvironment(UserNamespaceBuildConfig{Isolation: "bad-isolation"}); err == nil {
		t.Fatal("expected error for unknown isolation")
	}
}

func TestUserNamespaceBuildExecutionProfile(t *testing.T) {
	got, err := UserNamespaceBuildExecutionProfile(UserNamespaceBuildConfig{
		Runtime:                   "crun",
		Isolation:                 BuildIsolationOCI,
		UserNamespaceMode:         UserNamespaceModeExternal,
		StorageMode:               StorageFuseOverlay,
		RunRoot:                   "/run/custom",
		GraphRoot:                 "/graph/custom",
		FuseOverlayfsMountProgram: "/custom/fuse-overlayfs",
		RegistryCertDirPath:       "/etc/containers/certs.d",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ActualEUID != os.Geteuid() || got.ActualEGID != os.Getegid() {
		t.Fatalf("unexpected actual IDs: uid=%d gid=%d", got.ActualEUID, got.ActualEGID)
	}
	if !got.UserNamespace {
		t.Fatal("expected user namespace profile marker")
	}
	if got.UserNamespaceMode != UserNamespaceModeExternal {
		t.Fatalf("unexpected user namespace mode: %q", got.UserNamespaceMode)
	}
	if got.StorageMode != StorageFuseOverlay || got.StorageDriver != "overlay" {
		t.Fatalf("unexpected storage profile: mode=%q driver=%q", got.StorageMode, got.StorageDriver)
	}
	if got.RunRoot != "/run/custom" || got.GraphRoot != "/graph/custom" {
		t.Fatalf("unexpected roots: %q %q", got.RunRoot, got.GraphRoot)
	}
	if got.MountProgram != "/custom/fuse-overlayfs" {
		t.Fatalf("unexpected mount program: %q", got.MountProgram)
	}
	if got.Isolation != "oci" || got.Runtime != "crun" {
		t.Fatalf("unexpected build profile: isolation=%q runtime=%q", got.Isolation, got.Runtime)
	}
	if got.BuildahVersion != buildah.Version {
		t.Fatalf("unexpected buildah version: %q", got.BuildahVersion)
	}
	if got.RegistryCertDirPath != "/etc/containers/certs.d" {
		t.Fatalf("unexpected cert dir: %q", got.RegistryCertDirPath)
	}
}

func TestUserNamespaceBuildExecutionProfileRejectsInvalidStorageMode(t *testing.T) {
	if _, err := UserNamespaceBuildExecutionProfile(UserNamespaceBuildConfig{StorageMode: "bad-storage"}); err == nil {
		t.Fatal("expected error for bad storage mode")
	}
}

func TestUserNamespaceBuildExecutionProfileRejectsInvalidIsolation(t *testing.T) {
	if _, err := UserNamespaceBuildExecutionProfile(UserNamespaceBuildConfig{
		StorageMode: StorageVFS,
		Isolation:   "bad-isolation",
	}); err == nil {
		t.Fatal("expected error for bad isolation")
	}
}

func TestUserNamespaceBuildExecutionProfileRejectsInvalidUserNamespaceMode(t *testing.T) {
	if _, err := UserNamespaceBuildExecutionProfile(UserNamespaceBuildConfig{
		StorageMode:       StorageVFS,
		UserNamespaceMode: "nested-maybe",
	}); err == nil {
		t.Fatal("expected error for bad user namespace mode")
	}
}

func TestOverlayMountProgramNoOption(t *testing.T) {
	if got := overlayMountProgram([]string{"overlay.size=1G"}); got != "" {
		t.Fatalf("expected empty mount program, got %q", got)
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

func TestUserNamespaceSystemContext(t *testing.T) {
	got := UserNamespaceSystemContext(UserNamespaceBuildConfig{})
	if got.DockerInsecureSkipTLSVerify == imageTypes.OptionalBoolTrue {
		t.Fatal("expected TLS verification left at default when InsecureSkipTLSVerify is unset")
	}
	if got.DockerPerHostCertDirPath != "" {
		t.Fatalf("expected empty cert dir path, got %q", got.DockerPerHostCertDirPath)
	}

	got = UserNamespaceSystemContext(UserNamespaceBuildConfig{InsecureSkipTLSVerify: true, RegistryCertDirPath: "/certs"})
	if got.DockerInsecureSkipTLSVerify != imageTypes.OptionalBoolTrue {
		t.Fatalf("expected DockerInsecureSkipTLSVerify=true, got %v", got.DockerInsecureSkipTLSVerify)
	}
	if got.DockerPerHostCertDirPath != "/certs" {
		t.Fatalf("expected cert dir path to be preserved, got %q", got.DockerPerHostCertDirPath)
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
	if got.Chmod != "755" {
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
