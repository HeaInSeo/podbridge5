//go:build runtime

package podbridge5

import (
	"os"
	"testing"

	"go.podman.io/storage/pkg/reexec"
)

// TestMain registers the reexec handlers that buildah/containers-storage use
// to run chrooted subprocesses (e.g. resolving users inside a build context).
// Without this, any test path that triggers Builder.Run panics with
// "reexec.Init() was not called in main()".
func TestMain(m *testing.M) {
	if reexec.Init() {
		return
	}
	os.Exit(m.Run())
}
