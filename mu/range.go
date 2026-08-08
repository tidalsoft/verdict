package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU07 implements the range_bound check (MU-07, SPEC-MU §3):
// resolving what bounded (mu.go) needs to compare -- min and max
// normalized into the same units as the field's value -- and delegating
// the comparison itself to it. See bounded's doc comment for the
// comparison; this function's job is normalization, and handling every
// way that normalization can come up short as INDETERMINATE, never a
// guess.
//
// # Units
//
// SPEC-MU §3 does not say what unit a money field's min/max are declared
// in when its own value carries a scale (minor_units or major_units) of
// its own. This implementation reads them as always expressed in major
// units, independent of the field's declared scale -- a ruleset author
// writing `max: 100` on a USD field means $100.00, regardless of whether
// the field's own value arrives as "100.00" (major_units) or as the
// integer count of cents (minor_units). The field's value, by contrast, is
// in whatever its own scale declares. Comparing them correctly therefore
// requires normalizing one side to match the other -- never comparing an
// un-normalized major-units bound against a minor-units value, which
// would be off by the currency's minor-unit exponent (100x for USD).
//
// This function normalizes the bounds, not the value: when scale is
// minor_units, min and max are scaled up to minor units via
// ScaleByExponent using the currency's ISO 4217 minor-unit exponent before
// bounded ever sees them; when scale is major_units, they need no
// scaling at all, since the value is already in the same (major) unit
// they were declared in.
//
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS
// (invariant 1):
//   - no declaration for the field, or a declaration whose kind is not
//     money -- in practice a field.QuantityDeclaration, the only other
//     kind checksFor dispatches this check for; see moneyDeclaration's
//     doc comment for why that field.QuantityDeclaration case is a
//     genuine "nothing to evaluate," not a workaround -- → INDETERMINATE.
//   - no min and no max declared at all → INDETERMINATE (SPEC-MU §3's
//     "Requires min and/or max" unmet).
//   - no scale declared → INDETERMINATE. Without knowing which unit the
//     value is in, there is no safe way to normalize a major-units bound
//     against it -- guessing "major" by default would silently mismatch
//     the moment a customer's value is actually minor_units.
//   - scale: minor_units but no currency_field declared, no sibling value
//     for it in Input.Vals, an unresolvable code, or a currency the
//     injected ISO 4217 table reports has no minor-unit exponent (XAU,
//     XXX, ...) → INDETERMINATE. There is no exponent to scale the bounds
//     by.
//   - scale: minor_units and a declared bound overflows ScaleByExponent
//     (an extreme but legitimately parseable value) → INDETERMINATE for
//     this check alone, never an error that would abort every other
//     check's result (SPEC-MU §2.4 does not short-circuit on one check's
//     failure, and a Go error from an OnFunc aborts the whole batch --
//     see evaluateChecks -- so this function must not produce one for a
//     condition SPEC-MU classifies as "could not evaluate," not "cannot
//     evaluate at all").
//   - scale: major_units → no normalization needed; the bounds are
//     already in the same unit as the value.
//   - a resolved comparison → delegates to bounded, which returns FAIL
//     where the value falls outside a bound and PASS otherwise.
func checkMU07(in Input) (verdict.Result, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return indeterminateResult("MU-07")
	}

	min, hasMin := moneyDecl.Min()
	max, hasMax := moneyDecl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}

	scale, hasScale := moneyDecl.Scale()
	if !hasScale {
		return indeterminateResult("MU-07")
	}

	if scale == field.ScaleMinorUnits {
		currency, ok := resolveDeclaredCurrency(in, moneyDecl.CurrencyField)
		if !ok {
			return indeterminateResult("MU-07")
		}
		exponent, ok := currency.MinorUnitExponent()
		if !ok {
			return indeterminateResult("MU-07")
		}

		var err error
		if hasMin {
			min, err = min.ScaleByExponent(exponent)
			if err != nil {
				return indeterminateResult("MU-07")
			}
		}
		if hasMax {
			max, err = max.ScaleByExponent(exponent)
			if err != nil {
				return indeterminateResult("MU-07")
			}
		}
	}
	// scale == field.ScaleMajorUnits: WithScale's validation leaves no
	// third arm once hasScale == true and the minor_units case above has
	// already returned, so this is the only remaining possibility. min
	// and max (always major units, see the Units note above) already match
	// the value's own unit -- nothing left to normalize.

	return bounded(moneyDecl, in.Value, min, hasMin, max, hasMax)
}
