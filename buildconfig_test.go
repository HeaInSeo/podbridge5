package podbridge5

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInternalizeImageName(t *testing.T) {

	// 테스트 입력: 소스 이미지 이름
	sourceImage := "docker.io/library/alpine:latest"

	// 예상되는 내부 이미지 이름은 internalizeImageName 함수의 결과와 동일해야 함.
	// 예를 들어, "docker.io/library/alpine:latest" -> "docker.io/library/alpine-internal:latest"
	expectedImageName := internalizeImageName(sourceImage)
	correctImageName := "docker.io/library/alpine-internal:latest"
	if expectedImageName != correctImageName {
		t.Errorf("Expected ImageName to be %q, got %q", expectedImageName, correctImageName)
	}
	Log.Printf("Expected ImageName to be %q, got %q", expectedImageName, correctImageName)
}

func TestNewConfigFromFileErrorPaths(t *testing.T) {
	if _, err := NewConfigFromFile(""); err == nil {
		t.Fatal("expected empty path error")
	}

	dir := t.TempDir()
	if _, err := NewConfigFromFile(filepath.Join(dir, "missing.json")); err == nil {
		t.Fatal("expected missing file error")
	}

	badJSON := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{not json}"), 0o644); err != nil {
		t.Fatalf("write bad json: %v", err)
	}
	if _, err := NewConfigFromFile(badJSON); err == nil {
		t.Fatal("expected JSON decode error")
	}

	good := filepath.Join(dir, "good.json")
	cfg := BuildConfig{}
	data, _ := json.Marshal(cfg)
	if err := os.WriteFile(good, data, 0o644); err != nil {
		t.Fatalf("write good json: %v", err)
	}
	if _, err := NewConfigFromFile(good); err != nil {
		t.Fatalf("unexpected error on valid config: %v", err)
	}
}

func TestSetSourceImageNameAndImageName(t *testing.T) {
	// 새로운 Config 생성
	config, err := NewConfigFromFile("config.json")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}
	sourceImageName := "docker.io/library/alpine:latest"
	config.SetSourceImageNameAndImageName(sourceImageName)

	expectedImageName := internalizeImageName(sourceImageName)
	if config.Image.SourceImageName != sourceImageName {
		t.Errorf("Expected SourceImageName to be %q, but got %q", sourceImageName, config.Image.SourceImageName)
	}
	if config.Image.ImageName != expectedImageName {
		t.Errorf("Expected ImageName to be %q, but got %q", expectedImageName, config.Image.ImageName)
	}

	Log.Printf("SourceImageName: %q, ImageName: %q", config.Image.SourceImageName, config.Image.ImageName)
}
