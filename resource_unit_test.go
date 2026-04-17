package podbridge5

import (
	"reflect"
	"testing"

	"github.com/opencontainers/runtime-spec/specs-go"
)

func TestResourceOptions(t *testing.T) {
	tests := []struct {
		name            string
		opts            []ContainerOptions
		wantResources   *specs.LinuxResources
		wantOOMScoreAdj *int
	}{
		{
			name: "Only CPU limits",
			opts: []ContainerOptions{
				WithCPULimits(50000, 100000, 1024),
			},
			wantResources: &specs.LinuxResources{
				CPU: &specs.LinuxCPU{
					Quota:  ptrInt64(50000),
					Period: ptrUint64(100000),
					Shares: ptrUint64(1024),
				},
			},
			wantOOMScoreAdj: nil,
		},
		{
			name: "Only memory limit",
			opts: []ContainerOptions{
				WithMemoryLimit(256 * 1024 * 1024),
			},
			wantResources: &specs.LinuxResources{
				Memory: &specs.LinuxMemory{
					Limit: ptrInt64(256 * 1024 * 1024),
				},
			},
			wantOOMScoreAdj: nil,
		},
		{
			name: "Only OOM score adj",
			opts: []ContainerOptions{
				WithOOMScoreAdj(-500),
			},
			wantResources:   nil,
			wantOOMScoreAdj: ptrInt(-500),
		},
		{
			name: "All combined",
			opts: []ContainerOptions{
				WithCPULimits(20000, 50000, 512),
				WithMemoryLimit(128 * 1024 * 1024),
				WithOOMScoreAdj(100),
			},
			wantResources: &specs.LinuxResources{
				CPU: &specs.LinuxCPU{
					Quota:  ptrInt64(20000),
					Period: ptrUint64(50000),
					Shares: ptrUint64(512),
				},
				Memory: &specs.LinuxMemory{
					Limit: ptrInt64(128 * 1024 * 1024),
				},
			},
			wantOOMScoreAdj: ptrInt(100),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			spec, err := NewSpec(tt.opts...)
			if err != nil {
				t.Fatalf("NewSpec(%v) error: %v", tt.opts, err)
			}

			if tt.wantResources == nil {
				if spec.ResourceLimits != nil {
					t.Errorf("expected no ResourceLimits, got %+v", spec.ResourceLimits)
				}
			} else {
				if spec.ResourceLimits == nil {
					t.Fatalf("expected ResourceLimits, got nil")
				}
				if !reflect.DeepEqual(spec.ResourceLimits, tt.wantResources) {
					t.Errorf("ResourceLimits mismatch:\ngot  %+v\nwant %+v", spec.ResourceLimits, tt.wantResources)
				}
			}

			if tt.wantOOMScoreAdj == nil {
				if spec.OOMScoreAdj != nil {
					t.Errorf("expected no OOMScoreAdj, got %v", *spec.OOMScoreAdj)
				}
			} else {
				if spec.OOMScoreAdj == nil {
					t.Fatalf("expected OOMScoreAdj, got nil")
				}
				if *spec.OOMScoreAdj != *tt.wantOOMScoreAdj {
					t.Errorf("OOMScoreAdj = %d, want %d", *spec.OOMScoreAdj, *tt.wantOOMScoreAdj)
				}
			}
		})
	}
}
