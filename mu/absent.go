package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU05 implements the unit_absent check (MU-05, SPEC-MU §4).
//
// MU-05 rejects a bare number where the field requires an explicit unit.
//
// This check's *Applies to* condition -- `unit_required: true` -- is a
// gate (SPEC-MU §2.5.2: "a bare number is acceptable" is a coherent
// declaration in its own right when the attribute is absent, since
// unit_required defaults to false). This package's dispatch does not yet
// distinguish "not applicable" from INDETERMINATE at the Result level
// (see mu.go's package doc comment); consistent with every other gated
// check already shipped in this package (e.g. checkMU03's
// target_currency_field gate), an undeclared unit_required: true reports
// INDETERMINATE here rather than omitting a result.
//
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is not
//     quantity → INDETERMINATE.
//   - unit_required is not true → INDETERMINATE (this check's gate; see
//     the note above).
//   - the two sources resolve to different units (unit_conflict) →
//     INDETERMINATE (vector 58): "the unit is not absent; it is
//     contradicted."
//   - no unit resolves at all → FAIL (vector 24).
//   - a unit resolves (from either or both sources, agreeing) → PASS,
//     regardless of whether that unit is one the registry recognises
//     (vector 59: a quantity value decomposed for this check's benefit
//     alone, with nothing reading its number, still passes on its
//     embedded unit) and regardless of whether the value's number itself
//     was withheld as an ambiguous decomposition (SPEC-MU §2.6.1: the
//     unit is still reported even when the number is not).
func checkMU05(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-05")
	}
	qDecl, ok := decl.(field.QuantityDeclaration)
	if !ok {
		return indeterminateResult("MU-05")
	}
	if !qDecl.UnitRequired() {
		return indeterminateResult("MU-05")
	}

	resolved := resolveQuantityUnit(in, qDecl)
	if resolved.conflict {
		return indeterminateResult("MU-05")
	}
	if !resolved.ok {
		return failResult("MU-05")
	}
	return passResult("MU-05")
}
