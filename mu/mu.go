package mu

import (
	"fmt"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// Input is the per-field evaluation context every check receives: which
// field is being evaluated (Field), its value and how that value arrived
// (Value, Provenance), the ruleset's field declarations (Registry), the
// injected reference tables (Tables), and the sibling field values the
// checks may consult (Vals).
//
// Vals maps a sibling field path to its raw string value — e.g.
// "arguments.type" → "refund" for MU-06's sign_when conditions, or
// "arguments.currency" → "USD" for MU-03's target resolution. Absence from
// Vals is absence, never an empty string: a check that needs a sibling
// value it cannot find returns INDETERMINATE rather than guessing (the
// comma-ok idiom, mirroring field.Registry.Lookup).
type Input struct {
	Field      string
	Value      decimal.Decimal
	Provenance decimal.Provenance
	Registry   field.Registry
	Tables     Tables
	Vals       map[string]string
}

// Tables is the set of reference tables a single evaluation may consult.
// It is constructed once by the caller and injected — never built inside a
// check or in a hot path. Its zero value is safe: a zero CurrencyTable
// lookup simply misses, so currency-dependent checks evaluate to
// INDETERMINATE rather than panicking.
type Tables struct {
	ISO4217 tables.CurrencyTable
}

// OnFunc is the type of a single magnitude/unit check: a pure function of
// an Input producing a verdict.Result, or an error. It is the entry type of
// the dispatch table keyed by field.Kind.
type OnFunc func(Input) (verdict.Result, error)

// checkIDNoDeclaration is the checkID carried by the INDETERMINATE result
// Evaluate returns when the input field has no declaration in the Registry
// at all — the pre-dispatch absence case, where no specific check can be
// identified because the field's kind is unknown.
const checkIDNoDeclaration = "MU-00"

// Evaluate runs every check the input field's declared kind carries, in
// ascending check-ID order, and returns one Result per check — every
// applicable check runs, and every result is reported, including after a
// FAIL. SPEC-MU §2.4 requires exactly this: "Evaluation does not
// short-circuit: all applicable checks run and all failures are reported,"
// specifically so that an agent correcting one reported error and
// retrying does not waste a round trip hitting a second error it could
// have fixed in the same pass. (§2.4's one exception — an MU-09 FAIL
// forcing other numeric checks on the same field to INDETERMINATE — is
// MU-09's own concern; it has no check in this package yet and nothing
// here anticipates it.)
//
// A field with no declaration in the Registry evaluates to a single
// INDETERMINATE result carrying checkIDNoDeclaration — never a panic,
// never a guessed kind, and never zero results: "no declaration" is
// itself the one thing worth reporting when no check ran at all, so it is
// reported as a one-element slice rather than an empty one. A declared
// kind this package has no checks for (timestamp, identifier) is an
// error: the caller asked mu to evaluate a field mu cannot.
func Evaluate(in Input) ([]verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return []verdict.Result{mustResult(checkIDNoDeclaration, verdict.OutcomeIndeterminate)}, nil
	}
	checks, ok := checksFor(decl.Kind())
	if !ok {
		return nil, fmt.Errorf("mu: no checks for field kind %q", decl.Kind())
	}
	return evaluateChecks(in, checks)
}

// checksFor returns the checks a field of the given kind carries, in
// ascending check-ID order — SPEC-MU §2.4: "Checks are evaluated in
// ascending ID order within a field" — and whether any checks apply at
// all. The second return value is false for kinds this package has no
// checks for (timestamp, identifier, and the zero value). It is a
// function rather than a package-level map because this module forbids
// package-level state (gochecknoglobals); the switch is the dispatch
// table.
//
// The second return value is deliberately derived from len(checks) > 0
// rather than written as a literal true/false alongside each case. A
// case that returned (someSlice, true) by hand could drift out of sync
// with someSlice -- in particular, an empty slice paired with true. That
// combination is not a harmless edge case: evaluateChecks would pass it
// straight through as a valid, empty []verdict.Result, and
// verdict.ComputeAggregate documents that an empty result slice reads as
// VerdictAllow -- exactly the outcome invariant 1 (INDETERMINATE never
// collapses to PASS, and here nothing ran at all) exists to prevent. This
// function will be edited again in T3/T4/T5 as new kinds and checks are
// added to the switch; deriving ok structurally means a future case with
// a wrong or forgotten check list can produce a wrong check list, but it
// cannot produce this specific hazard.
func checksFor(kind field.Kind) ([]OnFunc, bool) {
	var checks []OnFunc
	switch kind {
	case field.KindMoney:
		checks = []OnFunc{checkMU01, checkMU02, checkMU03, checkMU06, checkMU07, checkMU14}
	case field.KindDecimal:
		checks = []OnFunc{checkMU02}
	case field.KindPercentage:
		checks = []OnFunc{checkMU13}
	case field.KindQuantity:
		checks = []OnFunc{checkMU07}
	}
	return checks, len(checks) > 0
}

// evaluateChecks runs every check in checks against in, in order, and
// returns each one's Result in that same order — SPEC-MU §2.4 forbids
// short-circuiting on a FAIL, so every applicable check must run and
// every result must be reported, not just the first FAIL or the first
// non-PASS. A check error aborts evaluation immediately and discards any
// results already collected: an error means a check could not produce a
// verdict at all (a programming or configuration defect, not an
// evaluation outcome), so a partial result set standing next to it would
// imply a completeness the evaluation never reached. An empty (or nil)
// checks list is not an error condition: unlike the single-Result design
// this replaced, which had no honest empty value and so had to invent an
// error for it, a slice already has one — "no checks ran" is exactly the
// empty slice.
func evaluateChecks(in Input, checks []OnFunc) ([]verdict.Result, error) {
	results := make([]verdict.Result, 0, len(checks))
	for _, check := range checks {
		res, err := check(in)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, nil
}

// moneyDeclaration looks up in.Field in the Registry and asserts the
// declaration found is a MoneyDeclaration, collapsing both failure modes
// -- no declaration at all, and a declaration for a different kind --
// into the single comma-ok signal every money-only check (MU-01, MU-03,
// MU-06, MU-14) starts with and treats identically: INDETERMINATE. There
// is nothing money-specific to say about a field that isn't declared as
// money in the first place, so the two cases share one helper rather than
// each check re-deriving the same two-step lookup.
//
// MU-07 is deliberately not on that list even though checksFor dispatches
// it for field.KindMoney. checksFor also dispatches checkMU07 for
// field.KindQuantity (SPEC-MU §3 range_bound applies to both kinds), and
// field.QuantityDeclaration is a distinct concrete type from
// MoneyDeclaration -- moneyDeclaration(in) returns ok == false for every
// quantity field, so a checkMU07 body built on this helper would go
// INDETERMINATE forever for quantity fields, silently. checkMU07 (T4)
// needs its own kind dispatch (a type switch on the two declaration types,
// or two sibling helpers) rather than reusing this one.
func moneyDeclaration(in Input) (field.MoneyDeclaration, bool) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return field.MoneyDeclaration{}, false
	}
	moneyDecl, ok := decl.(field.MoneyDeclaration)
	return moneyDecl, ok
}

// resolveCurrency case-folds code's ASCII letters to upper case and looks
// the result up in the injected ISO 4217 table, returning the table's
// entry and whether the code resolved at all. ISO 4217 alphabetic codes
// are three plain ASCII letters (A-Z) by convention, and
// CurrencyTable.Lookup does not case-fold, so normalizing case is the
// caller's job — this is it.
//
// The fold is deliberately ASCII-only rather than strings.ToUpper's full
// Unicode simple case mapping. Unicode's case-mapping tables sometimes
// map a non-ASCII letter onto plain ASCII output — Turkish dotless ı
// (U+0131) upper-cases to ASCII "I", for instance — so strings.ToUpper
// can manufacture a valid-looking ISO code (e.g. "ıqd" → "IQD") out of
// text that was never ISO 4217 text at all. A check that then resolved
// that manufactured code and computed a real verdict off it (IQD's
// exponent, in that example) would report PASS or FAIL for a currency
// the input never actually named — a wrong, confident answer where
// SPEC-MU requires INDETERMINATE ("resolvable currency code" is a
// requirement MU-14 must actually meet, not one a case-mapping accident
// can satisfy on its behalf). Folding only [a-z] to [A-Z] byte-for-byte
// closes that off: no sequence of non-ASCII bytes can turn into an ASCII
// table key, because nothing outside the ASCII range is ever touched.
//
// It returns the Currency rather than the normalized string so that
// resolving a code is a single lookup with a single absence signal: a
// caller needing the canonical spelling (MU-03's cross-field comparison)
// reads it back off Currency.Code(), and no caller is left holding a code
// it must look up a second time in a branch that cannot fail.
func (t Tables) resolveCurrency(code string) (tables.Currency, bool) {
	return t.ISO4217.Lookup(asciiUpper(code))
}

// asciiUpper upper-cases only the ASCII letters 'a'-'z' in s, leaving
// every other byte untouched — including every byte of a multi-byte UTF-8
// sequence, none of which fall in the ASCII 'a'-'z' range (UTF-8
// continuation and multi-byte lead bytes are always ≥ 0x80), so a
// byte-for-byte pass here can never corrupt one. See resolveCurrency's
// doc comment for why a full Unicode case fold (strings.ToUpper) is not
// safe to use for this instead.
func asciiUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			b[i] = c - 'a' + 'A'
		}
	}
	return string(b)
}

// bounded evaluates MU-07's range_bound check for a money field: it
// normalizes the value and the declared bounds to minor units via
// ScaleByExponent using the currency's ISO 4217 minor-unit exponent, then
// compares with inclusive semantics unless ExclusiveMin/ExclusiveMax flip
// equality. It returns INDETERMINATE when no bound is declared or the
// currency carries no minor-unit exponent (XAU, XXX, ...), FAIL when the
// value falls outside a bound, and PASS otherwise. Scaling both sides by
// the same exponent preserves the comparison whether the inputs were
// already in minor units or in major units.
func bounded(decl field.MoneyDeclaration, value decimal.Decimal, currency tables.Currency) (verdict.Result, error) {
	min, hasMin := decl.Min()
	max, hasMax := decl.Max()
	if !hasMin && !hasMax {
		return indeterminateResult("MU-07")
	}
	exponent, ok := currency.MinorUnitExponent()
	if !ok {
		return indeterminateResult("MU-07")
	}
	valueMinor, err := value.ScaleByExponent(exponent)
	if err != nil {
		return verdict.Result{}, err
	}
	var minMinor, maxMinor decimal.Decimal
	if hasMin {
		minMinor, err = min.ScaleByExponent(exponent)
		if err != nil {
			return verdict.Result{}, err
		}
		cmp := valueMinor.Compare(minMinor)
		if cmp < 0 || (cmp == 0 && decl.ExclusiveMin()) {
			return failResult("MU-07")
		}
	}
	if hasMax {
		maxMinor, err = max.ScaleByExponent(exponent)
		if err != nil {
			return verdict.Result{}, err
		}
		cmp := valueMinor.Compare(maxMinor)
		if cmp > 0 || (cmp == 0 && decl.ExclusiveMax()) {
			return failResult("MU-07")
		}
	}
	return passResult("MU-07")
}

// newResult builds a Class D result at block severity — the default for all
// seven checks — with the given outcome. It is the single construction seam
// for every result this package produces.
func newResult(checkID string, outcome verdict.Outcome) (verdict.Result, error) {
	return verdict.NewResult(checkID, verdict.ClassD, verdict.SeverityBlock, outcome)
}

// mustResult builds a verdict.Result for a checkID and Outcome the caller
// already knows to be valid, panicking if they are not. Its signature
// carries no error at all — unlike newResult, indeterminateResult,
// failResult, and passResult, which exist for check bodies whose checkID
// is always one of this package's own fixed constants but whose
// correctness (a check body wired up right, a valid Outcome constant
// used) is exactly what those functions' error return lets a caller
// verify once, in a test, rather than trust by inspection everywhere it
// is called with an ad-hoc justification comment.
//
// mustResult is for the narrower case: a call site where any error would
// itself be unreachable given the literals passed, and threading an
// `if err != nil` branch through it would be dead code the coverage bar
// must reject. verdict.NewResult's only failure modes (verdict/result.go)
// are an empty checkID, an invalid Class/Severity/Outcome, or a Class-S
// result at block severity — every current call site here passes fixed
// literals that satisfy all four, so the panic below never fires in this
// package's own use. It exists anyway, rather than the error being
// silently discarded, because "fail fast, fail loudly" means a future
// call site that gets this wrong (e.g. a typo'd checkID constant) crashes
// immediately and loudly instead of returning a zero-value Result no one
// asked for. TestMustResult_PanicsOnInvalidInput exercises the panic
// directly with deliberately invalid input, so the branch is real,
// tested, and not unreachable in the sense that would violate the
// coverage bar — it is simply never taken by any legitimate caller.
func mustResult(checkID string, outcome verdict.Outcome) verdict.Result {
	res, err := newResult(checkID, outcome)
	if err != nil {
		panic(fmt.Sprintf("mu: mustResult(%q, %v): %v", checkID, outcome, err))
	}
	return res
}

// indeterminateResult builds the INDETERMINATE result a check returns when
// a required declaration, schema, or piece of state was absent.
func indeterminateResult(checkID string) (verdict.Result, error) {
	return newResult(checkID, verdict.OutcomeIndeterminate)
}

// failResult builds the block-severity FAIL result a check returns when it
// finds a violation.
func failResult(checkID string) (verdict.Result, error) {
	return newResult(checkID, verdict.OutcomeFail)
}

// passResult builds the PASS result a check returns when it finds no
// violation.
func passResult(checkID string) (verdict.Result, error) {
	return newResult(checkID, verdict.OutcomePass)
}

// The remaining check functions are placeholders: each returns
// INDETERMINATE until its real branch matrix lands (MU-02/MU-13 in T3,
// MU-06/MU-07 in T4, MU-03 in T5). Their signatures are the frozen
// contract; only the bodies change. MU-01 and MU-14 are implemented in
// scale.go and exponent.go respectively.
func checkMU02(_ Input) (verdict.Result, error) { return indeterminateResult("MU-02") }
func checkMU03(_ Input) (verdict.Result, error) { return indeterminateResult("MU-03") }
func checkMU13(_ Input) (verdict.Result, error) { return indeterminateResult("MU-13") }
func checkMU06(_ Input) (verdict.Result, error) { return indeterminateResult("MU-06") }
func checkMU07(_ Input) (verdict.Result, error) { return indeterminateResult("MU-07") }
