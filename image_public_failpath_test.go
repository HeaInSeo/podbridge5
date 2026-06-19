package podbridge5

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.podman.io/buildah"
	buildahDefine "go.podman.io/buildah/define"
	imageTypes "go.podman.io/image/v5/types"
	"go.podman.io/storage"
)

func TestBuilderOptions(t *testing.T) {
	opts := &buildah.BuilderOptions{}

	if err := WithArg("A", "1")(opts); err != nil {
		t.Fatalf("WithArg returned error: %v", err)
	}
	if err := WithArg("A", "2")(opts); err != nil {
		t.Fatalf("WithArg duplicate returned error: %v", err)
	}
	if opts.Args["A"] != "1" {
		t.Fatalf("WithArg should not overwrite existing args: %#v", opts.Args)
	}

	if err := WithFromImage("alpine")(opts); err != nil {
		t.Fatalf("WithFromImage returned error: %v", err)
	}
	if opts.FromImage != "alpine" {
		t.Fatalf("from image mismatch: %q", opts.FromImage)
	}
	if err := WithFromImage(" ")(opts); err == nil {
		t.Fatal("expected empty from image error")
	}

	if err := WithIsolation(buildahDefine.IsolationOCI)(opts); err != nil {
		t.Fatalf("WithIsolation returned error: %v", err)
	}
	if opts.Isolation != buildahDefine.IsolationOCI {
		t.Fatalf("isolation mismatch: %v", opts.Isolation)
	}

	if err := WithCommonBuildOptions(nil)(opts); err != nil {
		t.Fatalf("WithCommonBuildOptions returned error: %v", err)
	}
	if opts.CommonBuildOpts == nil {
		t.Fatal("nil common build options should initialize an empty value")
	}
	sysCtx := &imageTypes.SystemContext{ArchitectureChoice: "amd64"}
	if err := WithSystemContext(sysCtx)(opts); err != nil {
		t.Fatalf("WithSystemContext returned error: %v", err)
	}
	if opts.SystemContext != sysCtx {
		t.Fatalf("system context pointer mismatch")
	}
	if err := WithNetworkConfiguration(buildahDefine.NetworkDisabled)(opts); err != nil {
		t.Fatalf("WithNetworkConfiguration returned error: %v", err)
	}
	if opts.ConfigureNetwork != buildahDefine.NetworkDisabled {
		t.Fatalf("network policy mismatch: %v", opts.ConfigureNetwork)
	}
	if err := WithFormat(buildah.OCI)(opts); err != nil {
		t.Fatalf("WithFormat returned error: %v", err)
	}
	if opts.Format != buildah.OCI {
		t.Fatalf("format mismatch: %q", opts.Format)
	}
}

func TestNewBuilderPropagatesOptionErrorBeforeRuntime(t *testing.T) {
	_, err := NewBuilder(context.Background(), nil, WithFromImage(""))
	if err == nil {
		t.Fatal("expected builder option error")
	}
	if !strings.Contains(err.Error(), "from image cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPublicImageWrappersRejectNilStore(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "BuildDockerfileContent",
			run: func() error {
				_, _, err := BuildDockerfileContent(ctx, nil, "FROM scratch", "example.com/app:latest")
				return err
			},
		},
		{
			name: "BuildDockerfileContentWithOptions",
			run: func() error {
				_, _, err := BuildDockerfileContentWithOptions(ctx, nil, "FROM scratch", buildahDefine.BuildOptions{})
				return err
			},
		},
		{
			name: "BuildDockerfileContentUserNamespace",
			run: func() error {
				_, _, err := BuildDockerfileContentUserNamespace(ctx, nil, "FROM scratch", UserNamespaceBuildConfig{})
				return err
			},
		},
		{
			name: "PushImage",
			run: func() error {
				_, err := PushImage(ctx, nil, "image", "registry.local/image:tag")
				return err
			},
		},
		{
			name: "BuildAndPushDockerfileContent",
			run: func() error {
				_, _, err := BuildAndPushDockerfileContent(ctx, nil, "FROM scratch", "registry.local/image:tag")
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.run()
			if err == nil {
				t.Fatal("expected nil store error")
			}
			if !strings.Contains(err.Error(), "store must not be nil") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestStoreConstructorsPropagateOptionErrorsBeforeRuntime(t *testing.T) {
	sentinel := errors.New("bad store option")
	badOption := func(*storage.StoreOptions) error { return sentinel }

	if _, err := NewStoreWithOptions(badOption); !errors.Is(err, sentinel) {
		t.Fatalf("NewStoreWithOptions error = %v, want %v", err, sentinel)
	}
	if _, err := NewUserNamespaceStore(badOption); !errors.Is(err, sentinel) {
		t.Fatalf("NewUserNamespaceStore error = %v, want %v", err, sentinel)
	}
	if _, err := NewUserNamespaceStoreWithConfig(UserNamespaceBuildConfig{}); err == nil {
		t.Fatal("expected NewUserNamespaceStoreWithConfig to reject missing storage mode")
	}
}

func TestShutdownRejectsNilStore(t *testing.T) {
	if err := shutdown(nil, false); err == nil {
		t.Fatal("expected nil store error")
	}
}

func TestAddAndCopyOptionSetters(t *testing.T) {
	idMap := &buildahDefine.IDMappingOptions{}
	var hasher strings.Builder
	got := NewAddAndCopyOptions(
		WithChmod("0755"),
		WithChown("1000:1000"),
		WithPreserveOwnership(true),
		WithHasher(&hasher),
		WithExcludes([]string{"*.tmp"}),
		WithIgnoreFile(".containerignore"),
		WithContextDir("/workspace"),
		WithIDMappingOptions(idMap),
		WithDryRun(true),
		WithStripSetuidBit(true),
		WithStripSetgidBit(true),
		WithStripStickyBit(true),
	)

	if got.Chmod != "0755" || got.Chown != "1000:1000" {
		t.Fatalf("ownership options mismatch: %#v", got)
	}
	if !got.PreserveOwnership || got.Hasher != &hasher {
		t.Fatalf("preserve/hasher mismatch: %#v", got)
	}
	if len(got.Excludes) != 1 || got.Excludes[0] != "*.tmp" {
		t.Fatalf("excludes mismatch: %#v", got.Excludes)
	}
	if got.IgnoreFile != ".containerignore" || got.ContextDir != "/workspace" {
		t.Fatalf("context options mismatch: %#v", got)
	}
	if got.IDMappingOptions != idMap {
		t.Fatalf("id mapping pointer mismatch")
	}
	if !got.DryRun || !got.StripSetuidBit || !got.StripSetgidBit || !got.StripStickyBit {
		t.Fatalf("strip/dry-run flags mismatch: %#v", got)
	}
}

func TestInternalizeImageNameWithoutTag(t *testing.T) {
	if got := internalizeImageName("localhost/app"); got != "localhost/app-internal" {
		t.Fatalf("unexpected internal image name: %q", got)
	}
}
