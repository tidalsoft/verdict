package mu

import (
	"sort"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
)

// This file holds the decimal-exact statistics MU-20 (outlier.go) and
// MU-21 (scaleshift.go) both need -- median, quartiles, and median
// absolute deviation -- plus the Class S Result constructors all three
// SPEC-MU §6 checks (MU-20, MU-21, MU-22) build their outcomes through.
// None of it performs I/O or reads a clock; every function here is a pure
// function of the decimal.Decimal values it is given, matching this
// package's purity contract (mu.go's Input doc comment, SPEC-SYS §14.1).

// statResult builds a Class S result at warn severity -- SPEC-MU §6's
// shared header for every check in this file's family: "All checks in
// this section default to warn and are subject to the promotion
// constraint in section 2.2." Promotion to block severity for a specific
// field happens outside this package, after a measured per-field
// precision review this library has no visibility into (see
// verdict.NewPromotedResult's own doc comment) -- so every Result this
// package's statistical checks build uses this constructor, never
// NewPromotedResult, and always carries SeverityWarn.
func statResult(checkID string, outcome verdict.Outcome) (verdict.Result, error) {
	return verdict.NewResult(checkID, verdict.ClassS, verdict.SeverityWarn, outcome)
}

// statIndeterminateResult builds the Class S, warn-severity INDETERMINATE
// result a statistical check returns when it is applicable but a required
// input -- the 200-observation floor, a degenerate distribution, an
// uncoercible value -- was not met. See warnResult's own doc comment (in
// mu.go) for why this and its two siblings below always report their bool
// return as true.
func statIndeterminateResult(checkID string) (verdict.Result, bool, error) {
	res, err := statResult(checkID, verdict.OutcomeIndeterminate)
	return res, true, err
}

// statFailResult builds the Class S, warn-severity FAIL result a
// statistical check returns when it finds a violation.
func statFailResult(checkID string) (verdict.Result, bool, error) {
	res, err := statResult(checkID, verdict.OutcomeFail)
	return res, true, err
}

// statPassResult builds the Class S, warn-severity PASS result a
// statistical check returns when it finds no violation.
func statPassResult(checkID string) (verdict.Result, bool, error) {
	res, err := statResult(checkID, verdict.OutcomePass)
	return res, true, err
}

// observationValues extracts the Value of every entry in observations,
// discarding Entity/HasEntity -- the view MU-20 and MU-21 both need of
// Input.Observations, which read the window's magnitudes only and never
// filter by entity (SPEC-MU §6: MU-20's and MU-21's windows are "per
// customer per field," not per entity; entity-scoping is MU-22's own
// addition, applied in transposition.go instead).
func observationValues(observations []Observation) []decimal.Decimal {
	values := make([]decimal.Decimal, len(observations))
	for i, obs := range observations {
		values[i] = obs.Value
	}
	return values
}

// sortedCopy returns a newly allocated, ascending-sorted copy of values.
// It never sorts values itself: Input.Observations is caller-owned data
// this package only reads (mu.go's Input doc comment), and sorting it in
// place would be an undocumented side effect on a slice the caller may
// reuse for the next field or the next call entirely.
func sortedCopy(values []decimal.Decimal) []decimal.Decimal {
	out := make([]decimal.Decimal, len(values))
	copy(out, values)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Compare(out[j]) < 0
	})
	return out
}

// half returns the exact decimal constant 0.5 -- median's own need for a
// fixed literal when averaging two middle order statistics, mirroring
// one() (mu.go) exactly: a zero-argument function rather than a
// parameterized helper, so no future call site can thread new,
// unvetted text through it.
func half() decimal.Decimal {
	return mustParseDecimal("0.5")
}

// median returns the median of sorted, which must already be
// ascending-sorted (sortedCopy) and non-empty -- every call site in this
// package enforces the 200-observation floor, or derives a half-window
// from one that already has, before calling this, so an empty sorted is
// never reachable and this function does not guard against it.
//
// Reading taken for an unspecified case. SPEC-MU §6 says "compute the
// median" and never states how to average an even-count window -- no
// vector's window has an even count either, so nothing forces a
// particular reading. This function takes the ordinary statistical
// definition, the arithmetic mean of the two middle order statistics for
// an even count and the single middle order statistic for an odd count,
// because it is the standard reading a customer or a future implementer
// would expect by default, and no reading in the other direction is
// suggested anywhere in the text. The average of two decimals is always
// exactly representable (halving a terminating decimal can add at most
// one further decimal place), so this needs no division this package's
// decimal type does not provide -- see half()'s own doc comment.
func median(sorted []decimal.Decimal) (decimal.Decimal, error) {
	n := len(sorted)
	mid := n / 2
	if n%2 == 1 {
		return sorted[mid], nil
	}
	sum, err := sorted[mid-1].Add(sorted[mid])
	if err != nil {
		return decimal.Decimal{}, err
	}
	return sum.Mul(half())
}

// quartiles returns the first and third quartiles of sorted, which must
// already be ascending-sorted and carry at least two elements -- MU-21
// (scaleshift.go) is this function's only caller, and it never reaches
// here below the 200-observation floor.
//
// Reading taken for an unspecified case, continuing median's own: SPEC-MU
// §6 requires "outside the interquartile range" for MU-21 but does not
// define a quartile method, and again no vector's window forces one. This
// function takes the exclusive-median (Tukey's hinges) method: split
// sorted at its midpoint into a lower and an upper half -- excluding the
// overall median element itself when the count is odd, since that
// element belongs to neither half -- and take each half's own median
// (defined exactly as above) as Q1 and Q3. This is the same standard
// reading median() takes for its own unspecified case, applied
// consistently rather than picking a different convention per function,
// and it is a common, teachable definition a reader encountering "the
// interquartile range" with no further qualification would expect.
//
// sorted[n-mid:] computes the upper half for both parities without a
// branch: for odd n, n-mid == mid+1, which skips the middle element
// exactly as the exclusive method requires; for even n, n-mid == mid,
// which splits the window evenly with nothing to exclude.
//
// Both q1 and q3 are computed via the checked median(), not a must-
// variant: q1's own Add can overflow independently of whatever the full
// window's own median, or q3, already proved representable. SPEC-MU
// imposes no same-signed requirement on MU-21's window, and a money field
// legitimately carries refunds alongside charges, so a window mixing
// large-magnitude positive and negative observations concentrates its
// extremes in whichever half sorts them together -- most often the lower
// half, since ascending sort places every negative observation there.
// checkMU21 turns either error into INDETERMINATE for MU-21 alone (see
// its own call site), never an aborting error.
func quartiles(sorted []decimal.Decimal) (decimal.Decimal, decimal.Decimal, error) {
	n := len(sorted)
	mid := n / 2
	lower := sorted[:mid]
	upper := sorted[n-mid:]

	q1, err := median(lower)
	if err != nil {
		return decimal.Decimal{}, decimal.Decimal{}, err
	}
	q3, err := median(upper)
	if err != nil {
		return decimal.Decimal{}, decimal.Decimal{}, err
	}
	return q1, q3, nil
}

// medianAbsoluteDeviation returns the median of the absolute deviation of
// every entry in values from med -- SPEC-MU §6 MU-20's own definition:
// "compute the median and median absolute deviation." values need not be
// sorted; the deviations computed from it are sorted internally
// (sortedCopy) before median() is applied to them, a second, independent
// application of median()'s own even/odd reading above.
func medianAbsoluteDeviation(values []decimal.Decimal, med decimal.Decimal) (decimal.Decimal, error) {
	deviations := make([]decimal.Decimal, len(values))
	for i, v := range values {
		diff, err := v.Sub(med)
		if err != nil {
			return decimal.Decimal{}, err
		}
		deviations[i] = diff.Abs()
	}
	return median(sortedCopy(deviations))
}
