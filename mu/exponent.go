package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU14 implements the minor_unit_exponent check (MU-14, SPEC-MU §3).
//
// MU-14 rejects amounts carrying more decimal places than the currency
// permits.
//
// Applicability (SPEC-MU §2.5.1: applies to money, gated on
// `scale: major_units` -- the one row in the trigger matrix whose gate can
// itself be undecidable, per §2.5.1 step 3 and §2.5.2's closing section):
//   - no declaration for the field, or a declaration whose kind is not
//     money → not applicable.
//   - money declaration with no scale declared → the gate itself cannot be
//     read true or false from the declaration, so this is the one
//     exception to "gate false → not applicable": applicable, and
//     INDETERMINATE (vector 101). MU-01 is not affected by this -- its own
//     applicability has no such gate.
//   - scale: minor_units → not applicable. The gate reads false outright:
//     MU-14's requirement is specifically "scale: major_units";
//     minor-units representations are MU-01's territory, not MU-14's, and
//     MU-01 owns the case entirely.
//
// Branch matrix, once applicable (scale: major_units) -- every unmet
// requirement is INDETERMINATE, never PASS:
//   - the value is not coercible (§2.6.3; MU-14 is value-dependent) →
//     INDETERMINATE, reason value_not_coercible.
//   - no currency_field declared → INDETERMINATE.
//   - currency_field declared but no sibling value at that path in
//     Input.Vals → INDETERMINATE.
//   - sibling value present but absent from the injected ISO 4217 table →
//     INDETERMINATE.
//   - currency resolves but MinorUnitExponent() reports it has none (XAU,
//     XXX, and the other ISO 4217 entries with no minor unit) →
//     INDETERMINATE. The exponent is never defaulted to 2 or any other
//     value here: guessing would turn a check that verified nothing into a
//     PASS, which SPEC-MU §2.1 and this project's invariants forbid.
//   - value's DecimalPlaces() exceeds the resolved exponent → FAIL.
//   - value's DecimalPlaces() is within the resolved exponent → PASS.
//
// Trailing zeros are significant, not noise: "49.90" under USD (exponent 2)
// is 2 places → PASS; "49.900" is 3 places → FAIL. DecimalPlaces() reports
// exactly what was supplied, per its own doc comment.
func checkMU14(in Input) (verdict.Result, bool, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return notApplicable()
	}

	scale, hasScale := moneyDecl.Scale()
	if !hasScale {
		// §2.5.1 step 3: the gate itself is undecidable without a
		// declared scale, so this is applicable and INDETERMINATE rather
		// than not applicable (vector 101) -- see this function's own
		// doc comment.
		return indeterminateResult("MU-14")
	}
	if scale != field.ScaleMajorUnits {
		return notApplicable()
	}
	if in.ValueCoercionFailed {
		return indeterminateResult("MU-14")
	}

	currency, ok := resolveDeclaredCurrency(in, moneyDecl.CurrencyField)
	if !ok {
		return indeterminateResult("MU-14")
	}

	exponent, ok := currency.MinorUnitExponent()
	if !ok {
		// ISO 4217 entries like XAU/XXX declare no minor unit at all
		// ("N.A." in the published list) -- not exponent 0, which is
		// itself a legitimate, common exponent (JPY, KRW, ...). There
		// is nothing to compare DecimalPlaces() against.
		return indeterminateResult("MU-14")
	}

	if in.Value.DecimalPlaces() > exponent {
		return failResult("MU-14")
	}
	return passResult("MU-14")
}
