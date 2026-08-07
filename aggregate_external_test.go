package verdict_test

import (
	"strings"
	"testing"

	"github.com/tidalsoft/verdict"
)

// TestComputeAggregate_ExternalReproductions lives in the black-box
// verdict_test package -- built only from the exported API, exactly as a
// real caller (a check package, the response assembler, a CLI command)
// would use it -- and reproduces the two findings from adversarial review:
// an unrecognised Mode silently degrading to permissive, and a slice of
// never-filled-in Results silently reading as a clean allow. Both must now
// surface as an error rather than a verdict.
func TestComputeAggregate_ExternalReproductions(t *testing.T) {
	t.Run("unrecognised mode no longer silently degrades to permissive", func(t *testing.T) {
		r, err := verdict.NewResult("PG-05", verdict.ClassD, verdict.SeverityBlock, verdict.OutcomeIndeterminate)
		if err != nil {
			t.Fatalf("NewResult: unexpected error: %v", err)
		}
		agg, err := verdict.ComputeAggregate([]verdict.Result{r}, verdict.Mode(2))
		if err == nil {
			t.Fatalf("ComputeAggregate with Mode(2) error = nil, want an error; got Aggregate %+v", agg)
		}
		if !strings.Contains(err.Error(), "invalid mode") {
			t.Fatalf("error = %q, want it to mention the invalid mode", err.Error())
		}
		if agg.Verdict != verdict.VerdictDeny {
			t.Fatalf("zero Aggregate on the error path has Verdict = %v, want VerdictDeny (fail closed)", agg.Verdict)
		}
	})

	t.Run("zero-value results no longer silently read as allow under strict mode", func(t *testing.T) {
		// The exact shape a bug like make([]verdict.Result, n) with
		// elements never filled in would produce: every element has
		// Outcome zero (Indeterminate) and Severity zero (Unspecified).
		results := make([]verdict.Result, 10)
		agg, err := verdict.ComputeAggregate(results, verdict.ModeStrict)
		if err == nil {
			t.Fatalf("ComputeAggregate with zero-value results error = nil, want an error; got Aggregate %+v", agg)
		}
		if agg.Verdict != verdict.VerdictDeny {
			t.Fatalf("zero Aggregate on the error path has Verdict = %v, want VerdictDeny (fail closed)", agg.Verdict)
		}
	})
}
