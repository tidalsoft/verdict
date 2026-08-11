package field

import (
	"errors"

	"github.com/tidalsoft/verdict/decimal"
)

// QuantityDeclaration is the field declaration for `kind: quantity`. Its
// zero value, produced by NewQuantityDeclaration, declares nothing beyond
// the kind itself.
//
// Dimension and CanonicalUnit are plain strings rather than closed enums
// on purpose: resolving a dimension or unit requires a unit registry this
// package does not have visibility into and does not provide (see
// tables.UnitRegistry). This package records what was declared; MU-04/
// MU-05/MU-07/MU-15 decide whether it resolves to anything.
//
// # U-25: Min/Max is MU-07's general bound, not MU-15's overflow check
//
// An earlier version of this type carried a single `max`/`hasMax` pair
// documented as "MU-15's overflow check" -- but SPEC-MU §4 MU-15
// (unit_conversion_overflow) no longer compares against any declared
// bound at all: that branch was deleted from the rule, precisely because
// it duplicated MU-07's own comparison on MU-07's own inputs, "reached by
// identical arithmetic," reporting one violation twice under two check
// IDs at two severities. What MU-15 tests now is round-trip precision
// only (see Tolerance below). MU-07's own *Requires* clause (SPEC-MU §3)
// is explicit that a quantity field's bounds are declared "in the
// canonical unit" -- so this is Min/Max in the ordinary SPEC-MU §2.4.2
// sense, symmetric with MoneyDeclaration/DecimalDeclaration/
// PercentageDeclaration's own Min/Max, not a dimension-specific overflow
// bound. checkMU07 (mu/range.go) is this pair's one consumer.
type QuantityDeclaration struct {
	common

	dimension    string
	hasDimension bool

	unitField    string
	hasUnitField bool

	canonicalUnit    string
	hasCanonicalUnit bool

	unitRequired bool

	min    decimal.Decimal
	hasMin bool
	max    decimal.Decimal
	hasMax bool

	exclusiveMin bool
	exclusiveMax bool

	tolerance    decimal.Decimal
	hasTolerance bool
}

// NewQuantityDeclaration returns a QuantityDeclaration with no attributes
// declared beyond kind: quantity. Chain With* methods to declare
// attributes.
func NewQuantityDeclaration() QuantityDeclaration { return QuantityDeclaration{} }

// Kind implements Declaration.
func (d QuantityDeclaration) Kind() Kind { return KindQuantity }

// Dimension returns the declared physical dimension (e.g. "mass"), if any.
// MU-04 returns INDETERMINATE when the second return value is false.
func (d QuantityDeclaration) Dimension() (string, bool) { return d.dimension, d.hasDimension }

// UnitField returns the path to the field naming this quantity's unit, if
// declared.
func (d QuantityDeclaration) UnitField() (string, bool) { return d.unitField, d.hasUnitField }

// CanonicalUnit returns the unit MU-04/MU-15 convert this quantity to for
// comparison, if declared.
func (d QuantityDeclaration) CanonicalUnit() (string, bool) {
	return d.canonicalUnit, d.hasCanonicalUnit
}

// UnitRequired reports whether MU-05 must fail a bare number on this field
// (`unit_required: true`).
func (d QuantityDeclaration) UnitRequired() bool { return d.unitRequired }

// Min returns the declared inclusive-unless-ExclusiveMin lower bound, if
// any (MU-07), expressed in the field's declared CanonicalUnit -- see the
// U-25 note on this type's doc comment.
func (d QuantityDeclaration) Min() (decimal.Decimal, bool) { return d.min, d.hasMin }

// Max returns the declared inclusive-unless-ExclusiveMax upper bound, if
// any (MU-07), expressed in the field's declared CanonicalUnit.
func (d QuantityDeclaration) Max() (decimal.Decimal, bool) { return d.max, d.hasMax }

// ExclusiveMin reports whether a declared Min excludes the boundary value
// itself. Meaningless (and false) when Min was never declared.
func (d QuantityDeclaration) ExclusiveMin() bool { return d.exclusiveMin }

// ExclusiveMax reports whether a declared Max excludes the boundary value
// itself. Meaningless (and false) when Max was never declared.
func (d QuantityDeclaration) ExclusiveMax() bool { return d.exclusiveMax }

// Tolerance returns the declared round-trip tolerance for MU-15's
// unit_conversion_overflow check, if declared. The default (1 part in
// 10^9 when absent) is applied by MU-15 itself, not this package, since
// the default is a check behaviour, not part of what the ruleset
// declared.
func (d QuantityDeclaration) Tolerance() (decimal.Decimal, bool) { return d.tolerance, d.hasTolerance }

// WithDimension declares the field's physical dimension. dim must be
// non-empty.
func (d QuantityDeclaration) WithDimension(dim string) (QuantityDeclaration, error) {
	if dim == "" {
		return QuantityDeclaration{}, errors.New("field: dimension must not be empty")
	}
	d.dimension = dim
	d.hasDimension = true
	return d, nil
}

// WithUnitField declares the path to this quantity's unit field. path must
// be non-empty.
func (d QuantityDeclaration) WithUnitField(path string) (QuantityDeclaration, error) {
	if path == "" {
		return QuantityDeclaration{}, errors.New("field: unit_field must not be empty")
	}
	d.unitField = path
	d.hasUnitField = true
	return d, nil
}

// WithCanonicalUnit declares the unit this quantity is converted to for
// comparison. unit must be non-empty.
func (d QuantityDeclaration) WithCanonicalUnit(unit string) (QuantityDeclaration, error) {
	if unit == "" {
		return QuantityDeclaration{}, errors.New("field: canonical_unit must not be empty")
	}
	d.canonicalUnit = unit
	d.hasCanonicalUnit = true
	return d, nil
}

// WithUnitRequired declares that a bare number without a resolvable unit
// must fail MU-05.
func (d QuantityDeclaration) WithUnitRequired() QuantityDeclaration {
	d.unitRequired = true
	return d
}

// WithMin declares the field's lower bound (MU-07), in CanonicalUnit's
// units.
func (d QuantityDeclaration) WithMin(min decimal.Decimal) QuantityDeclaration {
	d.min = min
	d.hasMin = true
	return d
}

// WithMax declares the field's upper bound (MU-07), in CanonicalUnit's
// units.
func (d QuantityDeclaration) WithMax(max decimal.Decimal) QuantityDeclaration {
	d.max = max
	d.hasMax = true
	return d
}

// WithExclusiveMin marks a declared Min as excluding the boundary value.
func (d QuantityDeclaration) WithExclusiveMin() QuantityDeclaration {
	d.exclusiveMin = true
	return d
}

// WithExclusiveMax marks a declared Max as excluding the boundary value.
func (d QuantityDeclaration) WithExclusiveMax() QuantityDeclaration {
	d.exclusiveMax = true
	return d
}

// WithTolerance declares the round-trip tolerance MU-15 enforces.
func (d QuantityDeclaration) WithTolerance(tolerance decimal.Decimal) QuantityDeclaration {
	d.tolerance = tolerance
	d.hasTolerance = true
	return d
}

// WithNullSemantics declares the field's null-vs-absent handling (MU-08).
// n must be NullSemanticsDistinct.
func (d QuantityDeclaration) WithNullSemantics(n NullSemantics) (QuantityDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return QuantityDeclaration{}, err
	}
	d.common = c
	return d, nil
}
