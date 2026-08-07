package verdict

import "testing"

func TestMode_ZeroValueIsPermissive(t *testing.T) {
	var m Mode
	if m != ModePermissive {
		t.Fatalf("zero value of Mode = %v, want ModePermissive (the documented default)", m)
	}
}

func TestMode_String(t *testing.T) {
	tests := []struct {
		name string
		m    Mode
		want string
	}{
		{"permissive", ModePermissive, "permissive"},
		{"strict", ModeStrict, "strict"},
		{"out of range", Mode(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.m.String(); got != tt.want {
				t.Fatalf("Mode(%d).String() = %q, want %q", tt.m, got, tt.want)
			}
		})
	}
}
