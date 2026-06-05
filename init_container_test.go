package podbridge5

import "testing"

func TestOverlayMountOptions(t *testing.T) {
	got, err := overlayMountOptions("/lower", "/upper", "/work")
	if err != nil {
		t.Fatalf("overlayMountOptions() error = %v", err)
	}
	want := "lowerdir=/lower,upperdir=/upper,workdir=/work"
	if got != want {
		t.Fatalf("overlayMountOptions() = %q, want %q", got, want)
	}
}

func TestOverlayMountOptions_RejectsInvalidPath(t *testing.T) {
	if _, err := overlayMountOptions("/lower", "/bad,path", "/work"); err == nil {
		t.Fatal("overlayMountOptions() error = nil, want validation error")
	}
}

func TestValidateOverlayPathArg(t *testing.T) {
	tests := []struct {
		name  string
		value string
		ok    bool
	}{
		{name: "valid", value: "/tmp/data", ok: true},
		{name: "empty", value: "   ", ok: false},
		{name: "comma", value: "/tmp,a", ok: false},
		{name: "newline", value: "/tmp/\n", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateOverlayPathArg("path", tc.value)
			if tc.ok && err != nil {
				t.Fatalf("validateOverlayPathArg() error = %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("validateOverlayPathArg() error = nil, want validation error")
			}
		})
	}
}
