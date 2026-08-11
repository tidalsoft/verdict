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
// Applicability (SPEC-MU §2.5.1: applies to money, gated on
// target_currency_field being declared -- §2.5.2 lists target_currency_field
// among the attributes whose absence is a complete statement, "off by
// default, enabled per field," not a gap):
//   - no declaration for the field, or a declaration whose kind is not
//     money → not applicable.
//   - target_currency_field not declared → not applicable. This is the gate,
//     not a missing-input INDETERMINATE: an author who never declared this
//     attribute has said cross-currency comparison is not wanted here.
//
// MU-03 is not value-dependent (SPEC-MU §2.6.3's table): it never reads the
// field's own decimal value, only two sibling currency codes, so §2.6.3's
// coercion gate never applies to it and this function never consults
// in.ValueCoercionFailed.
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS:
//   - either side's sibling value is absent from Input.Vals, does not
//     resolve to a JSON string, or is not exactly three ASCII letters →
//     INDETERMINATE ("Either unresolvable → INDETERMINATE"). This covers an
//     undeclared currency_field too, per the "Requires currency_field
//     declaration" clause: currency_field is a required input, unlike
//     target_currency_field, because a money field always has a source
//     currency question -- it is target_currency_field's declaration alone
//     that decides whether MU-03 runs at all.
//   - both resolve and, ASCII-uppercased, are equal → PASS.
//   - both resolve and, ASCII-uppercased, are unequal → FAIL.
//
// # Currency-code resolution is shape, not table membership
//
// SPEC-MU §2.4.2 defines "a currency code" once, for every check that
// needs one: "what a path resolves to when it yields a JSON string of
// exactly three ASCII letters... Three ASCII letters is a shape, not
// membership. A code of the right shape that the pinned ISO 4217 table
// does not carry resolves as a currency code." MU-03 only ever compares
// two currency codes for equality -- it needs neither of them to name a
// real, tabled currency, only to agree or disagree with each other -- so
// this check resolves both sides by shape alone (currencyCodeShape,
// below), never through the injected ISO 4217 table. MU-07's
// minor_units branch and MU-14 are the two checks that read a currency's
// minor-unit exponent, and they alone need the table lookup
// resolveDeclaredCurrency performs; see that function's own doc comment
// in mu.go for the enumeration of every consumer this hazard note
// requires.
func checkMU03(in Input) (verdict.Result, bool, error) {
	moneyDecl, ok := moneyDeclaration(in)
	if !ok {
		return notApplicable()
	}
	if _, hasTarget := moneyDecl.TargetCurrencyField(); !hasTarget {
		return notApplicable()
	}

	source, ok := resolveCurrencyCodeShape(in, moneyDecl.CurrencyField)
	if !ok {
		return indeterminateResult("MU-03")
	}
	target, ok := resolveCurrencyCodeShape(in, moneyDecl.TargetCurrencyField)
	if !ok {
		return indeterminateResult("MU-03")
	}

	if source != target {
		return failResult("MU-03")
	}
	return passResult("MU-03")
}

// currencyCodeShape reports whether code has the shape SPEC-MU §2.4.2
// defines for a currency code -- exactly three ASCII letters -- and, if
// so, returns it ASCII-uppercased. It performs no table lookup: shape is
// the entire test, deliberately independent of whether the pinned ISO
// 4217 table happens to recognise the result (see checkMU03's doc
// comment for why that independence matters specifically to MU-03).
func currencyCodeShape(code string) (string, bool) {
	if len(code) != 3 {
		return "", false
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return "", false
		}
	}
	return asciiUpper(code), true
}

// resolveCurrencyCodeShape resolves the currency code named by a
// currency-field accessor (MoneyDeclaration.CurrencyField or
// TargetCurrencyField) through Input.Vals, by shape alone -- see
// currencyCodeShape. This is checkMU03's one consumer; every other check
// needing a currency resolves through resolveDeclaredCurrency (below)
// instead, which additionally requires ISO 4217 table membership for the
// minor-unit exponent those checks read.
func resolveCurrencyCodeShape(in Input, accessor func() (string, bool)) (string, bool) {
	path, has := accessor()
	if !has {
		return "", false
	}
	v, ok := in.Vals[path]
	if !ok {
		return "", false
	}
	s, ok := v.StringValue()
	if !ok {
		return "", false
	}
	return currencyCodeShape(s)
}

// resolveDeclaredCurrency resolves the currency named by a currency-field
// accessor -- MoneyDeclaration.CurrencyField or
// MoneyDeclaration.TargetCurrencyField, both `func() (string, bool)` --
// through the declared sibling path, looked up in in.Vals, looked up in
// the injected ISO 4217 table. Any missing step collapses to a single
// false.
//
// # Every consumer of this resolution helper (the hazard section's
// # required trace)
//
// This is the currency resolution every check in this package that needs
// a currency's ISO 4217 *minor-unit exponent* performs -- as opposed to
// checkMU03's resolveCurrencyCodeShape above, which needs only shape and
// equality, never the table:
//
//   - MU-07 (checkMU07Money, range.go), on its scale: minor_units branch
//     only, to scale a major-units bound into minor units. A currency
//     whose code has the right shape but is not an ISO 4217 table member
//     still returns ok == false here (unlike currencyCodeShape), and
//     checkMU07Money reports MU-07 INDETERMINATE for want of an exponent
//     to scale by -- never a guessed exponent, never a FAIL, never a
//     PASS.
//   - MU-14 (checkMU14, exponent.go), to look up the exponent it compares
//     the value's decimal places against. Same false → INDETERMINATE
//     contract.
func resolveDeclaredCurrency(in Input, accessor func() (string, bool)) (tables.Currency, bool) {
	path, has := accessor()
	if !has {
		return tables.Currency{}, false
	}
	v, ok := in.Vals[path]
	if !ok {
		return tables.Currency{}, false
	}
	code, ok := v.StringValue()
	if !ok {
		return tables.Currency{}, false
	}
	return in.Tables.resolveCurrency(code)
}
