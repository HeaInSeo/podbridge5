package podbridge5

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewConfigDefaultsAndSetters(t *testing.T) {
	cfg := NewConfig("docker.io/library/alpine:3.20")

	if cfg.Image.SourceImageName != "docker.io/library/alpine:3.20" {
		t.Fatalf("source image mismatch: %q", cfg.Image.SourceImageName)
	}
	if cfg.Image.ImageName != "docker.io/library/alpine-internal:3.20" {
		t.Fatalf("internal image mismatch: %q", cfg.Image.ImageName)
	}
	if cfg.Image.WorkDir != ContainerAppDir || cfg.Container.WorkDir != ContainerAppDir {
		t.Fatalf("default workdir mismatch: image=%q container=%q", cfg.Image.WorkDir, cfg.Container.WorkDir)
	}
	if len(cfg.Image.CMD) == 0 || len(cfg.Container.Cmd) == 0 {
		t.Fatalf("default command was not set")
	}

	cfg.SetSourceImageName("base")
	cfg.SetImageName("target")
	cfg.SetDirectories([]string{"/a", "/b"})
	cfg.SetScriptMap(map[string][]string{"/a": {"run.sh"}})
	cfg.SetPermissionFiles([]string{"/a/run.sh"})
	cfg.SetWorkDir("/work")
	cfg.SetCMD([]string{"run"})

	if cfg.Image.SourceImageName != "base" || cfg.Image.ImageName != "target" {
		t.Fatalf("image setters failed: %#v", cfg.Image)
	}
	if strings.Join(cfg.Image.Directories, ",") != "/a,/b" || strings.Join(cfg.Container.Directories, ",") != "/a,/b" {
		t.Fatalf("directory setter did not update both sections")
	}
	if cfg.Image.ScriptMap["/a"][0] != "run.sh" || cfg.Container.ScriptMap["/a"][0] != "run.sh" {
		t.Fatalf("script map setter did not update both sections")
	}
	if cfg.Image.PermissionFiles[0] != "/a/run.sh" || cfg.Container.PermissionFiles[0] != "/a/run.sh" {
		t.Fatalf("permission setter did not update both sections")
	}
	if cfg.Image.WorkDir != "/work" || cfg.Container.WorkDir != "/work" {
		t.Fatalf("workdir setter did not update both sections")
	}
	if cfg.Image.CMD[0] != "run" || cfg.Container.Cmd[0] != "run" {
		t.Fatalf("cmd setter did not update both sections")
	}

	cfg.SetSourceImageNameAndImageName("registry.local/app:dev")
	if cfg.Image.SourceImageName != "registry.local/app:dev" {
		t.Fatalf("source image not updated: %q", cfg.Image.SourceImageName)
	}
	if cfg.Image.ImageName != "registry.local/app-internal:dev" {
		t.Fatalf("internal image not updated: %q", cfg.Image.ImageName)
	}
}

func TestNewConfigFromFileFailuresAndSuccess(t *testing.T) {
	if _, err := NewConfigFromFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected missing config file error")
	}

	badPath := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(badPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("write bad config: %v", err)
	}
	if _, err := NewConfigFromFile(badPath); err == nil {
		t.Fatal("expected invalid JSON error")
	}

	goodPath := filepath.Join(t.TempDir(), "good.json")
	data := []byte(`{"image":{"sourceImageName":"alpine","imageName":"target"},"container":{"executorShell":"exec.sh"}}`)
	if err := os.WriteFile(goodPath, data, 0o644); err != nil {
		t.Fatalf("write good config: %v", err)
	}
	cfg, err := NewConfigFromFile(goodPath)
	if err != nil {
		t.Fatalf("NewConfigFromFile returned error: %v", err)
	}
	if cfg.Image.SourceImageName != "alpine" || cfg.Image.ImageName != "target" {
		t.Fatalf("decoded config mismatch: %#v", cfg.Image)
	}
}
