//go:build integration

package podbridge5

import (
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"testing"
)

// setupOverlay mounts and returns lower, upper, work, merged dirs, and a cleanup function.
func setupOverlay(t *testing.T) (lower, upper, work, merged string, cleanup func()) {
	t.Helper()

	base := t.TempDir()
	lower = filepath.Join(base, "lower")
	upper = filepath.Join(base, "upper")
	work = filepath.Join(base, "work")
	merged = filepath.Join(base, "merged")

	// Prepare input and directories
	if err := os.MkdirAll(lower, 0755); err != nil {
		t.Fatalf("failed to create lower dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(lower, "input.txt"), []byte("original"), 0644); err != nil {
		t.Fatalf("failed to write input file: %v", err)
	}

	// Mount overlay; skip test if not supported
	if err := MountOverlay(lower, upper, work, merged); err != nil {
		t.Skipf("Skipping test: Overlay mount not supported: %v", err)
	}

	return lower, upper, work, merged, func() {
		_ = unix.Unmount(merged, 0)
	}
}

// TestCaseA_Output writes to /out under merged and verifies upper captures it.
func TestCaseA_Output(t *testing.T) {
	_, upper, _, merged, cleanup := setupOverlay(t)
	defer cleanup()

	// Simulate tool writing to /app/data/out
	outDir := filepath.Join(merged, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatalf("failed to create outDir: %v", err)
	}
	result := []byte("data")
	if err := os.WriteFile(filepath.Join(outDir, "result.txt"), result, 0644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	// Verify result in upper
	upperFile := filepath.Join(upper, "out", "result.txt")
	data, err := os.ReadFile(upperFile)
	if err != nil {
		t.Fatalf("expected file in upper, got error: %v", err)
	}
	if string(data) != string(result) {
		t.Errorf("upper result mismatch, got %s", data)
	}
}

// TestCaseB1_Sidecar creates a sidecar file next to input and verifies only upper contains it.
func TestCaseB1_Sidecar(t *testing.T) {
	lower, upper, _, merged, cleanup := setupOverlay(t)
	defer cleanup()

	sideFile := filepath.Join(merged, "index.bai")
	content := []byte("bai")
	if err := os.WriteFile(sideFile, content, 0644); err != nil {
		t.Fatalf("failed to write sidecar file: %v", err)
	}

	// Lower should not have it
	if _, err := os.Stat(filepath.Join(lower, "index.bai")); !os.IsNotExist(err) {
		t.Errorf("lower should not contain sidecar, err: %v", err)
	}
	// Upper should have it
	got, err := os.ReadFile(filepath.Join(upper, "index.bai"))
	if err != nil {
		t.Fatalf("upper missing sidecar: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("upper sidecar content mismatch, got %s", got)
	}
}

// TestCaseB2_Transparency ensures reads come from lower and writes to merged see correct separate layers.
func TestCaseB2_Transparency(t *testing.T) {
	lower, upper, _, merged, cleanup := setupOverlay(t)
	defer cleanup()

	// Verify read sees original
	data, err := os.ReadFile(filepath.Join(merged, "input.txt"))
	if err != nil {
		t.Fatalf("failed to read merged input: %v", err)
	}
	if string(data) != "original" {
		t.Errorf("merged read mismatch, got %s", data)
	}

	// Write new file and verify it goes to upper only
	newFile := filepath.Join(merged, "new.txt")
	if err := os.WriteFile(newFile, []byte("newdata"), 0644); err != nil {
		t.Fatalf("failed to write new file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(upper, "new.txt")); err != nil {
		t.Errorf("upper should have new file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(lower, "new.txt")); !os.IsNotExist(err) {
		t.Errorf("lower should not have new file, err: %v", err)
	}
}

// TestCaseC_ModifyInput modifies existing file and verifies diff stored in upper only.
func TestCaseC_ModifyInput(t *testing.T) {
	lower, _, _, merged, cleanup := setupOverlay(t)
	defer cleanup()

	mergedInput := filepath.Join(merged, "input.txt")
	if err := os.WriteFile(mergedInput, []byte("modified"), 0644); err != nil {
		t.Fatalf("failed to modify merged input: %v", err)
	}
	// Lower should stay original
	baseData, err := os.ReadFile(filepath.Join(lower, "input.txt"))
	if err != nil {
		t.Fatalf("failed to read lower input: %v", err)
	}
	if string(baseData) != "original" {
		t.Errorf("lower modified, expected original, got %s", baseData)
	}
	// Merged read should reflect modification
	modData, err := os.ReadFile(mergedInput)
	if err != nil {
		t.Fatalf("failed to read merged input: %v", err)
	}
	if string(modData) != "modified" {
		t.Errorf("merged input not modified, got %s", modData)
	}
}
