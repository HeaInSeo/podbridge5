package podbridge5

import (
	"fmt"
	"os"
	"testing"
)

func TestSocketDirectoryForCurrentUser(t *testing.T) {
	uid := os.Getuid()
	socketDir := fmt.Sprintf("/run/user/%d", uid)
	t.Logf("Expected socket directory: %s", socketDir)

	info, err := os.Stat(socketDir)
	if os.IsNotExist(err) {
		t.Skipf("Socket directory %q does not exist, skipping test", socketDir)
	} else if err != nil {
		t.Fatalf("Failed to stat %q: %v", socketDir, err)
	}

	if !info.IsDir() {
		t.Fatalf("Path %q exists but is not a directory", socketDir)
	}
}
