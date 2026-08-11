package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU04 implements the unit_dimension_mismatch check (MU-04, SPEC-MU
// §4).
//
// MU-04 rejects a quantity whose unit belongs to a different physical
// dimension than the field requires.
//
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is not
//     quantity → INDETERMINATE (the "Requires kind: quantity" clause).
//   - quantity declaration with no dimension declared → INDETERMINATE
//     (vector 57).
//   - the unit does not resolve at all -- neither embedded in the value
//     nor from unit_field -- → INDETERMINATE (vector 56): "a bare number
//     is not a wrong unit, it is an absent one," which is MU-05's report,
//     not this check's.
//   - the two sources resolve to different units (unit_conflict) →
//     INDETERMINATE (vectors 106, 121). See resolveQuantityUnit's doc
//     comment: neither source wins, and the conflicting string need not
//     itself be one the registry recognises.
//   - the resolved unit is not in the registry → FAIL. An unrecognised
//     unit is not safe to pass through (vector 25's "flurbs").
//   - the resolved unit is in the registry, but of a different dimension
//     than declared → FAIL (vector 23).
//   - the resolved unit is in the registry and of the declared dimension
//     → PASS (vector 22, and vector 26's affine-conversion case: MU-04
//     itself performs no conversion, only a dimension-bucket lookup, so
//     Fahrenheit resolving to DimensionTemperature is enough).
func checkMU04(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-04")
	}
	qDecl, ok := decl.(field.QuantityDeclaration)
	if !ok {
		return indeterminateResult("MU-04")
	}
	dimension, hasDimension := qDecl.Dimension()
	if !hasDimension {
		return indeterminateResult("MU-04")
	}

	resolved := resolveQuantityUnit(in, qDecl)
	if resolved.conflict || !resolved.ok {
		return indeterminateResult("MU-04")
	}

	unit, found := in.Tables.Units.Lookup(resolved.symbol)
	if !found {
		return failResult("MU-04")
	}
	if unit.Dimension().String() != dimension {
		return failResult("MU-04")
	}
	return passResult("MU-04")
}
