package podbridge5

import "testing"

func TestNewVolumeWriterSpec(t *testing.T) {
	spec, err := newVolumeWriterSpec("demo-volume", "/data")
	if err != nil {
		t.Fatalf("newVolumeWriterSpec() error = %v", err)
	}

	if spec.Image != volumeTransferImage {
		t.Fatalf("spec.Image = %q, want %q", spec.Image, volumeTransferImage)
	}
	if spec.Name != volumeWriterContainerName {
		t.Fatalf("spec.Name = %q, want %q", spec.Name, volumeWriterContainerName)
	}
	if got := spec.Env["MOUNT"]; got != "/data" {
		t.Fatalf("spec.Env[MOUNT] = %q, want /data", got)
	}
	if len(spec.Command) != 3 {
		t.Fatalf("len(spec.Command) = %d, want 3", len(spec.Command))
	}
	if spec.Command[2] != "mkdir -p \"$MOUNT\"; exec tail -f /dev/null" {
		t.Fatalf("spec.Command[2] = %q", spec.Command[2])
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("len(spec.Volumes) = %d, want 1", len(spec.Volumes))
	}
	if spec.Volumes[0].Name != "demo-volume" || spec.Volumes[0].Dest != "/data" {
		t.Fatalf("unexpected volume mapping: %+v", spec.Volumes[0])
	}
}

func TestNewVolumeReaderSpec(t *testing.T) {
	spec, err := newVolumeReaderSpec("demo-volume", "/cache")
	if err != nil {
		t.Fatalf("newVolumeReaderSpec() error = %v", err)
	}

	if spec.Image != volumeTransferImage {
		t.Fatalf("spec.Image = %q, want %q", spec.Image, volumeTransferImage)
	}
	if spec.Name != volumeReaderContainerName {
		t.Fatalf("spec.Name = %q, want %q", spec.Name, volumeReaderContainerName)
	}
	if len(spec.Command) != 3 {
		t.Fatalf("len(spec.Command) = %d, want 3", len(spec.Command))
	}
	if spec.Command[2] != "mkdir -p /data && sleep infinity" {
		t.Fatalf("spec.Command[2] = %q", spec.Command[2])
	}
	if len(spec.Volumes) != 1 {
		t.Fatalf("len(spec.Volumes) = %d, want 1", len(spec.Volumes))
	}
	if spec.Volumes[0].Name != "demo-volume" || spec.Volumes[0].Dest != "/cache" {
		t.Fatalf("unexpected volume mapping: %+v", spec.Volumes[0])
	}
}
