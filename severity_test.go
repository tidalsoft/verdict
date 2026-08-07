package verdict

import "testing"

func TestSeverity_ZeroValueIsUnspecified(t *testing.T) {
	var s Severity
	if s != SeverityUnspecified {
		t.Fatalf("zero value of Severity = %v, want SeverityUnspecified", s)
	}
}

func TestSeverity_String(t *testing.T) {
	tests := []struct {
		name string
		s    Severity
		want string
	}{
		{"unspecified", SeverityUnspecified, "unspecified"},
		{"warn", SeverityWarn, "warn"},
		{"block", SeverityBlock, "block"},
		{"out of range", Severity(99), "unspecified"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Fatalf("Severity(%d).String() = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}
