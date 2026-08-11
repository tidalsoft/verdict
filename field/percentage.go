package field

import (
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// Domain distinguishes a fractional (0-1) percentage representation from a
// hundred-scaled (0-100) one (MU-13 percentage_domain).
type Domain int

const (
	// DomainUnspecified is the zero value: no domain was declared.
	DomainUnspecified Domain = iota
	// DomainUnitInterval declares the value as a fraction in [0, 1] (e.g.
	// 0.5 for 50%).
	DomainUnitInterval
	// DomainHundred declares the value as already scaled to [0, 100] (e.g.
	// 50 for 50%).
	DomainHundred
)

// String renders the domain's canonical name.
func (d Domain) String() string {
	switch d {
	case DomainUnitInterval:
		return "unit_interval"
	case DomainHundred:
		return "hundred"
	default:
		return "unspecified"
	}
}

func (d Domain) valid() bool {
	return d == DomainUnitInterval || d == DomainHundred
}

// PercentageDeclaration is the field declaration for `kind: percentage`
// (MU-13). Its zero value, produced by NewPercentageDeclaration,
// declares nothing beyond the kind itself.
//
// SPEC-MU §2.4.2 also legislates `min`/`max`/`exclusive_min`/
// `exclusive_max` (MU-07 range_bound) on `percentage`: the bounds are "in
// the units of the declared domain," so MU-07 needs Domain resolved
// before it can even interpret Min/Max, but requires nothing else --
// unlike money's scale/currency normalization or quantity's unit
// conversion.
type PercentageDeclaration struct {
	common

	domain    Domain
	hasDomain bool

	min    decimal.Decimal
	hasMin bool
	max    decimal.Decimal
	hasMax bool

	exclusiveMin bool
	exclusiveMax bool
}

// NewPercentageDeclaration returns a PercentageDeclaration with no
// attributes declared beyond kind: percentage. Chain With* methods to
// declare attributes.
func NewPercentageDeclaration() PercentageDeclaration { return PercentageDeclaration{} }

// Kind implements Declaration.
func (d PercentageDeclaration) Kind() Kind { return KindPercentage }

// Domain returns the declared domain, if any. MU-13 requires both
// `domain: unit_interval` and `domain: hundred` to be explicitly declared
// -- there is no default -- so the second return value is false whenever
// the ruleset left this field's percentage domain ambiguous.
func (d PercentageDeclaration) Domain() (Domain, bool) { return d.domain, d.hasDomain }

// Min returns the declared inclusive-unless-ExclusiveMin lower bound, if
// any (MU-07), in the units of the declared Domain.
func (d PercentageDeclaration) Min() (decimal.Decimal, bool) { return d.min, d.hasMin }

// Max returns the declared inclusive-unless-ExclusiveMax upper bound, if
// any (MU-07).
func (d PercentageDeclaration) Max() (decimal.Decimal, bool) { return d.max, d.hasMax }

// ExclusiveMin reports whether a declared Min excludes the boundary value
// itself. Meaningless (and false) when Min was never declared.
func (d PercentageDeclaration) ExclusiveMin() bool { return d.exclusiveMin }

// ExclusiveMax reports whether a declared Max excludes the boundary value
// itself. Meaningless (and false) when Max was never declared.
func (d PercentageDeclaration) ExclusiveMax() bool { return d.exclusiveMax }

// WithDomain declares the field's percentage domain. dom must be
// DomainUnitInterval or DomainHundred.
func (d PercentageDeclaration) WithDomain(dom Domain) (PercentageDeclaration, error) {
	if !dom.valid() {
		return PercentageDeclaration{}, fmt.Errorf("field: invalid domain %v", dom)
	}
	d.domain = dom
	d.hasDomain = true
	return d, nil
}

// WithMin declares the field's lower bound.
func (d PercentageDeclaration) WithMin(min decimal.Decimal) PercentageDeclaration {
	d.min = min
	d.hasMin = true
	return d
}

// WithMax declares the field's upper bound.
func (d PercentageDeclaration) WithMax(max decimal.Decimal) PercentageDeclaration {
	d.max = max
	d.hasMax = true
	return d
}

// WithExclusiveMin marks a declared Min as excluding the boundary value.
func (d PercentageDeclaration) WithExclusiveMin() PercentageDeclaration {
	d.exclusiveMin = true
	return d
}

// WithExclusiveMax marks a declared Max as excluding the boundary value.
func (d PercentageDeclaration) WithExclusiveMax() PercentageDeclaration {
	d.exclusiveMax = true
	return d
}

// WithNullSemantics declares the field's null-vs-absent handling (MU-08).
// n must be NullSemanticsDistinct.
func (d PercentageDeclaration) WithNullSemantics(n NullSemantics) (PercentageDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return PercentageDeclaration{}, err
	}
	d.common = c
	return d, nil
}
