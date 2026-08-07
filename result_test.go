package verdict

import (
	"strings"
	"testing"
)

func TestResult_ZeroValueIsNotUsable(t *testing.T) {
	var r Result
	if r.CheckID() != "" {
		t.Fatalf("zero Result.CheckID() = %q, want empty", r.CheckID())
	}
	if r.Class() != ClassUnspecified {
		t.Fatalf("zero Result.Class() = %v, want ClassUnspecified", r.Class())
	}
	if r.Severity() != SeverityUnspecified {
		t.Fatalf("zero Result.Severity() = %v, want SeverityUnspecified", r.Severity())
	}
	if r.Outcome() != OutcomeIndeterminate {
		t.Fatalf("zero Result.Outcome() = %v, want OutcomeIndeterminate", r.Outcome())
	}
}

func TestNewResult(t *testing.T) {
	tests := []struct {
		name            string
		checkID         string
		class           Class
		severity        Severity
		outcome         Outcome
		wantErr         bool
		wantErrContains string // when set, the error message must contain this (e.g. the check ID, for correlation across ~40 call sites)
	}{
		{"class D at block passes", "MU-01", ClassD, SeverityBlock, OutcomeFail, false, ""},
		{"class D at warn passes (e.g. MU-13 partial FAIL)", "MU-13", ClassD, SeverityWarn, OutcomeFail, false, ""},
		{"class S at warn passes", "MU-20", ClassS, SeverityWarn, OutcomeIndeterminate, false, ""},
		{"empty check id rejected", "", ClassD, SeverityBlock, OutcomePass, true, ""},
		{"unspecified class rejected, error names the check", "MU-01", ClassUnspecified, SeverityBlock, OutcomePass, true, "MU-01"},
		{"out-of-range class rejected, error names the check", "MU-01", Class(99), SeverityBlock, OutcomePass, true, "MU-01"},
		{"unspecified severity rejected, error names the check", "MU-01", ClassD, SeverityUnspecified, OutcomePass, true, "MU-01"},
		{"out-of-range severity rejected, error names the check", "MU-01", ClassD, Severity(99), OutcomePass, true, "MU-01"},
		{"out-of-range outcome rejected, error names the check", "MU-01", ClassD, SeverityBlock, Outcome(99), true, "MU-01"},
		{"class S at block rejected -- promotion required, error names the check", "MU-20", ClassS, SeverityBlock, OutcomeFail, true, "MU-20"},
		{
			// Validity of outcome is checked before the class-S/block
			// promotion rule, so an invalid outcome is reported as such
			// even when the class/severity combination would also be
			// rejected -- the more fundamental defect is surfaced first.
			"invalid outcome takes priority over the promotion-required rejection",
			"MU-20", ClassS, SeverityBlock, Outcome(99), true, "invalid outcome",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewResult(tt.checkID, tt.class, tt.severity, tt.outcome)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewResult(%q, %v, %v, %v) error = nil, want an error", tt.checkID, tt.class, tt.severity, tt.outcome)
				}
				if r != (Result{}) {
					t.Fatalf("NewResult error path returned non-zero Result: %+v", r)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewResult(%q, %v, %v, %v) unexpected error: %v", tt.checkID, tt.class, tt.severity, tt.outcome, err)
			}
			if r.CheckID() != tt.checkID {
				t.Errorf("CheckID() = %q, want %q", r.CheckID(), tt.checkID)
			}
			if r.Class() != tt.class {
				t.Errorf("Class() = %v, want %v", r.Class(), tt.class)
			}
			if r.Severity() != tt.severity {
				t.Errorf("Severity() = %v, want %v", r.Severity(), tt.severity)
			}
			if r.Outcome() != tt.outcome {
				t.Errorf("Outcome() = %v, want %v", r.Outcome(), tt.outcome)
			}
		})
	}
}

func TestNewPromotedResult(t *testing.T) {
	tests := []struct {
		name            string
		checkID         string
		outcome         Outcome
		wantErr         bool
		wantErrContains string
	}{
		{"promoted class S check at block", "MU-20", OutcomeFail, false, ""},
		{"empty check id rejected", "", OutcomeFail, true, ""},
		{"out-of-range outcome rejected, error names the check", "MU-20", Outcome(99), true, "MU-20"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewPromotedResult(tt.checkID, tt.outcome)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NewPromotedResult(%q, %v) error = nil, want an error", tt.checkID, tt.outcome)
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewPromotedResult(%q, %v) unexpected error: %v", tt.checkID, tt.outcome, err)
			}
			if r.Class() != ClassS {
				t.Errorf("Class() = %v, want ClassS", r.Class())
			}
			if r.Severity() != SeverityBlock {
				t.Errorf("Severity() = %v, want SeverityBlock", r.Severity())
			}
			if r.Outcome() != tt.outcome {
				t.Errorf("Outcome() = %v, want %v", r.Outcome(), tt.outcome)
			}
		})
	}
}
