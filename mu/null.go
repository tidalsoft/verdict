package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU08 implements the null_vs_absent check (MU-08, SPEC-MU §5).
//
// MU-08 distinguishes an explicit `null` from an omitted field, where the
// target system assigns different meanings to the two (typically: omitted
// means "leave unchanged", null means "clear the value").
//
// Applicability (SPEC-MU §2.5.1: applies to every kind, gated on
// `null_semantics: distinct` being declared):
//   - no declaration for the field → not applicable.
//   - null_semantics is not declared → not applicable (§2.5.2:
//     `null_semantics` absent is a gate, not a gap -- "null and omission
//     mean the same thing to the target system" is a coherent, complete
//     declaration by its own absence).
//
// MU-08 is not value-dependent (SPEC-MU §2.6.3's table): it reads only
// whether the field's path is present in the request and, where it is,
// whether its value is an explicit JSON null -- both properties of
// Input.RawValue/Input.HasRawValue directly, never of the coerced
// Value/Provenance/ValueCoercionFailed a value-dependent check would read.
// SPEC-MU §5 states this check's own *Indeterminate when* is "Never": both
// properties it reads always resolve, and neither is the field's value read
// as a number, so the coercion gate never reaches it.
//
// Branch matrix, once applicable:
//   - Input.HasRawValue is true and RawValue.IsNull() is true (an explicit
//     JSON null at the field's path) → FAIL (vector 69).
//   - Input.HasRawValue is false (the field's path is absent from the
//     arguments) → PASS (vector 70). Absence is one of the two meanings
//     `null_semantics: distinct` distinguishes, and it is the one this
//     check does not object to.
//   - Input.HasRawValue is true and RawValue is present but not null (any
//     other JSON value) → PASS. This check reads only whether the value is
//     null; what it is otherwise belongs to the checks the field's kind
//     triggers.
func checkMU08(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	nullSemantics, has := decl.NullSemantics()
	if !has || nullSemantics != field.NullSemanticsDistinct {
		// `has` alone already gates this correctly, since
		// field.common.withNullSemantics rejects every value except
		// field.NullSemanticsDistinct -- but this checks the resolved
		// value explicitly too, rather than trusting `has` alone to
		// imply it, matching how this package's other closed two-plus-
		// value enum gates (e.g. checkMU07Money's Scale branch) read.
		return notApplicable()
	}

	if !in.HasRawValue {
		return passResult("MU-08")
	}
	if in.RawValue.IsNull() {
		return failResult("MU-08")
	}
	return passResult("MU-08")
}
