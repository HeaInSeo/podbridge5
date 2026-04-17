package podbridge5

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type fakeImageExportRuntime struct {
	imageID string
	data    []byte
	err     error
}

func (f *fakeImageExportRuntime) ExportImage(_ context.Context, imageID string, writer io.Writer) error {
	f.imageID = imageID
	if f.err != nil {
		return f.err
	}
	_, err := writer.Write(f.data)
	return err
}

func TestImageArchivePath(t *testing.T) {
	got := imageArchivePath("/tmp/out", "docker.io/library/alpine:latest", false)
	want := filepath.Join("/tmp/out", "alpine-latest.tar")
	if got != want {
		t.Fatalf("imageArchivePath() = %q, want %q", got, want)
	}
}

func TestPrepareImageArchiveWriter(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "images", "demo.tar")
	writer, closer, err := prepareImageArchiveWriter(archivePath, false)
	if err != nil {
		t.Fatalf("prepareImageArchiveWriter() error = %v", err)
	}
	if _, err := writer.Write([]byte("demo")); err != nil {
		t.Fatalf("writer.Write() error = %v", err)
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("closer.Close() error = %v", err)
	}
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != "demo" {
		t.Fatalf("archive contents = %q", string(data))
	}
}

func TestPrepareImageArchiveWriterCompressed(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "images", "demo.tar.gz")
	writer, closer, err := prepareImageArchiveWriter(archivePath, true)
	if err != nil {
		t.Fatalf("prepareImageArchiveWriter() error = %v", err)
	}
	if _, err := writer.Write([]byte("demo")); err != nil {
		t.Fatalf("writer.Write() error = %v", err)
	}
	if err := closer.Close(); err != nil {
		t.Fatalf("closer.Close() error = %v", err)
	}
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v", err)
	}
	defer gzr.Close()
	unzipped, err := io.ReadAll(gzr)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(unzipped) != "demo" {
		t.Fatalf("unzipped contents = %q", string(unzipped))
	}
}

func TestSaveImageWithRuntime(t *testing.T) {
	runtime := &fakeImageExportRuntime{data: []byte("exported-data")}
	root := t.TempDir()
	if err := saveImageWithRuntime(context.Background(), runtime, root, "docker.io/library/alpine:latest", "img-1", false); err != nil {
		t.Fatalf("saveImageWithRuntime() error = %v", err)
	}
	if runtime.imageID != "img-1" {
		t.Fatalf("runtime imageID = %q", runtime.imageID)
	}
	archivePath := filepath.Join(root, "alpine-latest.tar")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if string(data) != "exported-data" {
		t.Fatalf("archive contents = %q", string(data))
	}
}

func TestSaveImageWithRuntimePropagatesExportError(t *testing.T) {
	runtime := &fakeImageExportRuntime{err: errors.New("export failed")}
	err := saveImageWithRuntime(context.Background(), runtime, t.TempDir(), "docker.io/library/alpine:latest", "img-1", false)
	if err == nil || err.Error() != "export failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}
