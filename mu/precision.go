package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// checkMU02 implements the precision_loss check (MU-02, SPEC-MU §3).
//
// MU-02 rejects a value supplied in a representation that cannot hold it
// exactly.
//
// Applicability (SPEC-MU §2.5.1: applies to money and decimal, no further
// gate):
//   - no declaration for the field, or a declaration whose kind is neither
//     money nor decimal → not applicable.
//
// Branch matrix, once applicable:
//   - the value is not coercible (§2.6.3; MU-02 is value-dependent) →
//     INDETERMINATE, reason value_not_coercible (vectors 44, 45). This
//     precedes every branch below: a string resolution refused, or a value
//     that arrived as neither a string nor a number, never reaches the
//     "arrived as a string → PASS" branch, which is safe only because this
//     interception runs first (SPEC-MU §3 MU-02's own note).
//   - value arrived as a decimal string (Provenance: FromString) → PASS
//     unconditionally. A decimal string is exact by construction.
//   - value arrived as a JSON number (Provenance: FromJSONNumber) and its
//     exact value is not exactly representable in IEEE 754 binary64, or
//     its magnitude exceeds 2^53-1 → FAIL.
//   - value arrived as a JSON number and neither condition holds → PASS.
//
// decimal.PrecisionLoss (decimal/precision.go) implements the underlying
// binary64-fidelity and safe-integer-magnitude tests; this function's only
// job is gating that boolean on the field's declared kind and on
// coercion, per this package's applicability and coercion conventions.
func checkMU02(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	switch decl.(type) {
	case field.MoneyDeclaration, field.DecimalDeclaration:
		// MU-02 applies to both kinds identically -- fall through.
	default:
		return notApplicable()
	}
	if in.ValueCoercionFailed {
		return indeterminateResult("MU-02")
	}

	if decimal.PrecisionLoss(in.Value, in.Provenance) {
		return failResult("MU-02")
	}
	return passResult("MU-02")
}
