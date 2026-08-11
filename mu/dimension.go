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
// Applicability (SPEC-MU §2.5.1: applies to quantity, no further gate):
//   - no declaration for the field, or a declaration whose kind is not
//     quantity → not applicable.
//
// MU-04 is not value-dependent (SPEC-MU §2.6.3's table): it never reads the
// field's own decimal value, only its unit, so §2.6.3's coercion gate never
// applies to it.
//
// dimension itself is drawn from tables.Dimension's closed enumeration --
// field.QuantityDeclaration.WithDimension rejects anything else at
// construction (SPEC-MU §4's "Supported dimensions" list), so an
// unrecognised dimension string never reaches this function at all; see
// its own doc comment for why that rejection belongs at construction
// rather than as a manufactured FAIL here.
//
// Branch matrix, once applicable:
//   - quantity declaration with no dimension declared → INDETERMINATE
//     (§2.5.2: a quantity has a dimension whether or not the ruleset says
//     which -- a required input, not a gate; vector 57).
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
func checkMU04(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	qDecl, ok := decl.(field.QuantityDeclaration)
	if !ok {
		return notApplicable()
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
