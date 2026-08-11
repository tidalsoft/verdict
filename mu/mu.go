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
// (Value, Provenance), the unit embedded in the value's own transported
// text if the field is a decomposed quantity (EmbeddedUnit,
// HasEmbeddedUnit), the ruleset's field declarations (Registry), the
// injected reference tables (Tables), and the sibling and state-derived
// values the checks may consult (Vals).
//
// Vals maps a field path to its resolved field.Value, and carries two
// distinct kinds of data under that one map: sibling values from the same
// tool call (e.g. "arguments.type" → "refund" for MU-06's sign_when
// conditions, "arguments.currency" → "USD" for MU-14's currency_field, or
// a unit_field's resolved unit string for MU-04/MU-05/MU-07/MU-15), and
// state-derived data the caller has flattened into the same map under
// whatever path a declaration names (e.g. MU-03's target_currency_field --
// see field.MoneyDeclaration.TargetCurrencyField's doc comment for why:
// this package has no representation of "state" distinct from a value in
// Vals). Absence from Vals is absence, never a present-but-empty Value: a
// check that needs a value it cannot find returns INDETERMINATE rather
// than guessing (the comma-ok idiom, mirroring field.Registry.Lookup).
// SPEC-MU §2.6.1's resolution rule -- a path resolves to a currency code,
// or a unit, only where it yields a JSON string; a sequence, another JSON
// array, or a JSON object leaves it unresolved -- is the caller's
// responsibility to apply before populating Vals: a caller building Vals
// from a request simply omits an entry whose path resolved to one of
// those non-string shapes, exactly as it omits a path absent from the
// request entirely. field.Value's own richer shapes (ValueKindNumber,
// ValueKindBool, ValueKindNonComparable) exist for MU-06's `sign_when`,
// whose comparable-value rule needs to tell a resolved JSON object apart
// from a resolved string (SPEC-MU §3, vector 118) -- see sign.go.
//
// EmbeddedUnit/HasEmbeddedUnit is the unit part of a `kind: quantity`
// value decomposed from its own transported text (SPEC-MU §2.6.1) -- the
// "lb" in "12 lb" -- populated by whatever constructs Input, exactly as
// Value/Provenance already are: this package performs no string
// decomposition of its own. See resolveQuantityUnit (unit.go), the one
// place every quantity check reads both EmbeddedUnit and unit_field's
// entry in Vals to resolve a field's unit.
//
// ValueCoercionFailed marks a field whose value SPEC-MU §2.6.2's
// coercion gate could not turn into an exact decimal at all -- a string
// resolution refused (`"1,234"`), or a value that arrived as neither a
// JSON string nor a JSON number. Coercion itself happens upstream of this
// package, exactly as decomposition and resolution do: this field is the
// caller's report of that outcome, not something this package computes.
// Its zero value, false, means "coerced, Value is meaningful" -- matching
// every Input literal that predates this field and never mentions it, and
// matching the ordinary case, where a value that came in fine says
// nothing extra about itself. When true, Value and Provenance carry no
// meaning and must not be read: SPEC-MU §2.6.3 requires every applicable
// value-dependent check on the field (§2.6.3's own table: MU-01, MU-02,
// MU-06, MU-07, MU-13, MU-14, MU-15 among the checks this package
// implements) to return INDETERMINATE without looking at them, rather
// than evaluate against a number nobody can pin down. MU-03, MU-04, and
// MU-05 are not value-dependent (§2.6.3) and never consult this field.
type Input struct {
	Field               string
	Value               decimal.Decimal
	Provenance          decimal.Provenance
	ValueCoercionFailed bool
	EmbeddedUnit        string
	HasEmbeddedUnit     bool
	Registry            field.Registry
	Tables              Tables
	Vals                map[string]field.Value
}

// Tables is the set of reference tables a single evaluation may consult.
// It is constructed once by the caller and injected — never built inside a
// check or in a hot path. Its zero value is safe: a zero CurrencyTable or
// UnitRegistry lookup simply misses, so currency- and unit-dependent
// checks evaluate to INDETERMINATE (or, for MU-04, FAIL -- an unrecognised
// unit is unsafe to pass through) rather than panicking.
type Tables struct {
	ISO4217 tables.CurrencyTable
	Units   tables.UnitRegistry
}

// OnFunc is the type of a single magnitude/unit check: a pure function of
// an Input producing a verdict.Result and whether the check applied at all,
// or an error. It is the entry type of the dispatch table keyed by
// field.Kind.
//
// applicable is false exactly when SPEC-MU §2.5's applicability test
// failed: the field's declared kind is not in the check's *Applies to*
// set, or a §2.5.1 "Applicable only when" gate is not satisfied because
// the gating attribute is a coherent, complete declaration by its very
// absence (§2.5.2 -- e.g. MU-07 with neither `min` nor `max` declared:
// "a field with neither is simply unbounded," not a gap in the
// declaration). When applicable is false, result is the zero Result and
// must never be read: Evaluate omits it from its returned slice entirely,
// so nothing downstream can mistake it for a verdict. See notApplicable.
//
// applicable is true for every other outcome, including INDETERMINATE: a
// check that applies but could not resolve what it needed -- a required
// input absent in the §2.5.2 sense (a money field's `scale`, a quantity's
// `dimension`, a value SPEC-MU §2.6.3's coercion gate could not read) --
// still returns a real Result at applicable == true. Weakening this would
// undo invariant 1: INDETERMINATE never collapses to PASS, and it must
// not collapse into not-applicable's silence either. A check whose
// *Indeterminate when* subsection names a condition returns
// INDETERMINATE for it regardless of how the condition arose.
type OnFunc func(in Input) (result verdict.Result, applicable bool, err error)

// checkIDNoDeclaration is the checkID carried by the INDETERMINATE result
// Evaluate returns when the input field has no declaration in the Registry
// at all — the pre-dispatch absence case, where no specific check can be
// identified because the field's kind is unknown.
const checkIDNoDeclaration = "MU-00"

// Evaluate runs every check the input field's declared kind carries, in
// ascending check-ID order, and returns one Result per check — every
// applicable check runs, and every result is reported, including after a
// FAIL. SPEC-MU §2.6 requires exactly this: "evaluation does not
// short-circuit: all applicable checks run and all failures are reported,"
// specifically so that an agent correcting one reported error and
// retrying does not waste a round trip hitting a second error it could
// have fixed in the same pass. (§2.6's one exception — an MU-09 FAIL
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
//
// Beyond that pre-dispatch case, the returned slice holds one Result per
// *applicable* check only (SPEC-MU §2.5) — see OnFunc's doc comment for
// what distinguishes "not applicable" (no entry at all) from
// INDETERMINATE (a real entry reporting a blind spot).
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

// notApplicable is the (verdict.Result, bool, error) triple every check
// returns the moment SPEC-MU §2.5's applicability test fails: the field's
// kind is outside the check's *Applies to* set, or a §2.5.1 "Applicable
// only when" gate reads false against a coherent, complete declaration
// (§2.5.2). See OnFunc's own doc comment for the full test this implements
// and why the zero Result it returns must never be read -- evaluateChecks
// is the one place that omission is enforced, so every check need only
// return this value and stop.
func notApplicable() (verdict.Result, bool, error) {
	return verdict.Result{}, false, nil
}

// checksFor returns the checks a field of the given kind carries, in
// ascending check-ID order — SPEC-MU §2.6: "every other applicable check,
// in ascending ID order" — and whether any checks apply at
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
		// SPEC-MU §2.5.1's trigger matrix lists MU-06 (sign_convention) and
		// MU-07 (range_bound) as applying to `decimal` exactly as to
		// `money` -- alongside MU-02, which the earlier task-1-4 dispatch
		// already carried.
		checks = []OnFunc{checkMU02, checkMU06, checkMU07}
	case field.KindPercentage:
		// MU-07 also applies to `percentage` (bounds in the declared
		// domain's units), ahead of MU-13 in ascending ID order.
		checks = []OnFunc{checkMU07, checkMU13}
	case field.KindQuantity:
		checks = []OnFunc{checkMU04, checkMU05, checkMU07, checkMU15}
	}
	return checks, len(checks) > 0
}

// evaluateChecks runs every check in checks against in, in order, and
// returns each *applicable* one's Result in that same order — SPEC-MU
// §2.6 forbids short-circuiting on a FAIL, so every applicable check must
// run and every result must be reported, not just the first FAIL or the
// first non-PASS. A check reporting applicable == false (OnFunc's own doc
// comment: SPEC-MU §2.5, "not applicable") contributes no entry at all —
// this is the one place in the package where that omission happens, so
// every check's own not-applicable branches funnel through here rather
// than each needing its own filtering. A check error aborts evaluation
// immediately and discards any results already collected: an error means
// a check could not produce a verdict at all (a programming or
// configuration defect, not an evaluation outcome), so a partial result
// set standing next to it would imply a completeness the evaluation never
// reached. An empty (or nil) checks list, or every check in it being not
// applicable, is not an error condition: unlike the single-Result design
// this replaced, which had no honest empty value and so had to invent an
// error for it, a slice already has one — "no checks ran, or none
// applied" is exactly the empty slice.
func evaluateChecks(in Input, checks []OnFunc) ([]verdict.Result, error) {
	results := make([]verdict.Result, 0, len(checks))
	for _, check := range checks {
		res, applicable, err := check(in)
		if err != nil {
			return nil, err
		}
		if !applicable {
			continue
		}
		results = append(results, res)
	}
	return results, nil
}

// moneyDeclaration looks up in.Field in the Registry and asserts the
// declaration found is a MoneyDeclaration, collapsing both failure modes
// -- no declaration at all, and a declaration for a different kind --
// into the single comma-ok signal every money-only check (MU-01, MU-03,
// MU-14) starts with and treats identically: not applicable (SPEC-MU
// §2.5.1 step 1 -- the field's kind is outside the check's Applies to set
// -- see notApplicable). There is nothing money-specific to say about a
// field that isn't declared as money in the first place, and "not
// applicable" is the state that says so without asserting a blind spot
// the ruleset never asked to have covered.
//
// MU-06 and MU-07 also apply to other kinds (SPEC-MU §2.5.1's trigger
// matrix: MU-06 to money and decimal; MU-07 to money, decimal,
// percentage, and quantity), so their check bodies branch on the
// declaration's concrete type themselves rather than calling this helper
// -- see checkMU06 (sign.go) and checkMU07 (range.go).
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

// boundsDeclaration is implemented by every field.Declaration kind MU-07
// (range_bound) applies to -- money, decimal, percentage, and quantity
// (SPEC-MU §2.5.1) -- each of which declares ExclusiveMin/ExclusiveMax
// identically. mu declares this interface itself, rather than importing a
// shared one from field, because "which kinds this check reaches" is
// mu's own dispatch decision (checkMU07, range.go), not a property field
// asserts about its declaration types.
type boundsDeclaration interface {
	ExclusiveMin() bool
	ExclusiveMax() bool
}

// bounded evaluates MU-07's range_bound comparison itself: value against
// min and max, both already expressed in the same units as value by the
// time they reach this function, with inclusive semantics unless decl's
// ExclusiveMin/ExclusiveMax flip equality at that boundary. hasMin/hasMax
// mirror checkMU07's own comma-ok read of decl.Min()/decl.Max() -- passed
// through explicitly, rather than re-derived here from decl, because decl
// itself always carries the raw, un-normalized bound: getting min and max
// into value's units at all is checkMU07's job, not this function's (see
// its doc comment for why, and how). decl is still passed in, for
// ExclusiveMin()/ExclusiveMax() alone.
//
// SPEC-MU §3 MU-07's evaluation clause -- "Value below min or above max →
// FAIL" -- is exactly this comparison, performed in exact decimal
// arithmetic (decimal.Compare, whose own doc comment already cites
// MU-07): "never in floating point" is satisfied by that exactness, not
// by any unit conversion, which is why this function needs no currency
// and cannot itself fail -- unlike an earlier version of this function,
// which folded unit normalization in here too and could error on
// ScaleByExponent overflow.
//
// decl is boundsDeclaration rather than field.MoneyDeclaration -- SPEC-MU
// §2.5.1's trigger matrix applies MU-07 to money, decimal, percentage, and
// quantity alike, and all four field.*Declaration types expose
// ExclusiveMin/ExclusiveMax identically, so one comparison serves every
// kind's checkMU07 branch rather than four near-duplicates.
func bounded(decl boundsDeclaration, value decimal.Decimal, min decimal.Decimal, hasMin bool, max decimal.Decimal, hasMax bool) (verdict.Result, bool, error) {
	if hasMin {
		cmp := value.Compare(min)
		if cmp < 0 || (cmp == 0 && decl.ExclusiveMin()) {
			return failResult("MU-07")
		}
	}
	if hasMax {
		cmp := value.Compare(max)
		if cmp > 0 || (cmp == 0 && decl.ExclusiveMax()) {
			return failResult("MU-07")
		}
	}
	return passResult("MU-07")
}

// newResult builds a Class D result at block severity — the default for
// every check in this package except MU-15, whose own default severity is
// warn (see conversion.go, which uses warnResult exclusively instead) —
// with the given outcome. It is the single construction seam for every
// result this package produces at the block default severity.
func newResult(checkID string, outcome verdict.Outcome) (verdict.Result, error) {
	return verdict.NewResult(checkID, verdict.ClassD, verdict.SeverityBlock, outcome)
}

// warnResult builds a Class D result at warn severity. Two distinct cases
// in this package use it: MU-13's domain: hundred branch alone (SPEC-MU
// §3: "value ≤ 1 and value ≠ 0 → FAIL at severity warn only... A
// legitimate 0.5% exists; this is genuinely ambiguous in that direction,
// so it is asymmetric by design" — every other branch of MU-13 itself
// uses newResult's block default instead), and MU-15 (conversion.go) in
// its entirety, whose own declared default severity is warn, not block —
// every one of its Results, including PASS and INDETERMINATE, is built
// through this function rather than newResult.
// warnResult's, indeterminateResult's, failResult's, and passResult's bool
// return is always true: every one of them builds the Result for a check
// that SPEC-MU §2.5 has already determined is applicable (see OnFunc's doc
// comment) -- notApplicable is the only function in this package that ever
// returns false. They carry the bool at all, rather than leaving each call
// site to write `res, true, err`, so that every check function can return
// their result directly (`return failResult("MU-07")`) with a shape that
// already matches OnFunc, the same way mu.go's other constructors already
// did before this package needed a third return value.
func warnResult(checkID string, outcome verdict.Outcome) (verdict.Result, bool, error) {
	res, err := verdict.NewResult(checkID, verdict.ClassD, verdict.SeverityWarn, outcome)
	return res, true, err
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

// one returns the exact decimal constant 1 -- MU-13's unit-interval
// boundary (percentage.go), this package's only need for a fixed decimal
// literal. It takes no argument, unlike the mustDecimal(s string) helper
// it replaces: a zero-argument signature means no future call site can
// thread new, unvetted, data-dependent text through it the way a
// parameterized helper would invite. Getting a different constant out of
// this package requires writing a new function, a visible and deliberate
// act, not a one-line reuse of an existing helper with a different string.
//
// one() wraps mustParseDecimal, which retains the parse-and-panic
// mechanism (and its parameter) as an internal primitive: one()'s own
// literal, "1", can never actually make decimal.Parse fail, so the only
// way to exercise -- and therefore test -- that panic branch at all is
// through a caller that can pass it something invalid. mustParseDecimal is
// deliberately not called from anywhere else in this package; one() is its
// one real caller.
func one() decimal.Decimal {
	return mustParseDecimal("1")
}

// mustParseDecimal parses s as an exact decimal, panicking if s is not
// valid decimal text. It exists for the same "fail fast, fail loudly on a
// literal that should be infallible" reason mustResult does (see its doc
// comment): one()'s call to mustParseDecimal("1") can never actually reach
// the panic, so threading an `if err != nil` branch through one() itself
// would be dead code the coverage bar must reject. Deliberately not
// exported, nor added to the decimal package itself: decimal is a public
// Apache-2.0 library, and a general-purpose "build me a literal" seam
// there would be a permanent addition to its contract that only this
// package's one constant needs.
// TestMustParseDecimal_PanicsOnInvalidInput exercises the panic directly
// with deliberately invalid input, so the branch is real and tested, not
// merely unreachable in the sense that would violate the coverage bar.
func mustParseDecimal(s string) decimal.Decimal {
	d, err := decimal.Parse(s)
	if err != nil {
		panic(fmt.Sprintf("mu: mustParseDecimal(%q): %v", s, err))
	}
	return d
}

// indeterminateResult builds the INDETERMINATE result a check returns when
// it is applicable (SPEC-MU §2.5) but a required declaration, schema, table
// entry, or piece of state -- or the value itself, per §2.6.3's coercion
// gate -- was absent or unusable. See warnResult's doc comment for why its
// bool return is always true.
func indeterminateResult(checkID string) (verdict.Result, bool, error) {
	res, err := newResult(checkID, verdict.OutcomeIndeterminate)
	return res, true, err
}

// failResult builds the block-severity FAIL result a check returns when it
// finds a violation. See warnResult's doc comment for why its bool return
// is always true.
func failResult(checkID string) (verdict.Result, bool, error) {
	res, err := newResult(checkID, verdict.OutcomeFail)
	return res, true, err
}

// passResult builds the PASS result a check returns when it finds no
// violation. See warnResult's doc comment for why its bool return is
// always true.
func passResult(checkID string) (verdict.Result, bool, error) {
	res, err := newResult(checkID, verdict.OutcomePass)
	return res, true, err
}

// checkMU01, checkMU02, checkMU03, checkMU04, checkMU05, checkMU06,
// checkMU07, checkMU13, checkMU14, and checkMU15 are each implemented in
// their own file (scale.go, precision.go, currency.go, dimension.go,
// absent.go, sign.go, range.go, percentage.go, exponent.go, and
// conversion.go respectively), alongside that file's tests.
// resolveQuantityUnit (unit.go) is the one shared resolution helper every
// quantity check (MU-04, MU-05, MU-07's quantity branch, MU-15) calls to
// resolve a field's unit from its two possible sources -- see that file's
// doc comment.
