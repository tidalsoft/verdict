package verdict

import "testing"

func TestVerdict_ZeroValueIsDeny(t *testing.T) {
	var v Verdict
	if v != VerdictDeny {
		t.Fatalf("zero value of Verdict = %v, want VerdictDeny -- an uncomputed Aggregate must fail closed", v)
	}
}

func TestVerdict_String(t *testing.T) {
	tests := []struct {
		name string
		v    Verdict
		want string
	}{
		{"deny", VerdictDeny, "deny"},
		{"allow with warnings", VerdictAllowWithWarnings, "allow_with_warnings"},
		{"allow", VerdictAllow, "allow"},
		{"out of range", Verdict(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Fatalf("Verdict(%d).String() = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}

func TestDenyReason_ZeroValueIsNone(t *testing.T) {
	var d DenyReason
	if d != DenyReasonNone {
		t.Fatalf("zero value of DenyReason = %v, want DenyReasonNone", d)
	}
}

func TestDenyReason_String(t *testing.T) {
	tests := []struct {
		name string
		d    DenyReason
		want string
	}{
		{"none", DenyReasonNone, ""},
		{"insufficient evidence", DenyReasonInsufficientEvidence, "insufficient_evidence"},
		{"out of range", DenyReason(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Fatalf("DenyReason(%d).String() = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

// mustResult builds a Result directly (same-package literal, bypassing the
// constructors) so aggregate test vectors can freely combine class,
// severity, and outcome -- including combinations the constructors refuse,
// like ClassS+SeverityBlock (an already-promoted result) and out-of-range
// or zero-value fields that a corrupted/partially-initialized Result could
// carry despite the constructors never producing one.
func mustResult(checkID string, class Class, severity Severity, outcome Outcome) Result {
	return Result{checkID: checkID, class: class, severity: severity, outcome: outcome}
}

func TestComputeAggregate(t *testing.T) {
	tests := []struct {
		name    string
		results []Result
		mode    Mode
		want    Aggregate
		wantErr bool
	}{
		{
			name:    "no results allows",
			results: nil,
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictAllow},
		},
		{
			name:    "single pass allows",
			results: []Result{mustResult("MU-01", ClassD, SeverityBlock, OutcomePass)},
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictAllow},
		},
		{
			name:    "block fail denies",
			results: []Result{mustResult("MU-01", ClassD, SeverityBlock, OutcomeFail)},
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictDeny},
		},
		{
			name:    "warn fail allows with warnings",
			results: []Result{mustResult("MU-13", ClassD, SeverityWarn, OutcomeFail)},
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictAllowWithWarnings},
		},
		{
			name:    "class S warn fail allows with warnings -- class does not gate the aggregate rule",
			results: []Result{mustResult("MU-20", ClassS, SeverityWarn, OutcomeFail)},
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictAllowWithWarnings},
		},
		{
			name:    "block indeterminate under permissive mode allows -- never denies, never becomes a pass",
			results: []Result{mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate)},
			mode:    ModePermissive,
			want:    Aggregate{Verdict: VerdictAllow},
		},
		{
			name:    "block indeterminate under strict mode denies with insufficient_evidence",
			results: []Result{mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate)},
			mode:    ModeStrict,
			want:    Aggregate{Verdict: VerdictDeny, Reason: DenyReasonInsufficientEvidence},
		},
		{
			name:    "warn indeterminate under strict mode still allows -- only block severity triggers strict denial",
			results: []Result{mustResult("MU-20", ClassS, SeverityWarn, OutcomeIndeterminate)},
			mode:    ModeStrict,
			want:    Aggregate{Verdict: VerdictAllow},
		},
		{
			name:    "promoted class S check indeterminate at block under strict mode denies just like class D",
			results: []Result{mustResult("MU-20", ClassS, SeverityBlock, OutcomeIndeterminate)},
			mode:    ModeStrict,
			want:    Aggregate{Verdict: VerdictDeny, Reason: DenyReasonInsufficientEvidence},
		},
		{
			name: "block fail and warn fail together deny (block wins)",
			results: []Result{
				mustResult("MU-01", ClassD, SeverityBlock, OutcomeFail),
				mustResult("MU-13", ClassD, SeverityWarn, OutcomeFail),
			},
			mode: ModePermissive,
			want: Aggregate{Verdict: VerdictDeny},
		},
		{
			name: "block fail plus strict block indeterminate denies with no reason -- the real FAIL is the cause",
			results: []Result{
				mustResult("MU-01", ClassD, SeverityBlock, OutcomeFail),
				mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate),
			},
			mode: ModeStrict,
			want: Aggregate{Verdict: VerdictDeny},
		},
		{
			name: "strict block indeterminate outranks a warn fail",
			results: []Result{
				mustResult("MU-13", ClassD, SeverityWarn, OutcomeFail),
				mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate),
			},
			mode: ModeStrict,
			want: Aggregate{Verdict: VerdictDeny, Reason: DenyReasonInsufficientEvidence},
		},
		{
			name: "pass and permissive indeterminate together still allow",
			results: []Result{
				mustResult("MU-01", ClassD, SeverityBlock, OutcomePass),
				mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate),
			},
			mode: ModePermissive,
			want: Aggregate{Verdict: VerdictAllow},
		},
		{
			name: "mixed pass, warn fail, and permissive indeterminate allows with warnings",
			results: []Result{
				mustResult("MU-01", ClassD, SeverityBlock, OutcomePass),
				mustResult("MU-13", ClassD, SeverityWarn, OutcomeFail),
				mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate),
			},
			mode: ModePermissive,
			want: Aggregate{Verdict: VerdictAllowWithWarnings},
		},
		{
			name:    "invalid mode is rejected, not silently treated as permissive",
			results: []Result{mustResult("PG-05", ClassD, SeverityBlock, OutcomeIndeterminate)},
			mode:    Mode(2),
			wantErr: true,
		},
		{
			name:    "negative mode is rejected",
			results: nil,
			mode:    Mode(-1),
			wantErr: true,
		},
		{
			name:    "a FAIL with unspecified severity is rejected, not silently dropped from the aggregate",
			results: []Result{mustResult("MU-01", ClassD, SeverityUnspecified, OutcomeFail)},
			mode:    ModeStrict,
			wantErr: true,
		},
		{
			name:    "a FAIL with an out-of-range severity is rejected",
			results: []Result{mustResult("MU-01", ClassD, Severity(99), OutcomeFail)},
			mode:    ModePermissive,
			wantErr: true,
		},
		{
			name:    "an INDETERMINATE with unspecified severity is rejected, not silently excluded from strict denial",
			results: []Result{mustResult("PG-05", ClassD, SeverityUnspecified, OutcomeIndeterminate)},
			mode:    ModeStrict,
			wantErr: true,
		},
		{
			name:    "an out-of-range outcome is rejected",
			results: []Result{mustResult("MU-01", ClassD, SeverityBlock, Outcome(99))},
			mode:    ModePermissive,
			wantErr: true,
		},
		{
			name: "a slice of ten zero-value Results is rejected outright in strict mode, not read as a clean allow",
			// Reproduces the exact shape make([]Result, 10) would produce:
			// every element unfilled, i.e. Outcome zero (Indeterminate) and
			// Severity zero (Unspecified).
			results: make([]Result, 10),
			mode:    ModeStrict,
			wantErr: true,
		},
		{
			name: "one valid result followed by a corrupted one is still rejected",
			results: []Result{
				mustResult("MU-01", ClassD, SeverityBlock, OutcomePass),
				mustResult("MU-02", ClassD, Severity(7), OutcomeFail),
			},
			mode:    ModePermissive,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ComputeAggregate(tt.results, tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ComputeAggregate(%+v, %v) error = nil, want an error", tt.results, tt.mode)
				}
				if got != (Aggregate{}) {
					t.Fatalf("ComputeAggregate error path returned non-zero Aggregate: %+v", got)
				}
				if got.Verdict != VerdictDeny {
					t.Fatalf("zero Aggregate on error path has Verdict = %v, want VerdictDeny (fail closed)", got.Verdict)
				}
				return
			}
			if err != nil {
				t.Fatalf("ComputeAggregate(%+v, %v) unexpected error: %v", tt.results, tt.mode, err)
			}
			if got != tt.want {
				t.Fatalf("ComputeAggregate(%+v, %v) = %+v, want %+v", tt.results, tt.mode, got, tt.want)
			}
		})
	}
}
