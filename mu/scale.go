package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU01 implements the scale_declaration_conflict check (MU-01, SPEC-MU
// §3).
//
// MU-01 detects a value whose representation contradicts its declared scale.
// This is the deterministic form of cents-versus-dollars detection.
//
// Applicability (SPEC-MU §2.5.1: applies to money, no further gate):
//   - no declaration for the field, or a declaration whose kind is not
//     money → not applicable.
//
// Branch matrix, once applicable:
//   - the value is not coercible (§2.6.3; MU-01 is value-dependent) →
//     INDETERMINATE, reason value_not_coercible (vector 43).
//   - money declaration with no scale → INDETERMINATE (the "Requires scale
//     declaration" clause, §2.5.2: scale is a required input, not a gate --
//     a money field has a scale whether or not the ruleset says which).
//   - scale: minor_units and the value carries any decimal places (including
//     trailing zeros: "49.00" → 2 places) → FAIL. Minor units are integers
//     by definition, so the representation itself contradicts the
//     declaration regardless of whether the fractional part is zero.
//   - scale: minor_units and the value carries no decimal places → PASS.
//   - scale: major_units → PASS unconditionally; the fractional bound is
//     MU-14's job, not MU-01's.
func checkMU01(in Input) (verdict.Result, bool, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return notApplicable()
	}
	if in.ValueCoercionFailed {
		return indeterminateResult("MU-01")
	}

	scale, hasScale := moneyDecl.Scale()
	if !hasScale {
		return indeterminateResult("MU-01")
	}

	if scale == field.ScaleMinorUnits {
		// Minor units are integers by definition. Any decimal place at
		// all -- including a trailing-zero fractional part like "49.00"
		// -- means the value's own representation contradicts the
		// declaration, independent of its numeric magnitude. That is
		// why this tests DecimalPlaces() rather than comparing the
		// value against its own truncation.
		if in.Value.DecimalPlaces() > 0 {
			return failResult("MU-01")
		}
		return passResult("MU-01")
	}

	// scale is field.ScaleMajorUnits: WithScale rejects any Scale for
	// which valid() is false, so hasScale == true leaves exactly these
	// two values reachable here, and the minor_units case already
	// returned above. There is no third arm -- and so no `default` --
	// because Scale is a closed, validated two-value type, not an open
	// enum a future declaration could widen without also changing
	// WithScale's validation.
	//
	// SPEC-MU §3 MU-01: "If scale: major_units → PASS, whatever the
	// value's decimal places. How many a currency permits is MU-14's
	// subject and not this check's." This function reads no currency and
	// needs none here -- the Implementation note directly under that
	// sentence warns specifically against phrasing this arm in terms of
	// the currency's minor-unit exponent, which is what an earlier,
	// inaccurate form of this comment did; see MU-01's own Implementation
	// note for why that reading is a mistake worth naming explicitly.
	return passResult("MU-01")
}
