package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU07 implements the range_bound check (MU-07, SPEC-MU §3): reject
// a value outside its declared bounds. SPEC-MU §2.5.1's trigger matrix
// applies it to four kinds -- money, decimal, percentage, and quantity --
// each of which brings its bounds and its value into the same units a
// different way, so this function is a dispatch over the field's declared
// kind to one function per kind; bounded (mu.go) is the one comparison
// every branch ends in once its own normalization is done.
//
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS
// (invariant 1):
//   - no declaration for the field at all → INDETERMINATE.
//   - a declaration of a kind this check does not apply to (timestamp,
//     identifier) → INDETERMINATE.
//   - neither min nor max declared, on any kind → INDETERMINATE (the
//     "Requires min and/or max" gate all four branches share).
//   - each kind's own further requirements are documented on its own
//     function below.
func checkMU07(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-07")
	}

	switch d := decl.(type) {
	case field.MoneyDeclaration:
		return checkMU07Money(in, d)
	case field.DecimalDeclaration:
		return checkMU07Decimal(in, d)
	case field.PercentageDeclaration:
		return checkMU07Percentage(in, d)
	case field.QuantityDeclaration:
		return checkMU07Quantity(in, d)
	default:
		return indeterminateResult("MU-07")
	}
}

// checkMU07Money is MU-07's money branch.
//
// # Units
//
// SPEC-MU §3: "min and max on a money field are always expressed in major
// units, whatever the field's scale declares... The value and the bounds
// are brought to a common scale before they are compared. Where scale:
// minor_units, each declared bound is multiplied by 10 raised to the
// resolved currency's minor unit exponent, and the comparison is
// performed in minor units." This function normalizes the bounds, not the
// value: when scale is minor_units, min and max are scaled up to minor
// units via ScaleByExponent using the currency's ISO 4217 minor-unit
// exponent before bounded ever sees them; when scale is major_units, they
// need no scaling at all, since the value is already in the same (major)
// unit they were declared in. Vector 60 is the case that separates this
// reading from the no-op alternative of scaling both sides identically
// (see SPEC-MU §3's own "Why major units, and not the field's own scale"
// note); vector 62 catches an implementation that hardcodes exponent 2;
// vector 63 pins that a major_units field reads no currency at all.
//
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS:
//   - no min and no max declared at all → INDETERMINATE.
//   - no scale declared → INDETERMINATE. Without knowing which unit the
//     value is in, there is no safe way to normalize a major-units bound
//     against it.
//   - scale: minor_units but no currency_field declared, no sibling value
//     for it in Input.Vals, an unresolvable code, or a currency the
//     injected ISO 4217 table reports has no minor-unit exponent (XAU,
//     XXX, ...) → INDETERMINATE. There is no exponent to scale the bounds
//     by (vector 64).
//   - scale: minor_units and a declared bound overflows ScaleByExponent
//     (an extreme but legitimately parseable value) → INDETERMINATE for
//     this check alone, never an error that would abort every other
//     check's result (SPEC-MU §2.6 does not short-circuit on one check's
//     failure, and a Go error from an OnFunc aborts the whole batch --
//     see evaluateChecks -- so this function must not produce one for a
//     condition SPEC-MU classifies as "could not evaluate," not "cannot
//     evaluate at all").
//   - scale: major_units → no normalization needed; the bounds are
//     already in the same unit as the value (vector 63: no currency
//     read).
//   - a resolved comparison → delegates to bounded, which returns FAIL
//     where the value falls outside a bound and PASS otherwise.
func checkMU07Money(in Input, decl field.MoneyDeclaration) (verdict.Result, error) {
	min, hasMin := decl.Min()
	max, hasMax := decl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}

	scale, hasScale := decl.Scale()
	if !hasScale {
		return indeterminateResult("MU-07")
	}

	if scale == field.ScaleMinorUnits {
		currency, ok := resolveDeclaredCurrency(in, decl.CurrencyField)
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

	return bounded(decl, in.Value, min, hasMin, max, hasMax)
}

// checkMU07Decimal is MU-07's decimal branch. A decimal field's bounds
// and its value share the field's own units (SPEC-MU §2.4.2), so no
// normalization input beyond min/max themselves is required -- unlike
// money, percentage, or quantity, all of which need at least one further
// declared attribute before their bounds and value are comparable at all.
//
// Branch matrix: no min and no max declared → INDETERMINATE; otherwise
// delegates directly to bounded, against the field's own value.
func checkMU07Decimal(in Input, decl field.DecimalDeclaration) (verdict.Result, error) {
	min, hasMin := decl.Min()
	max, hasMax := decl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}
	return bounded(decl, in.Value, min, hasMin, max, hasMax)
}

// checkMU07Percentage is MU-07's percentage branch. Its bounds are stated
// "in the units of the declared domain" (SPEC-MU §2.4.2), so this branch
// requires Domain to be declared before it can interpret its own Min/Max
// at all -- the bound comparison itself needs no further conversion once
// that is settled, since the value already carries the same units the
// bound was written in.
//
// Branch matrix: no min and no max declared → INDETERMINATE; domain not
// declared → INDETERMINATE (vector 66); otherwise delegates to bounded,
// against the field's own value.
func checkMU07Percentage(in Input, decl field.PercentageDeclaration) (verdict.Result, error) {
	min, hasMin := decl.Min()
	max, hasMax := decl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}
	if _, hasDomain := decl.Domain(); !hasDomain {
		return indeterminateResult("MU-07")
	}
	return bounded(decl, in.Value, min, hasMin, max, hasMax)
}

// checkMU07Quantity is MU-07's quantity branch. Its bounds are declared
// "in the canonical unit," so this branch needs both a resolvable
// canonical_unit for the bounds and a resolvable, same-dimension unit for
// the value before the two can be compared at all -- resolveQuantityUnit
// (unit.go) is the shared resolution this branch shares with MU-04,
// MU-05, and MU-15; see its doc comment for the consumer trace.
//
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS:
//   - no min and no max declared → INDETERMINATE.
//   - canonical_unit is not declared → INDETERMINATE (vector 98: "bounds
//     have no stated units").
//   - canonical_unit is declared but the registry does not recognise it →
//     INDETERMINATE: same consequence as it not being declared at all.
//   - no unit resolves for the value → INDETERMINATE (vector 67).
//   - the two unit sources conflict (unit_conflict) → INDETERMINATE
//     (vector 105).
//   - the value's unit resolves but the registry does not recognise it →
//     INDETERMINATE: no conversion factor.
//   - the value's unit resolves, is recognised, but is of a different
//     dimension than canonical_unit's own → INDETERMINATE (vector 119) --
//     independent of MU-04's own FAIL on the identical input (SPEC-MU
//     §2.1: each check's outcome is its own).
//   - conversion overflows (ToCanonical's one failure mode) →
//     INDETERMINATE, never an aborting error (see checkMU07Money's own
//     note on this for the money branch's equivalent case).
//   - a resolved comparison, on the canonical-unit-converted value →
//     delegates to bounded (vector 84).
func checkMU07Quantity(in Input, decl field.QuantityDeclaration) (verdict.Result, error) {
	min, hasMin := decl.Min()
	max, hasMax := decl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}

	canonicalSymbol, hasCanonical := decl.CanonicalUnit()
	if !hasCanonical {
		return indeterminateResult("MU-07")
	}
	canonicalUnit, found := in.Tables.Units.Lookup(canonicalSymbol)
	if !found {
		return indeterminateResult("MU-07")
	}

	resolved := resolveQuantityUnit(in, decl)
	if resolved.conflict || !resolved.ok {
		return indeterminateResult("MU-07")
	}

	valueUnit, found := in.Tables.Units.Lookup(resolved.symbol)
	if !found {
		return indeterminateResult("MU-07")
	}
	if valueUnit.Dimension() != canonicalUnit.Dimension() {
		return indeterminateResult("MU-07")
	}

	converted, err := valueUnit.ToCanonical(in.Value)
	if err != nil {
		return indeterminateResult("MU-07")
	}

	return bounded(decl, converted, min, hasMin, max, hasMax)
}
