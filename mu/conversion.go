package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// defaultQuantityTolerance is MU-15's relative round-trip tolerance when
// `tolerance` is not declared (SPEC-MU §2.4.2): 1 part in 10^9.
// "0.000000001" is decimal text decimal.Parse always accepts -- the panic
// branch mustParseDecimal wraps this in is exercised directly by
// TestMustParseDecimal_PanicsOnInvalidInput (scale.go's test file), not
// reachable from this call site; see mustParseDecimal's own doc comment.
func defaultQuantityTolerance() decimal.Decimal {
	return mustParseDecimal("0.000000001")
}

// checkMU15 implements the unit_conversion_overflow check (MU-15, SPEC-MU
// §4).
//
// MU-15 detects a unit conversion that does not round-trip: the value
// converted to the canonical unit and back is not the value that arrived,
// beyond the declared (or default) relative tolerance. It is the one
// check in this package whose default severity is warn rather than
// block -- every Result it produces, including PASS and INDETERMINATE,
// carries SeverityWarn (warnResult), not just its FAIL branch (contrast
// MU-13, whose block default is overridden on one branch only).
//
// Applicability (SPEC-MU §2.5.1: applies to quantity, gated on
// `canonical_unit` being declared -- §2.5.2's own example of an attribute
// that gates this check while being merely a required input on MU-07):
//   - no declaration for the field, or a declaration whose kind is not
//     quantity → not applicable. Reported the same way every other not
//     applicable check is: no entry at all, regardless of MU-15's own warn
//     default severity -- that severity governs the Result this function
//     builds once it is applicable, never whether one is built at all.
//   - canonical_unit is not declared → not applicable (the gate itself).
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS:
//   - the value is not coercible (§2.6.3; MU-15 is value-dependent) →
//     INDETERMINATE, reason value_not_coercible. Checked immediately after
//     the applicability gate and before any required-input logic below --
//     the same position every other value-dependent check in this package
//     uses -- so that a value the coercion gate could not read is reported
//     as such rather than as whatever required-input gap the branch below
//     it happens to be.
//   - canonical_unit is declared but the registry does not recognise it →
//     INDETERMINATE: there is no canonical unit to convert to or from.
//   - the value's unit does not resolve at all → INDETERMINATE (vector
//     85): "nothing to convert from."
//   - the two unit sources conflict (unit_conflict) → INDETERMINATE
//     (vector 107).
//   - the value's unit resolves but the registry does not recognise it →
//     INDETERMINATE: no conversion factor (MU-04 separately reports the
//     same input as FAIL; this check's own outcome is independent, per
//     SPEC-MU §2.1).
//   - the value's unit resolves, is recognised, but is of a different
//     dimension than canonical_unit's own → INDETERMINATE (vector 120):
//     nothing to bring to a common scale.
//   - the round trip's absolute difference from the original value
//     exceeds tolerance * abs(original) → FAIL (vector 97: at
//     `tolerance: "0"`, Fahrenheit's forward conversion uses a truncated
//     decimal approximation of 5/9 and therefore never round-trips
//     exactly).
//   - otherwise → PASS (vector 83).
func checkMU15(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	qDecl, ok := decl.(field.QuantityDeclaration)
	if !ok {
		return notApplicable()
	}

	canonicalSymbol, hasCanonical := qDecl.CanonicalUnit()
	if !hasCanonical {
		return notApplicable()
	}
	if in.ValueCoercionFailed {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}
	canonicalUnit, found := in.Tables.Units.Lookup(canonicalSymbol)
	if !found {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}

	resolved := resolveQuantityUnit(in, qDecl)
	if resolved.conflict || !resolved.ok {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}

	valueUnit, found := in.Tables.Units.Lookup(resolved.symbol)
	if !found {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}
	if valueUnit.Dimension() != canonicalUnit.Dimension() {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}

	roundTripped, err := roundTrip(valueUnit, canonicalUnit, in.Value)
	if err != nil {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}

	tolerance := defaultQuantityTolerance()
	if t, has := qDecl.Tolerance(); has {
		tolerance = t
	}

	exceeds, err := exceedsTolerance(in.Value, roundTripped, tolerance)
	if err != nil {
		return warnResult("MU-15", verdict.OutcomeIndeterminate)
	}
	if exceeds {
		return warnResult("MU-15", verdict.OutcomeFail)
	}
	return warnResult("MU-15", verdict.OutcomePass)
}

// roundTrip converts value, expressed in valueUnit, to the *declared*
// canonicalUnit and back to valueUnit -- SPEC-MU §4 MU-15's own
// definition, "the value converted to the canonical unit and back is not
// the value that arrived" -- entirely via convertBetweenUnits (unit.go),
// whose own doc comment explains why this must go by way of the declared
// canonicalUnit rather than always the registry's fixed per-dimension
// canonical, and why a value already expressed in canonicalUnit
// round-trips exactly regardless of that unit's own stored factors. Its
// one failure mode is either leg's own (convertBetweenUnits' contract: an
// exponent-range overflow), collapsed into a single error here rather
// than reported per-leg, since checkMU15 treats both identically --
// INDETERMINATE, never an aborted evaluation (SPEC-MU §2.6 does not let
// one check's arithmetic failure discard its siblings' results).
func roundTrip(valueUnit, canonicalUnit tables.Unit, value decimal.Decimal) (decimal.Decimal, error) {
	inCanonical, err := convertBetweenUnits(valueUnit, canonicalUnit, value)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return convertBetweenUnits(canonicalUnit, valueUnit, inCanonical)
}

// exceedsTolerance reports whether roundTripped's absolute difference
// from original exceeds tolerance * abs(original) -- SPEC-MU §4 MU-15's
// comparison, restated to avoid division (decimal, §2.6.1's value model,
// has no division operator): the mathematically equivalent
// abs(original - roundTripped) > tolerance * abs(original), computed
// entirely in exact decimal arithmetic. Its error return covers both of
// its two arithmetic steps (Sub and Mul), for the same
// collapse-per-step-errors-into-one reason roundTrip does.
func exceedsTolerance(original, roundTripped, tolerance decimal.Decimal) (bool, error) {
	diff, err := original.Sub(roundTripped)
	if err != nil {
		return false, err
	}
	bound, err := tolerance.Mul(original.Abs())
	if err != nil {
		return false, err
	}
	return diff.Abs().Compare(bound) > 0, nil
}
