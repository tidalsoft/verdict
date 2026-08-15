package field

import (
	"errors"
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/tables"
)

// QuantityDeclaration is the field declaration for `kind: quantity`. Its
// zero value, produced by NewQuantityDeclaration, declares nothing beyond
// the kind itself.
//
// Dimension and CanonicalUnit are both exposed as plain strings, but for
// two different reasons that no longer coincide. Dimension is validated
// against tables.Dimension's closed enumeration at construction (see
// WithDimension) -- SPEC-MU §4's fixed "Supported dimensions" list lives
// inside this module, so checking membership needs no external registry,
// exactly like WithScale/WithSign/WithDomain validate their own closed
// enums elsewhere in this package. It is still typed string rather than
// tables.Dimension here only to keep this package's declared surface
// (every accessor's comma-ok string/bool shape) uniform with
// CanonicalUnit and UnitField. CanonicalUnit remains genuinely
// unvalidated: whether a unit symbol resolves depends on
// tables.UnitRegistry's compiled-in data, which is versioned and can grow,
// and this package has no visibility into it and provides none -- MU-04/
// MU-05/MU-07/MU-15 (mu package) decide whether a declared CanonicalUnit
// or a request's own unit resolves to anything.
//
// # Min/Max is MU-07's general bound, not MU-15's overflow check
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

// Dimension returns the declared physical dimension (e.g. "mass"), if any,
// spelled the way tables.Dimension.String() renders it -- see
// WithDimension. MU-04 returns INDETERMINATE when the second return value
// is false.
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
// Min/Max note on this type's doc comment.
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

// WithDimension declares the field's physical dimension. dim must be one
// of SPEC-MU §4's "Supported dimensions" tokens (tables.ParseDimension's
// own doc comment lists them and the two whose spec spelling differs from
// this package's internal one).
//
// This validates at construction rather than leaving an unrecognised
// dimension to reach MU-04 (unit_dimension_mismatch) at evaluation: SPEC-MU
// §2.2 defines Class D's false-positive rate as zero by construction --
// "a FAIL means the input contradicts its own declaration" -- and a field
// declared dimension: "weight" (not a dimension this document defines)
// contradicts nothing about a value arriving in kg. §2.4.1 sets the
// pattern this follows for `kind` itself: "an unrecognised kind is a
// ruleset error, never a field the engine waves through unchecked."
// Rejecting the ruleset here is that same rule applied to `dimension`.
func (d QuantityDeclaration) WithDimension(dim string) (QuantityDeclaration, error) {
	parsed, ok := tables.ParseDimension(dim)
	if !ok {
		return QuantityDeclaration{}, fmt.Errorf("field: invalid dimension %q", dim)
	}
	d.dimension = parsed.String()
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
