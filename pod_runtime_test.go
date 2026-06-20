//go:build runtime

package podbridge5

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestCreateAndRemovePod(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	podName := "test-pod-" + uuid.New().String()
	spec, err := NewPodSpec(WithPodName(podName), WithPodNoInfra(true))
	if err != nil {
		t.Fatalf("NewPodSpec() failed: %v", err)
	}

	podID, err := CreatePod(ctx, spec)
	if err != nil {
		t.Fatalf("CreatePod() failed: %v", err)
	}
	if podID == "" {
		t.Fatal("CreatePod() returned empty pod ID")
	}

	if err := RemovePod(ctx, podID, true); err != nil {
		t.Fatalf("RemovePod() failed: %v", err)
	}
}

func TestRemovePod_NonExistent(t *testing.T) {
	ctx, err := NewConnectionLinux5(context.Background())
	if err != nil {
		t.Fatalf("NewConnectionLinux5() failed: %v", err)
	}

	if err := RemovePod(ctx, "no-such-pod-"+uuid.New().String(), true); err == nil {
		t.Fatal("expected error removing a non-existent pod, got nil")
	}
}
