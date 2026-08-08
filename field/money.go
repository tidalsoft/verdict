package field

import (
	"errors"
	"fmt"

	"github.com/tidalsoft/verdict/decimal"
)

// Scale distinguishes whether a money field's numeric value is expressed
// in a currency's minor units (e.g. cents) or its major units (e.g.
// dollars) (MU-01 scale_declaration_conflict).
type Scale int

const (
	// ScaleUnspecified is the zero value: no scale was declared.
	ScaleUnspecified Scale = iota
	// ScaleMinorUnits means the value is a count of the currency's minor
	// unit (e.g. 4999 for $49.99). Minor units are integers by
	// definition.
	ScaleMinorUnits
	// ScaleMajorUnits means the value is denominated in the currency's
	// major unit (e.g. 49.99 for $49.99), whose fractional part is bounded
	// by the currency's ISO 4217 minor-unit exponent (MU-14).
	ScaleMajorUnits
)

// String renders the scale's canonical name.
func (s Scale) String() string {
	switch s {
	case ScaleMinorUnits:
		return "minor_units"
	case ScaleMajorUnits:
		return "major_units"
	default:
		return "unspecified"
	}
}

func (s Scale) valid() bool {
	return s == ScaleMinorUnits || s == ScaleMajorUnits
}

// Sign is the permitted sign of a money field's value (MU-06
// sign_convention).
type Sign int

const (
	// SignUnspecified is the zero value: no sign was declared.
	SignUnspecified Sign = iota
	// SignPositive requires the value to be positive (zero permitted
	// unless Nonzero is also declared).
	SignPositive
	// SignNegative requires the value to be negative (zero permitted
	// unless Nonzero is also declared).
	SignNegative
	// SignAny permits either sign. Declaring it explicitly (rather than
	// simply not declaring a sign at all) is meaningful: it lets a
	// ConditionalSign's default branch say "any sign is fine here"
	// without the field falling back to INDETERMINATE.
	SignAny
)

// String renders the sign's canonical name.
func (s Sign) String() string {
	switch s {
	case SignPositive:
		return "positive"
	case SignNegative:
		return "negative"
	case SignAny:
		return "any"
	default:
		return "unspecified"
	}
}

func (s Sign) valid() bool {
	return s == SignPositive || s == SignNegative || s == SignAny
}

// ConditionalSign pairs a condition -- another field at the same argument
// level equalling a specific value -- with the Sign required when that
// condition holds (MU-06's sign_when form, e.g. "when
// arguments.type is refund, sign must be negative").
type ConditionalSign struct {
	whenField string
	whenValue string
	sign      Sign
}

// NewConditionalSign constructs a ConditionalSign. whenField must be
// non-empty; whenValue may be empty (a condition legitimately matching an
// empty string); sign must be one of SignPositive, SignNegative, or
// SignAny.
func NewConditionalSign(whenField, whenValue string, sign Sign) (ConditionalSign, error) {
	if whenField == "" {
		return ConditionalSign{}, errors.New("field: sign_when condition field must not be empty")
	}
	if !sign.valid() {
		return ConditionalSign{}, fmt.Errorf("field: sign_when condition on %q: invalid sign %v", whenField, sign)
	}
	return ConditionalSign{whenField: whenField, whenValue: whenValue, sign: sign}, nil
}

// WhenField returns the path of the field this condition inspects.
func (c ConditionalSign) WhenField() string { return c.whenField }

// WhenValue returns the value whenField must equal for this condition to
// match.
func (c ConditionalSign) WhenValue() string { return c.whenValue }

// Sign returns the sign required when this condition matches.
func (c ConditionalSign) Sign() Sign { return c.sign }

// MoneyDeclaration is the field declaration for `kind: money`.
// Its zero value, produced by NewMoneyDeclaration, declares nothing beyond
// the kind itself -- every attribute accessor's second return value is
// false until the corresponding With* method is called.
type MoneyDeclaration struct {
	common

	currencyField    string
	hasCurrencyField bool

	targetCurrencyField    string
	hasTargetCurrencyField bool

	scale    Scale
	hasScale bool

	sign    Sign
	hasSign bool

	signWhen []ConditionalSign // nil when absent

	min    decimal.Decimal
	hasMin bool
	max    decimal.Decimal
	hasMax bool

	exclusiveMin bool
	exclusiveMax bool
	nonzero      bool
}

// NewMoneyDeclaration returns a MoneyDeclaration with no attributes
// declared beyond kind: money. Chain With* methods to declare attributes.
func NewMoneyDeclaration() MoneyDeclaration { return MoneyDeclaration{} }

// Kind implements Declaration.
func (d MoneyDeclaration) Kind() Kind { return KindMoney }

// CurrencyField returns the path to the field naming this amount's
// currency, if declared.
func (d MoneyDeclaration) CurrencyField() (string, bool) {
	return d.currencyField, d.hasCurrencyField
}

// TargetCurrencyField returns the path to the field naming the currency
// this amount is being compared against, if declared (MU-03
// currency_mismatch's "target currency").
//
// SPEC-MU §3 MU-03 requires "a resolvable target currency, supplied either
// in state or as a second declared field," without saying how a
// declaration expresses either option. This attribute models the second
// option literally: a second field path, resolved through the same
// sibling-value mechanism as CurrencyField (see mu.Input.Vals). There is
// deliberately no separate representation of "state" here — this
// package's Registry and the mu package's Input carry no concept of
// account or system state distinct from a sibling field's value, so a
// caller wanting to compare against state-derived data supplies it the
// same way: as a value in Vals at the path this field names.
func (d MoneyDeclaration) TargetCurrencyField() (string, bool) {
	return d.targetCurrencyField, d.hasTargetCurrencyField
}

// Scale returns the declared scale (minor_units or major_units), if any.
// MU-01 returns INDETERMINATE when the second return value is false.
func (d MoneyDeclaration) Scale() (Scale, bool) { return d.scale, d.hasScale }

// Sign returns the field's unconditional declared sign, if any. Absent
// when the field instead (or additionally) declares SignWhen.
func (d MoneyDeclaration) Sign() (Sign, bool) { return d.sign, d.hasSign }

// SignWhen returns the field's conditional sign declarations, if any. The
// returned slice is a defensive copy; mutating it has no effect on d.
func (d MoneyDeclaration) SignWhen() []ConditionalSign {
	if d.signWhen == nil {
		return nil
	}
	return append([]ConditionalSign(nil), d.signWhen...)
}

// Min returns the declared inclusive-unless-ExclusiveMin lower bound, if
// any.
func (d MoneyDeclaration) Min() (decimal.Decimal, bool) { return d.min, d.hasMin }

// Max returns the declared inclusive-unless-ExclusiveMax upper bound, if
// any.
func (d MoneyDeclaration) Max() (decimal.Decimal, bool) { return d.max, d.hasMax }

// ExclusiveMin reports whether a declared Min excludes the boundary value
// itself. Meaningless (and false) when Min was never declared.
func (d MoneyDeclaration) ExclusiveMin() bool { return d.exclusiveMin }

// ExclusiveMax reports whether a declared Max excludes the boundary value
// itself. Meaningless (and false) when Max was never declared.
func (d MoneyDeclaration) ExclusiveMax() bool { return d.exclusiveMax }

// Nonzero reports whether a declared Sign forbids zero (MU-06:
// "Zero is permitted under both positive and negative unless nonzero: true
// is also declared").
func (d MoneyDeclaration) Nonzero() bool { return d.nonzero }

// WithCurrencyField declares the path to this amount's currency field.
// path must be non-empty, and must not equal an already-declared
// TargetCurrencyField -- see WithTargetCurrencyField for why.
func (d MoneyDeclaration) WithCurrencyField(path string) (MoneyDeclaration, error) {
	if path == "" {
		return MoneyDeclaration{}, errors.New("field: currency_field must not be empty")
	}
	if d.hasTargetCurrencyField && path == d.targetCurrencyField {
		return MoneyDeclaration{}, fmt.Errorf("field: currency_field must not equal target_currency_field (%q)", path)
	}
	d.currencyField = path
	d.hasCurrencyField = true
	return d, nil
}

// WithTargetCurrencyField declares the path to the field naming the
// currency this amount is compared against for MU-03. path must be
// non-empty, and must not equal an already-declared CurrencyField: a
// MoneyDeclaration whose currency_field and target_currency_field name the
// same path would compare a currency against itself, so MU-03 would PASS
// unconditionally without ever verifying anything -- exactly the
// "reports success while verifying nothing" failure mode SPEC-PG §2.1 and
// this codebase's invariants exist to prevent. Rejecting it here, at
// construction, means a MoneyDeclaration that could produce that outcome
// is simply never constructible (this package's "constructors validate
// everything; no half-initialized objects" principle), rather than a
// vacuous check silently shipping in a ruleset.
func (d MoneyDeclaration) WithTargetCurrencyField(path string) (MoneyDeclaration, error) {
	if path == "" {
		return MoneyDeclaration{}, errors.New("field: target_currency_field must not be empty")
	}
	if d.hasCurrencyField && path == d.currencyField {
		return MoneyDeclaration{}, fmt.Errorf("field: target_currency_field must not equal currency_field (%q)", path)
	}
	d.targetCurrencyField = path
	d.hasTargetCurrencyField = true
	return d, nil
}

// WithScale declares the field's scale. s must be ScaleMinorUnits or
// ScaleMajorUnits.
func (d MoneyDeclaration) WithScale(s Scale) (MoneyDeclaration, error) {
	if !s.valid() {
		return MoneyDeclaration{}, fmt.Errorf("field: invalid scale %v", s)
	}
	d.scale = s
	d.hasScale = true
	return d, nil
}

// WithSign declares the field's unconditional sign. s must be
// SignPositive, SignNegative, or SignAny.
func (d MoneyDeclaration) WithSign(s Sign) (MoneyDeclaration, error) {
	if !s.valid() {
		return MoneyDeclaration{}, fmt.Errorf("field: invalid sign %v", s)
	}
	d.sign = s
	d.hasSign = true
	return d, nil
}

// WithSignWhen declares the field's conditional sign rules. conds must be
// non-empty; each element is validated by NewConditionalSign before it can
// exist, so WithSignWhen itself only rejects an empty slice.
func (d MoneyDeclaration) WithSignWhen(conds []ConditionalSign) (MoneyDeclaration, error) {
	if len(conds) == 0 {
		return MoneyDeclaration{}, errors.New("field: sign_when must not be empty")
	}
	d.signWhen = append([]ConditionalSign(nil), conds...)
	return d, nil
}

// WithMin declares the field's lower bound.
func (d MoneyDeclaration) WithMin(min decimal.Decimal) MoneyDeclaration {
	d.min = min
	d.hasMin = true
	return d
}

// WithMax declares the field's upper bound.
func (d MoneyDeclaration) WithMax(max decimal.Decimal) MoneyDeclaration {
	d.max = max
	d.hasMax = true
	return d
}

// WithExclusiveMin marks a declared Min as excluding the boundary value.
func (d MoneyDeclaration) WithExclusiveMin() MoneyDeclaration {
	d.exclusiveMin = true
	return d
}

// WithExclusiveMax marks a declared Max as excluding the boundary value.
func (d MoneyDeclaration) WithExclusiveMax() MoneyDeclaration {
	d.exclusiveMax = true
	return d
}

// WithNonzero forbids zero under the field's declared Sign or SignWhen.
func (d MoneyDeclaration) WithNonzero() MoneyDeclaration {
	d.nonzero = true
	return d
}

// WithNullSemantics declares the field's null-vs-absent handling (MU-08).
// n must be NullSemanticsDistinct.
func (d MoneyDeclaration) WithNullSemantics(n NullSemantics) (MoneyDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return MoneyDeclaration{}, err
	}
	d.common = c
	return d, nil
}
