//go:build runtime

package podbridge5

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/bindings/system"
)

func TestNewConnectionLinux5(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}
	t.Log("Podman connection established")

	verOpts := &system.VersionOptions{}
	verReport, err := system.Version(ctx, verOpts)
	if err != nil {
		t.Logf("Warning: could not retrieve Podman version: %v", err)
		return
	}

	if verReport.Server != nil {
		t.Logf("Podman server version: %s", verReport.Server.Version)
	} else {
		t.Log("Podman server version: <nil>")
	}

	if verReport.Client != nil {
		t.Logf("Podman client version: %s", verReport.Client.Version)
	} else {
		t.Log("Podman client version: <nil>")
	}
}
