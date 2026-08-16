package field

import (
	"errors"
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// ReconcileDeclaration is the ruleset-level declaration for MU-12
// (total_reconciliation, SPEC-MU §2.4.3). Unlike every other declaration in
// this package, it is not one of Kind's six values and does not implement
// Declaration: it names three argument paths that together describe one
// reconciliation -- a total, a component collection, and an optional list
// of adjustments -- rather than describing one field's own magnitude
// semantics. A ruleset carries a list of these ("a request may reconcile
// more than one total," SPEC-MU §2.4.3), kept outside field.Registry, which
// holds exactly one Declaration per field path. MU-12 (mu package) is
// evaluated once per ReconcileDeclaration, against its Total path,
// independent of whatever Kind (if any) that path's own field.Registry
// entry declares -- SPEC-MU §2.3 is explicit that a reconcile entry's total
// path is evaluated "together with" ordinary declared fields, not only
// when it happens to have one of its own.
//
// Its zero value is not usable: NewReconcileDeclaration requires Total and
// Components, per SPEC-MU §2.4.3's table marking both *required*.
type ReconcileDeclaration struct {
	total      string
	components string

	adjustments []string // nil when absent; SPEC-MU §2.4.3's default is "empty list"

	// tolerance's zero value is the exact decimal 0, which is SPEC-MU
	// §2.4.3's own stated default for an un-set tolerance -- see
	// Tolerance's doc comment for why that coincidence means this field
	// needs no companion has-bool.
	tolerance decimal.Decimal
}

// NewReconcileDeclaration constructs a ReconcileDeclaration naming the
// total and components paths. Both must be non-empty. Chain
// WithAdjustments and WithTolerance to declare the remaining, optional
// attributes.
func NewReconcileDeclaration(total, components string) (ReconcileDeclaration, error) {
	if total == "" {
		return ReconcileDeclaration{}, errors.New("field: reconcile: total must not be empty")
	}
	if components == "" {
		return ReconcileDeclaration{}, errors.New("field: reconcile: components must not be empty")
	}
	return ReconcileDeclaration{total: total, components: components}, nil
}

// Total returns the path to the declared total.
func (d ReconcileDeclaration) Total() string { return d.total }

// Components returns the path -- ordinarily containing a `[*]` wildcard --
// to the values summed.
func (d ReconcileDeclaration) Components() string { return d.components }

// Adjustments returns the declared adjustment paths, in declaration order.
// Empty (never nil) when none were declared -- SPEC-MU §2.4.3's default,
// "empty list," is itself a meaningful, complete declaration (no
// adjustments apply to this reconciliation), not an absent required input,
// so this accessor carries no comma-ok: every ReconcileDeclaration has an
// Adjustments list, possibly of length zero. The returned slice is a
// defensive copy; mutating it has no effect on d.
func (d ReconcileDeclaration) Adjustments() []string {
	return append([]string(nil), d.adjustments...)
}

// Tolerance returns the declared absolute tolerance MU-12 compares the
// reconciliation delta against, defaulting to the exact decimal 0 when
// never declared (SPEC-MU §2.4.3: "Default when omitted: 0").
// decimal.Decimal's own zero value is already the exact 0, so an un-set
// tolerance and one explicitly declared "0" are indistinguishable here by
// construction -- MU-12 treats them identically either way, so this
// accessor carries no comma-ok either, unlike Min/Max elsewhere in this
// package, whose absence means "unconstrained" rather than "zero."
func (d ReconcileDeclaration) Tolerance() decimal.Decimal { return d.tolerance }

// WithAdjustments declares the paths added to the component sum. Every
// path must be non-empty. A defensive copy of paths is taken, so the
// caller's own slice may be reused or mutated afterward with no effect on
// the returned value.
func (d ReconcileDeclaration) WithAdjustments(paths []string) (ReconcileDeclaration, error) {
	for _, p := range paths {
		if p == "" {
			return ReconcileDeclaration{}, errors.New("field: reconcile: adjustment path must not be empty")
		}
	}
	d.adjustments = append([]string(nil), paths...)
	return d, nil
}

// WithTolerance declares the absolute tolerance MU-12 compares the
// reconciliation delta against. tolerance must not be negative.
//
// SPEC-MU defines MU-12's tolerance comparison only for a non-negative
// value; a negative tolerance is never given a meaning anywhere in the
// document, and left unvalidated it would let a ruleset silently disable
// reconciliation altogether (an arbitrarily large negative tolerance makes
// every delta "within tolerance"). decimal.Reconciles, this attribute's one
// consumer, documents tolerance as a caller precondition it deliberately
// does not itself validate, clamp, or guess about -- see that function's
// own doc comment. This package's constructors validate everything so that
// no half-valid object is ever reachable; rejecting a negative tolerance
// here, at construction, is that principle applied to the one place SPEC-MU
// leaves silent, so a ReconcileDeclaration that could reach evaluation with
// a negative tolerance is simply never constructible.
func (d ReconcileDeclaration) WithTolerance(tolerance decimal.Decimal) (ReconcileDeclaration, error) {
	if tolerance.Sign() < 0 {
		return ReconcileDeclaration{}, fmt.Errorf("field: reconcile: tolerance must not be negative, got %s", tolerance.String())
	}
	d.tolerance = tolerance
	return d, nil
}
