package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// signDeclaration is implemented by every field.Declaration kind MU-06
// (sign_convention) applies to -- money and decimal (SPEC-MU §2.5.1) --
// each of which declares Sign/SignWhen/Nonzero identically. mu declares
// this interface itself, rather than importing a shared one from field,
// because "which kinds this check reaches" is mu's own dispatch decision,
// not a property field asserts about its declaration types (percentage,
// quantity, timestamp, and identifier declarations do not implement this
// interface at all, so the type assertion in checkMU06 below is itself
// the applicability gate for every other kind).
type signDeclaration interface {
	Sign() (field.Sign, bool)
	SignWhen() []field.ConditionalSign
	Nonzero() bool
}

// checkMU06 implements the sign_convention check (MU-06, SPEC-MU §3).
//
// MU-06 rejects sign inversions -- a refund recorded as a positive charge,
// or a credit as a debit.
//
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is
//     neither money nor decimal → INDETERMINATE.
//   - no governing sign resolves (see resolveSign) → INDETERMINATE.
//   - the resolved sign is violated by the value (see signViolated) →
//     FAIL.
//   - otherwise → PASS.
//
// # Precedence between Sign and SignWhen
//
// SPEC-MU §3 settles what an earlier version of this check (and this
// codebase) left as an open reading: "a matching clause wins; otherwise
// if any clause is undecidable the result is INDETERMINATE; otherwise the
// unconditional sign governs." The unconditional `sign` is therefore a
// *fallback*, consulted only once every sign_when clause has been
// checked and none of them either matched or left the answer genuinely
// unknown -- never dead text the moment sign_when is declared at all, and
// never bypassed by an undecidable clause either. See resolveSign for the
// four-rule precedence this implements, and clauseState for how a single
// clause's three-state taxonomy (matches / undecidable / definitively
// does not match) is computed.
func checkMU06(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-06")
	}
	signDecl, ok := decl.(signDeclaration)
	if !ok {
		return indeterminateResult("MU-06")
	}

	sign, applicable := resolveSign(signDecl, in.Vals)
	if !applicable {
		return indeterminateResult("MU-06")
	}

	if signViolated(sign, in.Value, signDecl.Nonzero()) {
		return failResult("MU-06")
	}
	return passResult("MU-06")
}

// clauseVerdict is a single sign_when clause's state against a request --
// SPEC-MU §3's three-state taxonomy, computed once per clause by
// clauseState.
type clauseVerdict int

const (
	clauseDoesNotMatch clauseVerdict = iota
	clauseUndecidable
	clauseMatches
)

// clauseState computes SPEC-MU §3's three-state taxonomy for a single
// sign_when clause's `when` entries against vals: "Partition the map's
// entries into three sets -- those that compare and agree, those that
// compare and contradict, and those that either do not resolve or do not
// compare at all -- and read them in this order: contradicting set
// non-empty → definitively does not match; otherwise the third set
// non-empty → undecidable; otherwise → matches." A single contradicting
// entry rules the clause out regardless of what any other entry in the
// same clause does (vector 91) -- which is why this loop never returns
// early on an unresolved or non-comparable entry, only on a contradiction:
// an unresolved entry earlier in the list must not pre-empt a
// contradiction found later.
func clauseState(entries []field.WhenEntry, vals map[string]field.Value) clauseVerdict {
	anyUndecidable := false
	for _, entry := range entries {
		resolved, ok := vals[entry.Path()]
		if !ok || !resolved.Comparable() || !entry.Value().Comparable() {
			anyUndecidable = true
			continue
		}
		if !resolved.Equal(entry.Value()) {
			return clauseDoesNotMatch
		}
	}
	if anyUndecidable {
		return clauseUndecidable
	}
	return clauseMatches
}

// resolveSign determines which field.Sign governs this evaluation, and
// whether any does at all, applying SPEC-MU §3's four ordered rules:
//
//  1. The earliest clause (in declaration order) whose clauseState is
//     clauseMatches governs; later clauses are not consulted once one
//     matches, regardless of any undecidable clause encountered earlier
//     in the scan (rule 1 has absolute priority over rule 2).
//  2. Otherwise, if any clause scanned was clauseUndecidable, no sign is
//     established (INDETERMINATE) -- checked before the unconditional
//     Sign fallback below, so a fallback present alongside an undecidable
//     clause does not rescue it (vectors 81, 113, 118).
//  3. Otherwise (every clause, if any, definitively does not match), an
//     unconditional Sign, if declared, governs (vectors 79, 80).
//  4. Otherwise (no clause matched, none was undecidable, and no
//     unconditional Sign is declared -- including when SignWhen was never
//     declared at all, an empty clause list satisfying this vacuously),
//     no sign is established.
func resolveSign(decl signDeclaration, vals map[string]field.Value) (field.Sign, bool) {
	anyUndecidable := false
	for _, clause := range decl.SignWhen() {
		switch clauseState(clause.When(), vals) {
		case clauseMatches:
			return clause.Sign(), true
		case clauseUndecidable:
			anyUndecidable = true
		case clauseDoesNotMatch:
			// No effect: this clause contributes neither a governing sign
			// nor an undecidable mark.
		}
	}
	if anyUndecidable {
		return field.SignUnspecified, false
	}
	return decl.Sign()
}

// signViolated reports whether value violates the required sign. SPEC-MU
// §3: "Zero is permitted under both positive and negative unless nonzero:
// true is also declared" -- nonzero is applied uniformly ahead of the
// sign-specific comparison, since it is documented as an independent
// constraint on zero, not one scoped only to positive/negative. Negative
// zero is zero (SPEC-MU §2.6.1) and IsZero() already treats it as such, so
// no separate check is needed here.
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
