package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/tables"
)

// checkMU03 implements the currency_mismatch check (MU-03, SPEC-MU §3).
//
// MU-03 rejects a transaction whose currency differs from the target's
// currency.
//
// Branch matrix -- every unmet requirement is INDETERMINATE, never PASS:
//   - no declaration for the field, or a declaration whose kind is not
//     money → INDETERMINATE (only a money field carries a currency_field
//     at all).
//   - no currency_field declared → INDETERMINATE (the "Requires
//     currency_field declaration" clause).
//   - no target_currency_field declared → INDETERMINATE. See
//     field.MoneyDeclaration.TargetCurrencyField's doc comment for why
//     this attribute exists and what interpretive choice it represents.
//   - either side's sibling value is absent from Input.Vals, or does not
//     resolve in the injected ISO 4217 table → INDETERMINATE ("Either
//     unresolvable → INDETERMINATE").
//   - both resolve and are unequal → FAIL.
//   - both resolve and are equal → PASS.
//
// # Open question this check turns on
//
// SPEC-MU §3 requires "a resolvable target currency, supplied either in
// state or as a second declared field," but neither this package's Input
// nor field.MoneyDeclaration has any notion of "state" distinct from a
// sibling field's value in Vals. This function resolves the target
// currency strictly as a second declared field
// (MoneyDeclaration.TargetCurrencyField), the same shape and mechanism as
// currency_field itself -- see this task's report for the alternative
// readings and the blast radius of this choice.
func checkMU03(in Input) (verdict.Result, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return indeterminateResult("MU-03")
	}

	source, ok := resolveDeclaredCurrency(in, moneyDecl.CurrencyField)
	if !ok {
		return indeterminateResult("MU-03")
	}
	target, ok := resolveDeclaredCurrency(in, moneyDecl.TargetCurrencyField)
	if !ok {
		return indeterminateResult("MU-03")
	}

	if source.Code() != target.Code() {
		return failResult("MU-03")
	}
	return passResult("MU-03")
}

// resolveDeclaredCurrency resolves the currency named by a currency-field
// accessor -- MoneyDeclaration.CurrencyField or
// MoneyDeclaration.TargetCurrencyField, both `func() (string, bool)` --
// through the declared sibling path, looked up in in.Vals, looked up in
// the injected ISO 4217 table. Any missing step collapses to a single
// false. This is the currency resolution every check in this package that
// needs one performs: MU-03 (checkMU03, above) calls it twice, once per
// side of its comparison; MU-07 (checkMU07, range.go) and MU-14
// (checkMU14, exponent.go) each call it once for the single currency they
// need.
func resolveDeclaredCurrency(in Input, accessor func() (string, bool)) (tables.Currency, bool) {
	path, has := accessor()
	if !has {
		return tables.Currency{}, false
	}
	code, ok := in.Vals[path]
	if !ok {
		return tables.Currency{}, false
	}
	return in.Tables.resolveCurrency(code)
}
