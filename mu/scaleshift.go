package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
)

// mu21MinObservations is SPEC-MU §6 MU-21's own floor, identical in value
// to MU-20's (mu20MinObservations, outlier.go) but declared separately:
// the two checks' requirements happen to share a number, not a
// declaration, and a future change to one must not silently move the
// other.
const mu21MinObservations = 200

// mu21TenPercent is SPEC-MU §6 MU-21's tolerance: "within 10% of exactly
// 100x or exactly 1/100 of the field's rolling median".
func mu21TenPercent() decimal.Decimal { return mustParseDecimal("0.1") }

// mu21Hundred is the 100x scale-shift factor SPEC-MU §6 MU-21 tests
// against.
func mu21Hundred() decimal.Decimal { return mustParseDecimal("100") }

// checkMU21 implements the scale_shift_suspect check (MU-21, SPEC-MU §6):
// the heuristic cents-versus-dollars fallback for a money field that
// declares no `scale` at all, so MU-01 (the deterministic form) has
// nothing to check.
//
// Applicability (SPEC-MU §2.5.1: applies to `money`, applicable only when
// `scale` is not declared):
//   - no declaration for the field, or a declaration of a kind other than
//     money -> not applicable.
//   - `scale` is declared (either value) -> not applicable: "a field that
//     declares its scale is MU-01's, and this check does not run on it."
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS or FAIL:
//   - the value is not coercible (§2.6.3; MU-21 is value-dependent) ->
//     INDETERMINATE, reason value_not_coercible.
//   - fewer than mu21MinObservations recorded observations ->
//     INDETERMINATE (vector MU-V111).
//   - the rolling median is zero, the *Degenerate case* SPEC-MU §6 names
//     explicitly: "100x it and 1/100 of it are both zero, and the test
//     distinguishes nothing at all" -> INDETERMINATE, never PASS (vector
//     MU-V112).
//   - the value is within 10% of exactly 100x or exactly 1/100 of the
//     median, and outside the interquartile range -> FAIL at warn
//     (vector MU-V40).
//   - otherwise -> PASS.
//   - hundredX, oneHundredth, the tolerance comparisons, or quartiles
//     overflow the exact decimal range on some legitimately parseable but
//     extreme median or value -> INDETERMINATE for MU-21 alone, never an
//     aborted evaluation (SPEC-MU §2.6 does not let one check's arithmetic
//     failure discard its siblings' results).
//
// "Within 10%" and the interquartile bound are both computed via
// multiplication rather than division, mirroring checkMU20's own
// restatement (outlier.go) for the same reason: decimal.Decimal provides
// no division.
func checkMU21(in Input) (verdict.Result, bool, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return notApplicable()
	}
	if _, hasScale := moneyDecl.Scale(); hasScale {
		return notApplicable()
	}

	if in.ValueCoercionFailed {
		return statIndeterminateResult("MU-21")
	}
	if len(in.Observations) < mu21MinObservations {
		return statIndeterminateResult("MU-21")
	}

	values := observationValues(in.Observations)
	sorted := sortedCopy(values)
	med, err := median(sorted)
	if err != nil {
		return statIndeterminateResult("MU-21")
	}
	if med.IsZero() {
		return statIndeterminateResult("MU-21")
	}

	hundredX, err := med.Mul(mu21Hundred())
	if err != nil {
		return statIndeterminateResult("MU-21")
	}
	// A power-of-ten division is exact and needs no general division
	// operator: ScaleByExponent(-2) shifts the decimal point, which is
	// what dividing by 100 means for any exact decimal.
	oneHundredth, err := med.ScaleByExponent(-2)
	if err != nil {
		return statIndeterminateResult("MU-21")
	}

	nearHundredX, err := withinRelativeTolerance(in.Value, hundredX, mu21TenPercent())
	if err != nil {
		return statIndeterminateResult("MU-21")
	}
	// This second comparison is checked separately from the first, not
	// assumed infallible because the first one succeeded: Sub's cost and
	// failure mode turn on the exponent *gap* between its two operands,
	// not on which of hundredX/oneHundredth sits closer to med in
	// magnitude. hundredX sits two decimal places above med's own exponent
	// and oneHundredth two below it, so against an in.Value whose exponent
	// is far above both, oneHundredth is the operand with the *larger*
	// gap, not the smaller one -- the reverse of what an argument from
	// algebraic magnitude alone would suggest. The first comparison
	// succeeding says nothing about whether this one will.
	nearOneHundredth, err := withinRelativeTolerance(in.Value, oneHundredth, mu21TenPercent())
	if err != nil {
		return statIndeterminateResult("MU-21")
	}
	if !nearHundredX && !nearOneHundredth {
		return statPassResult("MU-21")
	}

	q1, q3, err := quartiles(sorted)
	if err != nil {
		return statIndeterminateResult("MU-21")
	}
	if in.Value.Compare(q1) >= 0 && in.Value.Compare(q3) <= 0 {
		// Inside the interquartile range: MU-21's own text requires
		// "outside the interquartile range" as an additional, separate
		// condition to FAIL, so a value inside it PASSes regardless of
		// how close it fell to a 100x/1-in-100 shift.
		return statPassResult("MU-21")
	}
	return statFailResult("MU-21")
}

// withinRelativeTolerance reports whether value falls within the given
// relative tolerance of target -- SPEC-MU §6 MU-21's "within 10%",
// restated as abs(value - target) <= tolerance * abs(target) to avoid
// division (see checkMU21's own doc comment). Boundary equality counts as
// "within," matching this package's inclusive-by-default reading of every
// other bound comparison (bounded, mu.go).
func withinRelativeTolerance(value, target, tolerance decimal.Decimal) (bool, error) {
	diff, err := value.Sub(target)
	if err != nil {
		return false, err
	}
	// This Mul is checked, not assumed infallible: tolerance is always
	// mu21TenPercent() (0.1), whose adjusted exponent is -1, so
	// multiplying by it shifts the product's adjusted exponent one digit
	// *lower* than target.Abs()'s own -- the same low-end overflow
	// checkMU20's lhs computation (outlier.go) can now hit, for the same
	// reason.
	bound, err := tolerance.Mul(target.Abs())
	if err != nil {
		return false, err
	}
	return diff.Abs().Compare(bound) <= 0, nil
}
