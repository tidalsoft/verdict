package field

import (
	"errors"
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// DecimalDeclaration is the field declaration for `kind: decimal` (MU-02
// precision_loss). This kind is the escape hatch for a field that
// legitimately carries more decimal places than any currency's minor unit
// permits -- an intermediate calculation value, for instance -- so that
// declaring it does not trigger MU-01/MU-14's money-specific scale checks.
//
// SPEC-MU §2.4.2's attribute table legislates `sign`/`sign_when`/`nonzero`
// (MU-06 sign_convention) and `min`/`max`/`exclusive_min`/`exclusive_max`
// (MU-07 range_bound) as legal on `decimal` exactly as on `money` -- a
// decimal field's bounds and its own units coincide, so MU-07 needs
// nothing beyond them here, unlike money's scale/currency normalization.
type DecimalDeclaration struct {
	common

	sign    Sign
	hasSign bool

	signWhen []ConditionalSign // nil when absent

	nonzero bool

	min    decimal.Decimal
	hasMin bool
	max    decimal.Decimal
	hasMax bool

	exclusiveMin bool
	exclusiveMax bool
}

// NewDecimalDeclaration returns a DecimalDeclaration declaring kind:
// decimal and nothing else. Chain With* methods to declare attributes.
func NewDecimalDeclaration() DecimalDeclaration { return DecimalDeclaration{} }

// Kind implements Declaration.
func (d DecimalDeclaration) Kind() Kind { return KindDecimal }

// Sign returns the field's unconditional declared sign, if any (MU-06).
func (d DecimalDeclaration) Sign() (Sign, bool) { return d.sign, d.hasSign }

// SignWhen returns the field's conditional sign declarations, if any. The
// returned slice is a defensive copy; mutating it has no effect on d.
func (d DecimalDeclaration) SignWhen() []ConditionalSign {
	if d.signWhen == nil {
		return nil
	}
	return append([]ConditionalSign(nil), d.signWhen...)
}

// Nonzero reports whether a declared Sign forbids zero (MU-06).
func (d DecimalDeclaration) Nonzero() bool { return d.nonzero }

// Min returns the declared inclusive-unless-ExclusiveMin lower bound, if
// any (MU-07). A decimal field's bound is in the value's own units --
// unlike money or quantity, no further normalization input is required.
func (d DecimalDeclaration) Min() (decimal.Decimal, bool) { return d.min, d.hasMin }

// Max returns the declared inclusive-unless-ExclusiveMax upper bound, if
// any (MU-07).
func (d DecimalDeclaration) Max() (decimal.Decimal, bool) { return d.max, d.hasMax }

// ExclusiveMin reports whether a declared Min excludes the boundary value
// itself. Meaningless (and false) when Min was never declared.
func (d DecimalDeclaration) ExclusiveMin() bool { return d.exclusiveMin }

// ExclusiveMax reports whether a declared Max excludes the boundary value
// itself. Meaningless (and false) when Max was never declared.
func (d DecimalDeclaration) ExclusiveMax() bool { return d.exclusiveMax }

// WithSign declares the field's unconditional sign. s must be
// SignPositive, SignNegative, or SignAny.
func (d DecimalDeclaration) WithSign(s Sign) (DecimalDeclaration, error) {
	if !s.valid() {
		return DecimalDeclaration{}, fmt.Errorf("field: invalid sign %v", s)
	}
	d.sign = s
	d.hasSign = true
	return d, nil
}

// WithSignWhen declares the field's conditional sign rules. conds must be
// non-empty.
func (d DecimalDeclaration) WithSignWhen(conds []ConditionalSign) (DecimalDeclaration, error) {
	if len(conds) == 0 {
		return DecimalDeclaration{}, errors.New("field: sign_when must not be empty")
	}
	d.signWhen = append([]ConditionalSign(nil), conds...)
	return d, nil
}

// WithNonzero forbids zero under the field's declared Sign or SignWhen.
func (d DecimalDeclaration) WithNonzero() DecimalDeclaration {
	d.nonzero = true
	return d
}

// WithMin declares the field's lower bound.
func (d DecimalDeclaration) WithMin(min decimal.Decimal) DecimalDeclaration {
	d.min = min
	d.hasMin = true
	return d
}

// WithMax declares the field's upper bound.
func (d DecimalDeclaration) WithMax(max decimal.Decimal) DecimalDeclaration {
	d.max = max
	d.hasMax = true
	return d
}

// WithExclusiveMin marks a declared Min as excluding the boundary value.
func (d DecimalDeclaration) WithExclusiveMin() DecimalDeclaration {
	d.exclusiveMin = true
	return d
}

// WithExclusiveMax marks a declared Max as excluding the boundary value.
func (d DecimalDeclaration) WithExclusiveMax() DecimalDeclaration {
	d.exclusiveMax = true
	return d
}

// WithNullSemantics declares the field's null-vs-absent handling (MU-08).
// n must be NullSemanticsDistinct.
func (d DecimalDeclaration) WithNullSemantics(n NullSemantics) (DecimalDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return DecimalDeclaration{}, err
	}
	d.common = c
	return d, nil
}
