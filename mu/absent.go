package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU05 implements the unit_absent check (MU-05, SPEC-MU §4).
//
// MU-05 rejects a bare number where the field requires an explicit unit.
//
// Applicability (SPEC-MU §2.5.1: applies to quantity, gated on
// `unit_required: true`):
//   - no declaration for the field, or a declaration whose kind is not
//     quantity → not applicable.
//   - unit_required is not true (absent, or explicitly false) → not
//     applicable (§2.5.2: unit_required defaults to false, an actual
//     substituted value, so its absence is a coherent "a bare number is
//     acceptable" declaration, decidable without INDETERMINATE -- unlike
//     `scale`, which has no substituted default).
//
// MU-05 is not value-dependent (SPEC-MU §2.6.3's table): it never reads the
// field's own decimal value, only its unit, so §2.6.3's coercion gate never
// applies to it.
//
// Branch matrix, once applicable:
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
func checkMU05(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	qDecl, ok := decl.(field.QuantityDeclaration)
	if !ok {
		return notApplicable()
	}
	if !qDecl.UnitRequired() {
		return notApplicable()
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
