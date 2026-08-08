package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// checkMU06 implements the sign_convention check (MU-06, SPEC-MU §3).
//
// MU-06 rejects sign inversions -- a refund recorded as a positive charge,
// or a credit as a debit.
//
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is not
//     money → INDETERMINATE.
//   - no applicable sign requirement resolves (see resolveSign) →
//     INDETERMINATE (the "Requires sign... or a conditional declaration"
//     clause: neither form applies).
//   - the resolved sign is violated by the value (see signViolated) →
//     FAIL.
//   - otherwise → PASS.
//
// # Precedence between Sign and SignWhen
//
// SPEC-MU §3 presents an unconditional Sign and a conditional SignWhen as
// alternative ways to declare this field's requirement ("Requires. sign:
// positive | negative declaration on the field, or a conditional
// declaration...") without saying what happens when a ruleset declares
// both, or when SignWhen is declared but no condition matches while an
// unconditional Sign is also present. This package treats them as
// mutually exclusive per that "or": if SignWhen is declared at all, it is
// authoritative and Sign is not consulted, even when no condition in it
// matches (that case is INDETERMINATE, not a fallback to Sign). Sign is
// only consulted when SignWhen was never declared. See resolveSign.
func checkMU06(in Input) (verdict.Result, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return indeterminateResult("MU-06")
	}

	sign, applicable := resolveSign(moneyDecl, in.Vals)
	if !applicable {
		return indeterminateResult("MU-06")
	}

	if signViolated(sign, in.Value, moneyDecl.Nonzero()) {
		return failResult("MU-06")
	}
	return passResult("MU-06")
}

// resolveSign determines which field.Sign applies to this evaluation, and
// whether any applies at all. When decl declares SignWhen conditions, they
// are checked in declaration order and the first one whose whenField is
// present in vals with the exact whenValue wins; if none match, no sign
// applies (SignWhen is authoritative once declared -- see checkMU06's doc
// comment) regardless of whether decl also declares an unconditional Sign.
// When decl declares no SignWhen conditions at all, the unconditional Sign
// is used instead.
//
// A sibling value absent from vals is absence, never an empty string
// (comma-ok), matching field.Registry.Lookup and this package's Input.Vals
// convention: a condition is never satisfied by a missing sibling
// happening to compare equal to an empty whenValue.
func resolveSign(decl field.MoneyDeclaration, vals map[string]string) (field.Sign, bool) {
	if conds := decl.SignWhen(); len(conds) > 0 {
		for _, cond := range conds {
			v, ok := vals[cond.WhenField()]
			if ok && v == cond.WhenValue() {
				return cond.Sign(), true
			}
		}
		return field.SignUnspecified, false
	}
	return decl.Sign()
}

// signViolated reports whether value violates the required sign. SPEC-MU
// §3: "Zero is permitted under both positive and negative unless nonzero:
// true is also declared" -- nonzero is applied uniformly ahead of the
// sign-specific comparison, since it is documented as an independent
// constraint on zero, not one scoped only to positive/negative.
func signViolated(sign field.Sign, value decimal.Decimal, nonzero bool) bool {
	if value.IsZero() {
		return nonzero
	}
	switch sign {
	case field.SignPositive:
		return value.Sign() < 0
	case field.SignNegative:
		return value.Sign() > 0
	default: // field.SignAny
		return false
	}
}
