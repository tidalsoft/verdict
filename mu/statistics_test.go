package mu

import (
	"testing"

	"github.com/tidalsoft/verdict/decimal"
)

// This file tests the decimal-exact statistics in statistics.go directly,
// against hand-computed expected values -- not through checkMU20/checkMU21
// outcome assertions, which exercise every statement here but assert
// nothing about the numbers these functions actually compute. A check
// test asserting PASS/FAIL/INDETERMINATE would still pass if median()
// silently returned the wrong quartile or the wrong deviation, so long as
// the final threshold comparison happened to land on the same side; these
// tests pin the arithmetic itself.

func wantDecimal(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()
	if got.Compare(mustParse(t, want)) != 0 {
		t.Errorf("got %s, want %s", got.String(), want)
	}
}

func decimalsOf(t *testing.T, values ...string) []decimal.Decimal {
	t.Helper()
	out := make([]decimal.Decimal, len(values))
	for i, v := range values {
		out[i] = mustParse(t, v)
	}
	return out
}

func TestMedian_OddCount(t *testing.T) {
	// [3, 1, 2] sorted is [1, 2, 3]; the odd-count branch picks the single
	// middle order statistic directly, no Add involved.
	sorted := sortedCopy(decimalsOf(t, "3", "1", "2"))
	got, err := median(sorted)
	if err != nil {
		t.Fatalf("median: unexpected error: %v", err)
	}
	wantDecimal(t, got, "2")
}

func TestMedian_EvenCount(t *testing.T) {
	// [1, 2, 3, 4]: the even-count branch averages the two middle order
	// statistics, (2+3)/2 = 2.5.
	sorted := sortedCopy(decimalsOf(t, "4", "2", "1", "3"))
	got, err := median(sorted)
	if err != nil {
		t.Fatalf("median: unexpected error: %v", err)
	}
	wantDecimal(t, got, "2.5")
}

func TestMedian_EvenCount_MixedSign(t *testing.T) {
	// [-2, -1, 0, 1]: the two middle order statistics are -1 and 0,
	// averaging to -0.5 -- pins that averaging works correctly across a
	// sign change, not just within one sign.
	sorted := sortedCopy(decimalsOf(t, "1", "-1", "-2", "0"))
	got, err := median(sorted)
	if err != nil {
		t.Fatalf("median: unexpected error: %v", err)
	}
	wantDecimal(t, got, "-0.5")
}

func TestQuartiles_EvenCount(t *testing.T) {
	// [1..8]: mid=4, lower=[1,2,3,4] (Q1 = (2+3)/2 = 2.5), upper=[5,6,7,8]
	// (Q3 = (6+7)/2 = 6.5).
	sorted := sortedCopy(decimalsOf(t, "1", "2", "3", "4", "5", "6", "7", "8"))
	q1, q3, err := quartiles(sorted)
	if err != nil {
		t.Fatalf("quartiles: unexpected error: %v", err)
	}
	wantDecimal(t, q1, "2.5")
	wantDecimal(t, q3, "6.5")
}

// TestQuartiles_OddCount_Heterogeneous is the branch a fresh reviewer
// found no test reaching: an odd-count window of genuinely different
// values (not all-identical, not overflow-sized) whose Q1 and Q3 are a
// real interquartile decision, not a degenerate single point. [1..7]:
// mid=3, so the overall median (element 4, excluded from both halves by
// the exclusive-median method) is 4; lower=[1,2,3] picks Q1=2 directly;
// upper=sorted[7-3:]=[5,6,7] picks Q3=6 directly.
func TestQuartiles_OddCount_Heterogeneous(t *testing.T) {
	sorted := sortedCopy(decimalsOf(t, "7", "3", "1", "5", "2", "6", "4"))
	q1, q3, err := quartiles(sorted)
	if err != nil {
		t.Fatalf("quartiles: unexpected error: %v", err)
	}
	wantDecimal(t, q1, "2")
	wantDecimal(t, q3, "6")
}

func TestQuartiles_OddCount_MixedSign(t *testing.T) {
	// [-5, -3, -1, 0, 2, 4, 6]: mid=3, lower=[-5,-3,-1] picks Q1=-3
	// directly, upper=sorted[7-3:]=[2,4,6] picks Q3=4 directly -- a
	// window whose extremes sit on opposite sides of zero.
	sorted := sortedCopy(decimalsOf(t, "6", "-1", "4", "-5", "0", "2", "-3"))
	q1, q3, err := quartiles(sorted)
	if err != nil {
		t.Fatalf("quartiles: unexpected error: %v", err)
	}
	wantDecimal(t, q1, "-3")
	wantDecimal(t, q3, "4")
}

func TestMedianAbsoluteDeviation_OddCount(t *testing.T) {
	// values [1,2,3,4,5] against med=3: deviations are [2,1,0,1,2],
	// sorted [0,1,1,2,2], odd-count median picks the middle one, 1.
	values := decimalsOf(t, "1", "2", "3", "4", "5")
	got, err := medianAbsoluteDeviation(values, mustParse(t, "3"))
	if err != nil {
		t.Fatalf("medianAbsoluteDeviation: unexpected error: %v", err)
	}
	wantDecimal(t, got, "1")
}

func TestMedianAbsoluteDeviation_EvenCount(t *testing.T) {
	// values [1,2,3,4] against med=2.5: deviations are [1.5,0.5,0.5,1.5],
	// sorted [0.5,0.5,1.5,1.5], even-count median averages the two
	// middle entries: (0.5+1.5)/2 = 1.
	values := decimalsOf(t, "1", "2", "3", "4")
	got, err := medianAbsoluteDeviation(values, mustParse(t, "2.5"))
	if err != nil {
		t.Fatalf("medianAbsoluteDeviation: unexpected error: %v", err)
	}
	wantDecimal(t, got, "1")
}

func TestMedianAbsoluteDeviation_MixedSign(t *testing.T) {
	// values [-3,-1,1,3] against med=0: deviations are [3,1,1,3], sorted
	// [1,1,3,3], even-count median averages the two middle entries:
	// (1+3)/2 = 2.
	values := decimalsOf(t, "-3", "-1", "1", "3")
	got, err := medianAbsoluteDeviation(values, mustParse(t, "0"))
	if err != nil {
		t.Fatalf("medianAbsoluteDeviation: unexpected error: %v", err)
	}
	wantDecimal(t, got, "2")
}

func TestHalf(t *testing.T) {
	wantDecimal(t, half(), "0.5")
}

func TestSortedCopy(t *testing.T) {
	original := decimalsOf(t, "3", "1", "2")
	got := sortedCopy(original)

	wantOrder := []string{"1", "2", "3"}
	if len(got) != len(wantOrder) {
		t.Fatalf("sortedCopy returned %d elements, want %d", len(got), len(wantOrder))
	}
	for i, want := range wantOrder {
		wantDecimal(t, got[i], want)
	}

	// original must be untouched -- sortedCopy must not sort in place.
	wantDecimal(t, original[0], "3")
	wantDecimal(t, original[1], "1")
	wantDecimal(t, original[2], "2")
}

func TestObservationValues(t *testing.T) {
	observations := []Observation{
		{Value: mustParse(t, "1"), Entity: "cust-1", HasEntity: true},
		{Value: mustParse(t, "2")},
	}
	got := observationValues(observations)
	if len(got) != 2 {
		t.Fatalf("observationValues returned %d elements, want 2", len(got))
	}
	wantDecimal(t, got[0], "1")
	wantDecimal(t, got[1], "2")
}
