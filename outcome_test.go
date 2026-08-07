package engine

import "testing"

func TestOutcome_ZeroValueIsIndeterminate(t *testing.T) {
	var o Outcome
	if o != OutcomeIndeterminate {
		t.Fatalf("zero value of Outcome = %v, want OutcomeIndeterminate -- INDETERMINATE must never collapse to PASS", o)
	}
}

func TestOutcome_String(t *testing.T) {
	tests := []struct {
		name string
		o    Outcome
		want string
	}{
		{"indeterminate", OutcomeIndeterminate, "INDETERMINATE"},
		{"pass", OutcomePass, "PASS"},
		{"fail", OutcomeFail, "FAIL"},
		{"out of range", Outcome(99), "UNKNOWN_OUTCOME"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.String(); got != tt.want {
				t.Fatalf("Outcome(%d).String() = %q, want %q", tt.o, got, tt.want)
			}
		})
	}
}

func TestOutcome_Valid(t *testing.T) {
	tests := []struct {
		name string
		o    Outcome
		want bool
	}{
		{"indeterminate", OutcomeIndeterminate, true},
		{"pass", OutcomePass, true},
		{"fail", OutcomeFail, true},
		{"out of range", Outcome(99), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.o.valid(); got != tt.want {
				t.Fatalf("Outcome(%d).valid() = %v, want %v", tt.o, got, tt.want)
			}
		})
	}
}
