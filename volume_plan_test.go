package podbridge5

import "testing"

func TestPlanVolumeSetup(t *testing.T) {
	tests := []struct {
		name      string
		mode      VolumeMode
		exists    bool
		want      volumeSetupAction
		wantError bool
	}{
		{name: "overwrite ignores existing state when missing", mode: ModeOverwrite, exists: false, want: volumeSetupActionOverwrite},
		{name: "overwrite ignores existing state when present", mode: ModeOverwrite, exists: true, want: volumeSetupActionOverwrite},
		{name: "skip on existing volume", mode: ModeSkip, exists: true, want: volumeSetupActionSkip},
		{name: "skip creates when missing", mode: ModeSkip, exists: false, want: volumeSetupActionCreate},
		{name: "update reuses when present", mode: ModeUpdate, exists: true, want: volumeSetupActionReuse},
		{name: "update creates when missing", mode: ModeUpdate, exists: false, want: volumeSetupActionCreate},
		{name: "unknown mode", mode: VolumeMode(99), exists: true, wantError: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := planVolumeSetup(tc.mode, tc.exists)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("unexpected action: got %v want %v", got, tc.want)
			}
		})
	}
}
