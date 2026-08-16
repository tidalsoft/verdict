package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// mu20MinObservations is SPEC-MU §6 MU-20's own floor: "at least 200
// recorded observations for the field. Below that -> INDETERMINATE." See
// Input.Observations' doc comment (mu.go) for the reading this package
// takes of "observation" -- one entry of Input.Observations -- since
// SPEC-MU never defines the term itself.
const mu20MinObservations = 200

// mu20Threshold is the default modified z-score threshold SPEC-MU §6
// states for MU-20: "exceeds the configured threshold (default 8)". This
// package implements only the default; a per-field configured threshold
// is a ruleset-resolution concern outside this pure evaluation function,
// exactly as a Class S check's promotion to block severity is (see
// statResult's own doc comment).
func mu20Threshold() decimal.Decimal { return mustParseDecimal("8") }

// mu20ZConstant is SPEC-MU §6's fixed constant in the modified z-score
// formula: "modified_z = 0.6745 * (x - median) / MAD".
func mu20ZConstant() decimal.Decimal { return mustParseDecimal("0.6745") }

// checkMU20 implements the magnitude_outlier check (MU-20, SPEC-MU §6):
// flag a value far outside the observed distribution for its field, by
// modified z-score against the field's rolling median and median absolute
// deviation (MAD).
//
// Applicability (SPEC-MU §2.5.1: applies to money, decimal, percentage,
// and quantity, with no further gate):
//   - no declaration for the field, or a declaration of a kind this check
//     does not apply to (timestamp, identifier) -> not applicable.
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS or FAIL:
//   - the value is not coercible (§2.6.3; MU-20 is value-dependent) ->
//     INDETERMINATE, reason value_not_coercible.
//   - fewer than mu20MinObservations recorded observations -> INDETERMINATE
//     (vector MU-V38).
//   - MAD is zero, the *Degenerate case* SPEC-MU §6 names explicitly: "An
//     engine dividing by zero here will flag every non-identical value,
//     which is the worst possible behaviour" -> INDETERMINATE, never FAIL
//     (vector MU-V39).
//   - median, MAD, or the z-score comparison itself overflows the exact
//     decimal range on some legitimately parseable but extreme window or
//     value -> INDETERMINATE for MU-20 alone, never an aborted evaluation
//     (SPEC-MU §2.6 does not let one check's arithmetic failure discard
//     its siblings' results).
//   - abs(modified_z) exceeds mu20Threshold -> FAIL at warn (vector
//     MU-V86: this test is two-sided, so a value far *below* the median
//     fires exactly as one far above it does -- SPEC-MU §6's own
//     "Why the test is two-sided" note).
//   - otherwise -> PASS (vector MU-V87).
//
// The comparison against the threshold is restated to avoid division,
// which decimal.Decimal does not provide (see checkMU07Money's Units note
// and conversion.go's exceedsTolerance for the same restatement pattern
// elsewhere in this package): abs(modified_z) > threshold is
// mathematically equivalent to 0.6745 * abs(x - median) > threshold * MAD,
// since MAD is already known to be strictly positive at the point this
// comparison runs.
func checkMU20(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	switch decl.(type) {
	case field.MoneyDeclaration, field.DecimalDeclaration, field.PercentageDeclaration, field.QuantityDeclaration:
	default:
		return notApplicable()
	}

	if in.ValueCoercionFailed {
		return statIndeterminateResult("MU-20")
	}
	if len(in.Observations) < mu20MinObservations {
		return statIndeterminateResult("MU-20")
	}

	values := observationValues(in.Observations)
	med, err := median(sortedCopy(values))
	if err != nil {
		return statIndeterminateResult("MU-20")
	}
	mad, err := medianAbsoluteDeviation(values, med)
	if err != nil {
		return statIndeterminateResult("MU-20")
	}
	if mad.IsZero() {
		return statIndeterminateResult("MU-20")
	}

	diff, err := in.Value.Sub(med)
	if err != nil {
		return statIndeterminateResult("MU-20")
	}
	// This Mul can overflow at the *low* end of the supported exponent
	// range, not just the high end an earlier version of this comment
	// argued from: mu20ZConstant (0.6745) has adjusted exponent -1, so
	// multiplying by it shifts the product's adjusted exponent one digit
	// *lower* than diff.Abs()'s own, not higher. A diff already sitting at
	// the arithmetic package's own exponent floor -- a window whose MAD is
	// an ordinary value and whose Input.Value differs from the median by
	// something as small as 1e-100000 -- pushes the product one place past
	// that floor. SPEC-MU §2.6 does not let one check's arithmetic failure
	// discard its siblings' results, so this is INDETERMINATE for MU-20
	// alone, exactly as checkMU07Money, checkMU12, and checkMU15 already
	// treat the identical class of failure in their own arithmetic.
	lhs, err := mu20ZConstant().Mul(diff.Abs())
	if err != nil {
		return statIndeterminateResult("MU-20")
	}
	rhs, err := mu20Threshold().Mul(mad)
	if err != nil {
		return statIndeterminateResult("MU-20")
	}

	if lhs.Compare(rhs) > 0 {
		return statFailResult("MU-20")
	}
	return statPassResult("MU-20")
}
