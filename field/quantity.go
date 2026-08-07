package field

import (
	"errors"

	"github.com/tidalsoft/gatepost/verdict/decimal"
)

// QuantityDeclaration is the field declaration for `kind: quantity`
// (SPEC-MU §4). Its zero value, produced by NewQuantityDeclaration,
// declares nothing beyond the kind itself.
//
// Dimension and CanonicalUnit are plain strings rather than closed enums
// on purpose: resolving a dimension or unit requires the unit registry
// (SPEC-MU §4's "Supported dimensions" list; task 1-5), which this package
// does not have visibility into. This package records what was declared;
// MU-04/MU-05 decide whether it resolves to anything.
type QuantityDeclaration struct {
	common

	dimension    string
	hasDimension bool

	unitField    string
	hasUnitField bool

	canonicalUnit    string
	hasCanonicalUnit bool

	unitRequired bool

	max    decimal.Decimal
	hasMax bool

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
// (SPEC-MU MU-05: `unit_required: true`).
func (d QuantityDeclaration) UnitRequired() bool { return d.unitRequired }

// Max returns the declared upper bound on the canonical-unit
// representation used by MU-15's overflow check, if any.
func (d QuantityDeclaration) Max() (decimal.Decimal, bool) { return d.max, d.hasMax }

// Tolerance returns the declared round-trip tolerance for MU-15's
// unit_conversion_overflow check, if declared. SPEC-MU §4 defaults this to
// 1 part in 10^9 when absent -- applying that default is MU-15's
// responsibility (task 1-5), not this package's, since the default is a
// check behaviour, not part of what the ruleset declared.
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

// WithMax declares the upper bound MU-15 enforces on the canonical-unit
// representation.
func (d QuantityDeclaration) WithMax(max decimal.Decimal) QuantityDeclaration {
	d.max = max
	d.hasMax = true
	return d
}

// WithTolerance declares the round-trip tolerance MU-15 enforces.
func (d QuantityDeclaration) WithTolerance(tolerance decimal.Decimal) QuantityDeclaration {
	d.tolerance = tolerance
	d.hasTolerance = true
	return d
}

// WithNullSemantics declares the field's null-vs-absent handling (SPEC-MU
// MU-08). n must be NullSemanticsDistinct.
func (d QuantityDeclaration) WithNullSemantics(n NullSemantics) (QuantityDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return QuantityDeclaration{}, err
	}
	d.common = c
	return d, nil
}
