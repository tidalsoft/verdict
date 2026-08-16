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

// Parse parses s as decimal text (SPEC-MU §2.6.1): an optional leading
// sign, a significand, and an optional exponent -- "49.99", "-10.5", "0",
// and "5e-3" are all valid. Malformed text produces a clearly wrapped
// error; non-finite forms ("NaN", "Infinity", "-Infinity") are rejected
// too, since neither is a valid monetary or quantity value.
//
// # Exponents are part of the wire format, not an extension to it
//
// An earlier version of this function rejected scientific/exponential
// notation outright. SPEC-MU §2.6.1's decimal-text grammar admits an
// exponent deliberately: "An exponent is admitted, because JSON
// serialisers emit one unbidden and because this document's own default
// for tolerance is written 1e-9." Rejecting it here would make this
// package unable to parse either the values that grammar declares valid or
// its own SPEC-MU §2.4.2 default. Vector 42 (MU-01) pins the case
// directly -- "100e-2" is minor_units input carrying two decimal places,
// exactly as if it had been written "1.00" -- which this function must be
// able to parse at all before MU-01 can ever see it.
//
// The result's decimal-place count still reflects the text as written --
// see DecimalPlaces' doc comment -- and apd's own literal-exponent parsing
// (rather than a normalised/reduced coefficient) is what makes "100e-2"
// report 2 decimal places, "5e-3" report 3, and "1.2e3" report 0, matching
// SPEC-MU §2.6.1's own worked examples exactly with no extra logic in this
// package: apd stores "100e-2" as coefficient 100 at exponent -2, not as a
// reduced coefficient 1 at exponent 0.
//
// String() always renders the parsed result back out in plain-decimal
// form regardless of whether s itself used exponential notation -- see its
// own doc comment -- so accepting an exponent on the way in never lets one
// leak back out through this package's own output.
//
// Decimal places are preserved exactly as written — Parse does not
// normalise trailing zeros or reduce the coefficient — because MU-01/MU-14
// need to know how many decimal places the caller actually supplied, not a
// mathematically-reduced count.
func Parse(s string) (Decimal, error) {
	// apd accepts a trailing point ("5." parses as 5); SPEC-MU §2.6.1's
	// grammar does not. Its significand is `1*DIGIT [ "." 1*DIGIT ]` or
	// `"." 1*DIGIT` -- at least one digit after the point, always -- and
	// the prose spells the same rule out: "no trailing point". A string
	// this function accepts is decimal text to every rule that consumes
	// it, so a leniency here is not confined to one check: MU-09 reported
	// PASS for "5." on the strength of it parsing, when "5." matches none
	// of MU-09's enumerated shapes and owes the caller INDETERMINATE.
	// Rejecting it at the one place that defines what decimal text means
	// keeps every consumer agreeing with the specification rather than
	// with apd. A leading point is not affected: ".5" is decimal text
	// under the second alternative and still parses.
	//
	// The test is "a point must be followed by a digit", not "the string
	// must not end in a point". Those differ whenever an exponent follows:
	// "5.e3" ends in "3", but its significand is still "5." and is still
	// not decimal text. An end-of-string test lets that through, and it
	// reaches MU-09 as a PASS exactly as "5." did.
	if i := strings.IndexByte(s, '.'); i >= 0 {
		if i+1 >= len(s) || s[i+1] < '0' || s[i+1] > '9' {
			return Decimal{}, fmt.Errorf("decimal: parse %q: a decimal point must be followed by a digit", s)
		}
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

// Mul returns the exact product d * other. Multiplying two finite decimals
// is always exact -- the result's coefficient is the product of the two
// operands' coefficients and its exponent their sum, with no rounding
// involved at any precision -- so, exactly as with Add and Sub, the only
// failure mode is the product's exponent falling outside the range this
// package's underlying arithmetic supports.
//
// MU-04/MU-05/MU-07/MU-15 (quantity unit conversion, SPEC-MU §4) need
// multiplication to apply a unit's scale factor -- kg = lb * 0.45359237,
// K = °F * (a decimal approximation of 5/9) + offset -- entirely in exact
// decimal arithmetic, never float64. A conversion factor that is not
// itself exactly representable as a decimal (Fahrenheit's 5/9) is supplied
// by the caller pre-truncated to a fixed number of digits; Mul itself
// performs no rounding and cannot introduce imprecision of its own -- see
// tables.NewUnitRegistry's doc comment for where that truncation happens
// and why.
func (d Decimal) Mul(other Decimal) (Decimal, error) {
	var product apd.Decimal
	if _, err := exactContext().Mul(&product, &d.v, &other.v); err != nil {
		return Decimal{}, fmt.Errorf("decimal: multiply (%s) * (%s): %w", operandSummary(d), operandSummary(other), err)
	}
	return Decimal{v: product}, nil
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
