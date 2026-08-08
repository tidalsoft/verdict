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
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is not
//     money → INDETERMINATE (the "Requires kind: money" clause).
//   - money declaration with no scale → INDETERMINATE (the "Requires scale
//     declaration" clause).
//   - scale: minor_units and the value carries any decimal places (including
//     trailing zeros: "49.00" → 2 places) → FAIL. Minor units are integers
//     by definition, so the representation itself contradicts the
//     declaration regardless of whether the fractional part is zero.
//   - scale: minor_units and the value carries no decimal places → PASS.
//   - scale: major_units → PASS unconditionally; the fractional bound is
//     MU-14's job, not MU-01's.
func checkMU01(in Input) (verdict.Result, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
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
	// SPEC-MU §3 MU-01: "If scale: major_units and the value's decimal
	// places exceed the currency's minor unit exponent → defer to MU-14;
	// return PASS here." MU-01 does not know the currency's exponent
	// (MU-14's job), so it passes unconditionally.
	return passResult("MU-01")
}
