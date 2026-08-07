package verdict

import "testing"

func TestClass_ZeroValueIsUnspecified(t *testing.T) {
	var c Class
	if c != ClassUnspecified {
		t.Fatalf("zero value of Class = %v, want ClassUnspecified", c)
	}
}

func TestClass_String(t *testing.T) {
	tests := []struct {
		name string
		c    Class
		want string
	}{
		{"unspecified", ClassUnspecified, "UNKNOWN_CLASS"},
		{"D", ClassD, "D"},
		{"S", ClassS, "S"},
		{"out of range", Class(99), "UNKNOWN_CLASS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Fatalf("Class(%d).String() = %q, want %q", tt.c, got, tt.want)
			}
		})
	}
}

func TestClass_DefaultSeverity(t *testing.T) {
	tests := []struct {
		name    string
		c       Class
		want    Severity
		wantErr bool
	}{
		{"D defaults to block", ClassD, SeverityBlock, false},
		{"S defaults to warn", ClassS, SeverityWarn, false},
		{"unspecified is an error", ClassUnspecified, SeverityUnspecified, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.DefaultSeverity()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("DefaultSeverity() error = nil, want an error for class %v", tt.c)
				}
				return
			}
			if err != nil {
				t.Fatalf("DefaultSeverity() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("DefaultSeverity() = %v, want %v", got, tt.want)
			}
		})
	}
}
