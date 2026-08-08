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
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS:
//   - no declaration for the field, or a declaration whose kind is not
//     money → INDETERMINATE.
//   - money declaration with no scale declared → INDETERMINATE.
//   - scale: minor_units → INDETERMINATE. MU-14's requirement is
//     specifically "scale: major_units"; minor-units representations are
//     MU-01's territory, not MU-14's.
//   - scale: major_units but no currency_field declared → INDETERMINATE.
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
func checkMU14(in Input) (verdict.Result, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return indeterminateResult("MU-14")
	}

	scale, hasScale := moneyDecl.Scale()
	if !hasScale || scale != field.ScaleMajorUnits {
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
