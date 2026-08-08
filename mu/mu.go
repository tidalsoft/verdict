package mu

import (
	"errors"
	"fmt"
	"strings"

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

// Evaluate runs every check the input field's declared kind carries, in the
// spec's internal order, and returns the first FAIL, else the first
// non-PASS (INDETERMINATE), else PASS. A field with no declaration in the
// Registry evaluates to INDETERMINATE — never a panic, never a guessed
// kind. A declared kind this package has no checks for (timestamp,
// identifier) is an error: the caller asked mu to evaluate a field mu
// cannot.
func Evaluate(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult(checkIDNoDeclaration)
	}
	checks, ok := checksFor(decl.Kind())
	if !ok {
		return verdict.Result{}, fmt.Errorf("mu: no checks for field kind %q", decl.Kind())
	}
	return evaluateChecks(in, checks)
}

// checksFor returns the ordered checks a field of the given kind carries,
// in the spec's internal order (MU-01 → MU-14 → MU-02 → MU-03 → MU-13 →
// MU-06 → MU-07), and whether any checks apply at all. The second return
// value is false for kinds this package has no checks for (timestamp,
// identifier, and the zero value). It is a function rather than a
// package-level map because this module forbids package-level state
// (gochecknoglobals); the switch is the dispatch table.
func checksFor(kind field.Kind) ([]OnFunc, bool) {
	switch kind {
	case field.KindMoney:
		return []OnFunc{checkMU01, checkMU14, checkMU02, checkMU03, checkMU06, checkMU07}, true
	case field.KindDecimal:
		return []OnFunc{checkMU02}, true
	case field.KindPercentage:
		return []OnFunc{checkMU13}, true
	case field.KindQuantity:
		return []OnFunc{checkMU07}, true
	default:
		return nil, false
	}
}

// evaluateChecks runs checks in order and aggregates: the first FAIL wins
// outright; otherwise the first INDETERMINATE is returned; otherwise the
// first PASS. A check error aborts evaluation. An empty check list is an
// error — Evaluate never produces one, but the function is total.
func evaluateChecks(in Input, checks []OnFunc) (verdict.Result, error) {
	var firstIndeterminate *verdict.Result
	var firstPass *verdict.Result
	for _, check := range checks {
		res, err := check(in)
		if err != nil {
			return verdict.Result{}, err
		}
		switch res.Outcome() {
		case verdict.OutcomeFail:
			return res, nil
		case verdict.OutcomeIndeterminate:
			if firstIndeterminate == nil {
				r := res
				firstIndeterminate = &r
			}
		case verdict.OutcomePass:
			if firstPass == nil {
				r := res
				firstPass = &r
			}
		}
	}
	if firstIndeterminate != nil {
		return *firstIndeterminate, nil
	}
	if firstPass != nil {
		return *firstPass, nil
	}
	return verdict.Result{}, errors.New("mu: no checks to evaluate")
}

// normalizeCurrency uppercases a currency code and looks it up in the
// injected ISO 4217 table. On an exact match it returns the normalized
// (uppercased) code and true; otherwise it returns the original code
// unchanged and false. ISO 4217 codes are uppercase by convention and
// CurrencyTable.Lookup does not case-fold, so normalization is the caller's
// job — this is it.
func (t Tables) normalizeCurrency(code string) (string, bool) {
	normalized := strings.ToUpper(code)
	_, ok := t.ISO4217.Lookup(normalized)
	if !ok {
		return code, false
	}
	return normalized, true
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

// The seven check functions are placeholders: each returns INDETERMINATE
// until its real branch matrix lands (MU-01/MU-14 in T2, MU-02/MU-13 in T3,
// MU-06/MU-07 in T4, MU-03 in T5). Their signatures are the frozen
// contract; only the bodies change.
func checkMU01(_ Input) (verdict.Result, error) { return indeterminateResult("MU-01") }
func checkMU14(_ Input) (verdict.Result, error) { return indeterminateResult("MU-14") }
func checkMU02(_ Input) (verdict.Result, error) { return indeterminateResult("MU-02") }
func checkMU03(_ Input) (verdict.Result, error) { return indeterminateResult("MU-03") }
func checkMU13(_ Input) (verdict.Result, error) { return indeterminateResult("MU-13") }
func checkMU06(_ Input) (verdict.Result, error) { return indeterminateResult("MU-06") }
func checkMU07(_ Input) (verdict.Result, error) { return indeterminateResult("MU-07") }
