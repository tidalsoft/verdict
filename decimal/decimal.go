package decimal

import (
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// exactContext builds a fresh, unbounded-precision context for a single
// operation. Precision 0 disables digit-count rounding entirely (see
// apd.BaseContext's doc comment: "Disable rounding"), which is what "exact"
// means here — reconciliation arithmetic (MU-12) requires no rounding at
// all. A new context is built on every call rather than shared
// as package state, per this package's "no package-level state" rule (see
// doc.go).
func exactContext() *apd.Context {
	ctx := apd.BaseContext
	return &ctx
}

// Decimal is an exact, arbitrary-precision decimal value. Its zero value is
// the exact decimal 0, a valid value in its own right — every Decimal that
// exists, whether zero-valued or produced by Parse or an arithmetic method,
// is valid by construction, and there is no separate invalid state to guard
// against.
//
// Decimal values must be compared with Compare or IsZero, never with Go's
// == or !=: two Decimals holding the same numeric value (e.g. one parsed
// from "1" and one from "1.0") are not required to share an internal
// representation.
type Decimal struct {
	v apd.Decimal
}

// Parse parses s as an exact decimal number. s must be the plain-text
// decimal-string form used for monetary and quantity values on the wire
// (e.g. "49.99", "-10.5", "0"). Malformed text produces a clearly wrapped
// error; non-finite forms ("NaN", "Infinity", "-Infinity") are rejected
// too, since neither is a valid monetary or quantity value.
//
// Scientific/exponential notation ("1E3", "1.5e10", "-2E-5") is rejected
// even though the underlying library accepts it: the wire format here is
// plain decimal, a customer's tool-call argument is never going to arrive
// as "1E3" for a dollar amount, and accepting it here would let this
// package's own String() echo a value back out in a form the wire format
// does not permit (see String's doc comment).
//
// Decimal places are preserved exactly as written — Parse does not
// normalise trailing zeros or reduce the coefficient — because MU-14
// (minor_unit_exponent) needs to know how many decimal places the caller
// actually supplied, not a mathematically-reduced count.
func Parse(s string) (Decimal, error) {
	if strings.ContainsAny(s, "eE") {
		return Decimal{}, fmt.Errorf("decimal: parse %q: scientific notation is not a valid decimal string", s)
	}
	d, _, err := apd.NewFromString(s)
	if err != nil {
		return Decimal{}, fmt.Errorf("decimal: parse %q: %w", s, err)
	}
	if d.Form != apd.Finite {
		return Decimal{}, fmt.Errorf("decimal: parse %q: not a finite decimal value", s)
	}
	return Decimal{v: *d}, nil
}

// String returns d's plain decimal-string representation (never scientific
// notation), preserving the sign and the exponent (decimal-place count) it
// was constructed or computed with. Forcing plain-decimal output — rather
// than apd's default, which switches to scientific notation for large
// exponents — matters because a Decimal produced inside this package (by
// Add, Sub, or ScaleByExponent) must remain representable in the same
// plain-decimal wire format Parse requires on the way in; nothing computed
// by this package should be able to hand a caller a value it could not
// itself have parsed.
func (d Decimal) String() string {
	return d.v.Text('f')
}

// Sign returns -1 if d is negative, 0 if d is zero (or negative zero), and
// +1 if d is positive.
func (d Decimal) Sign() int {
	return d.v.Sign()
}

// IsZero reports whether d is exactly zero.
func (d Decimal) IsZero() bool {
	return d.v.IsZero()
}

// Compare returns -1 if d < other, 0 if d == other, and +1 if d > other.
// The comparison is exact: no float64 conversion occurs anywhere in it.
// Money never passes through float64; MU-07 (range_bound) requires that
// kind: money comparisons happen "never in floating point."
func (d Decimal) Compare(other Decimal) int {
	return d.v.Cmp(&other.v)
}

// operandSummary returns a cheap, bounded description of d for use in error
// messages: its exponent, coefficient digit count, and sign, never its full
// decimal text. A value large enough to make Add or Sub overflow can have a
// coefficient up to apd's ~10^5-digit ceiling; String() on such a value is a
// bounded but non-trivial cost, and an error path has no business paying it
// just to identify which operand failed.
func operandSummary(d Decimal) string {
	return fmt.Sprintf("exponent=%d digits=%d sign=%d", d.v.Exponent, d.v.NumDigits(), d.v.Sign())
}

// Add returns the exact sum d + other. Addition of two finite decimals
// never requires digit rounding, so the only way this can fail is if the
// result's exponent falls outside the range this package's underlying
// arithmetic supports (±10^5) — far beyond any monetary or quantity value
// this engine handles, but reported as an error rather than silently
// clamped or panicked, per this codebase's "fail fast, fail loudly"
// principle.
func (d Decimal) Add(other Decimal) (Decimal, error) {
	var sum apd.Decimal
	if _, err := exactContext().Add(&sum, &d.v, &other.v); err != nil {
		return Decimal{}, fmt.Errorf("decimal: add (%s) + (%s): %w", operandSummary(d), operandSummary(other), err)
	}
	return Decimal{v: sum}, nil
}

// Sub returns the exact difference d - other. See Add for the sole failure
// condition.
func (d Decimal) Sub(other Decimal) (Decimal, error) {
	var diff apd.Decimal
	if _, err := exactContext().Sub(&diff, &d.v, &other.v); err != nil {
		return Decimal{}, fmt.Errorf("decimal: subtract (%s) - (%s): %w", operandSummary(d), operandSummary(other), err)
	}
	return Decimal{v: diff}, nil
}

// Abs returns the absolute value of d. This only ever flips a sign flag —
// it never touches the coefficient or exponent — so, unlike Add and Sub, it
// cannot fail.
func (d Decimal) Abs() Decimal {
	abs := d.v
	abs.Negative = false
	return Decimal{v: abs}
}

// DecimalPlaces returns the number of digits after the decimal point in d's
// stored representation — e.g. 3 for a value parsed from "49.999", 2 for
// "49.90", 0 for "500". This reflects exactly how the value was supplied or
// computed, not a reduced or canonicalised count, which is what MU-14
// (reject amounts carrying more decimal places than a currency's
// minor-unit exponent permits) requires.
func (d Decimal) DecimalPlaces() int32 {
	if d.v.Exponent >= 0 {
		return 0
	}
	return -d.v.Exponent
}

// ScaleByExponent returns d * 10^exponent. Multiplying a decimal by a power
// of ten only shifts where the decimal point falls, and never rounds a
// digit away, but the shifted exponent can still put the result outside the
// range this package's underlying arithmetic supports, or, before any such
// bound is even consulted, overflow int32 outright.
//
// apd enforces two independent bounds, both checked here: the raw exponent
// itself must fall within apd's ±10^5 system limits, and separately the
// adjusted exponent — raw exponent + (coefficient digit count) − 1, i.e. the
// power-of-ten position of the leading digit, which is what actually
// determines a decimal's magnitude — must too. Checking only the raw value
// is not enough: a multi-digit coefficient can have a raw exponent that
// looks perfectly safe while its adjusted exponent is already over the
// limit (e.g. a 100000-digit coefficient at exponent 0 has adjusted
// exponent 99999 already; scaling by a further +2 moves the raw exponent to
// a harmless-looking 2, but pushes the adjusted exponent to 100001). Both
// bounds are checked with 64-bit arithmetic before any shift is applied,
// and a violation is reported as an error rather than silently wrapping or
// handing back a Decimal that apd itself would then refuse to Add, Sub, or
// round-trip through Parse(String()) — which is exactly the class of
// silent wrong-value defect this package exists to keep out of the engine.
//
// MU-07 requires that range comparison for kind: money happen "in
// minor units after normalisation, never in floating point." ScaleByExponent
// is that normalisation step: callers scale a major-units amount by the
// currency's ISO 4217 minor-unit exponent (e.g. 2 for USD) and compare the
// result with Compare, entirely in exact decimal arithmetic.
//
// The error message deliberately reports exponents and a digit count (cheap
// integers), never d's or the result's full decimal-string form: the whole
// reason this method can fail is that some exponent involved is huge, or
// heading there, so formatting a full plain-decimal String() of it would
// itself be an expensive, unbounded-size operation. An error path must stay
// cheap regardless of how the caller got d into whatever state it's in.
func (d Decimal) ScaleByExponent(exponent int32) (Decimal, error) {
	shifted := int64(d.v.Exponent) + int64(exponent)
	numDigits := d.v.NumDigits()
	adjusted := shifted + numDigits - 1

	if shifted < int64(apd.MinExponent) || shifted > int64(apd.MaxExponent) ||
		adjusted < int64(apd.MinExponent) || adjusted > int64(apd.MaxExponent) {
		return Decimal{}, fmt.Errorf(
			"decimal: scale by 10^%d: resulting exponent %d (digits=%d, adjusted exponent %d) exceeds supported range ±%d",
			exponent, shifted, numDigits, adjusted, apd.MaxExponent,
		)
	}
	scaled := d.v
	scaled.Exponent = int32(shifted)
	return Decimal{v: scaled}, nil
}
