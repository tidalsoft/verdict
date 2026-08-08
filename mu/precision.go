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
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is neither
//     money nor decimal → INDETERMINATE (the "Requires kind: money or
//     kind: decimal" clause).
//   - value arrived as a decimal string (Provenance: FromString) → PASS
//     unconditionally. A decimal string is exact by construction.
//   - value arrived as a JSON number (Provenance: FromJSONNumber) and its
//     exact value is not exactly representable in IEEE 754 binary64, or
//     its magnitude exceeds 2^53-1 → FAIL.
//   - value arrived as a JSON number and neither condition holds → PASS.
//
// decimal.PrecisionLoss (decimal/precision.go) implements the underlying
// binary64-fidelity and safe-integer-magnitude tests; this function's only
// job is gating that boolean on the field's declared kind, per this
// package's INDETERMINATE-on-absent-declaration convention (SPEC-MU §2.3).
func checkMU02(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-02")
	}
	switch decl.(type) {
	case field.MoneyDeclaration, field.DecimalDeclaration:
		// MU-02 applies to both kinds identically -- fall through.
	default:
		return indeterminateResult("MU-02")
	}

	if decimal.PrecisionLoss(in.Value, in.Provenance) {
		return failResult("MU-02")
	}
	return passResult("MU-02")
}
