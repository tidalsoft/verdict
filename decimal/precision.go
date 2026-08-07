package decimal

import (
	"math/big"

	"github.com/cockroachdb/apd/v3"
)

// Provenance records how a numeric value entered the engine. MU-02
// (precision_loss) treats the same numeric value differently depending on
// this: a value transported as a decimal string is exact by construction;
// the identical value decoded from a JSON number is subject to binary64
// fidelity checks. See the package doc comment for why this package cannot
// infer provenance itself.
type Provenance int

const (
	// FromString marks a value that arrived as a decimal string. MU-02
	// never fails a FromString value.
	FromString Provenance = iota
	// FromJSONNumber marks a value decoded from a JSON number token.
	// MU-02 fails a FromJSONNumber value that is not exactly
	// representable in IEEE 754 binary64, or whose magnitude exceeds
	// 2^53-1.
	FromJSONNumber
)

// safeIntegerMagnitude is 2^53-1 (9007199254740991), the largest integer
// magnitude an IEEE 754 binary64 float can represent without
// representational gaps. It is built fresh inside the one
// function that needs it rather than held as package state, per this
// package's "no package-level state" rule (see doc.go); apd.New takes an
// int64 coefficient directly and cannot fail, so there is no error path to
// consider here.
func safeIntegerMagnitude() Decimal {
	return Decimal{v: *apd.New(9007199254740991, 0)}
}

// rat converts d to an exact big.Rat. The conversion itself never loses
// precision: a decimal's value is coeff * 10^exponent, which big.Rat
// represents exactly as a fraction of arbitrary-precision integers.
func (d Decimal) rat() *big.Rat {
	coeff := new(big.Int).Set(d.v.Coeff.MathBigInt())
	if d.v.Negative {
		coeff.Neg(coeff)
	}
	r := new(big.Rat).SetInt(coeff)
	switch {
	case d.v.Exponent > 0:
		pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d.v.Exponent)), nil)
		r.Mul(r, new(big.Rat).SetInt(pow))
	case d.v.Exponent < 0:
		pow := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-d.v.Exponent)), nil)
		r.Quo(r, new(big.Rat).SetInt(pow))
	}
	return r
}

// ExactlyRepresentableInBinary64 reports whether d's exact value can be
// represented exactly as an IEEE 754 binary64 float — i.e. converting d to
// the nearest float64 and back loses no precision. MU-02 fails a
// JSON-number value for which this is false: the JSON number 0.1 is not
// exactly representable and fails; the identical value supplied as the
// string "0.1" is exact by construction and would pass.
func (d Decimal) ExactlyRepresentableInBinary64() bool {
	_, exact := d.rat().Float64()
	return exact
}

// ExceedsSafeIntegerMagnitude reports whether abs(d) > 2^53-1, the largest
// integer magnitude a binary64 float can carry without representational
// gaps. MU-02 fails a JSON-number value for which this is true. This is
// independent of ExactlyRepresentableInBinary64: 2^53 itself is exactly
// representable in binary64 yet still exceeds this ceiling.
func (d Decimal) ExceedsSafeIntegerMagnitude() bool {
	return d.Abs().Compare(safeIntegerMagnitude()) > 0
}

// PrecisionLoss reports whether d fails MU-02's precision_loss check, given
// how d arrived (provenance). A FromString value never fails: MU-02 only
// examines values that arrived as JSON numbers, since a decimal string is
// exact by construction. A FromJSONNumber value fails if it is not exactly
// representable in IEEE 754 binary64, or if its magnitude exceeds 2^53-1.
//
// This function reports the underlying condition only; turning it into a
// verdict (PASS/FAIL/INDETERMINATE, severity, message) is MU-02's own
// responsibility, not this package's.
func PrecisionLoss(d Decimal, provenance Provenance) bool {
	if provenance == FromString {
		return false
	}
	return !d.ExactlyRepresentableInBinary64() || d.ExceedsSafeIntegerMagnitude()
}
