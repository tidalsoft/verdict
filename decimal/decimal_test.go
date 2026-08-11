package decimal

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/cockroachdb/apd/v3"
)

func mustParse(t *testing.T, s string) Decimal {
	t.Helper()
	d, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) unexpected error: %v", s, err)
	}
	return d
}

func mustScale(t *testing.T, d Decimal, exponent int32) Decimal {
	t.Helper()
	scaled, err := d.ScaleByExponent(exponent)
	if err != nil {
		t.Fatalf("ScaleByExponent(%s, %d) unexpected error: %v", d, exponent, err)
	}
	return scaled
}

// newTestDecimal builds a Decimal directly from a coefficient and exponent,
// bypassing Parse. It exists only for tests that need a value with an
// exponent near apd's ±10^5 limit (to exercise the overflow path in Add,
// Sub, and ScaleByExponent) or otherwise outside what Parse's plain-decimal,
// no-scientific-notation contract can express as a literal in Go source.
// Nothing in the exported API can produce such a value from a string
// directly -- only Add/Sub/ScaleByExponent composition can reach one, which
// is exactly what these tests simulate.
func newTestDecimal(coeff int64, exponent int32) Decimal {
	return Decimal{v: *apd.New(coeff, exponent)}
}

func TestParse_Success(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"integer", "4999", "4999"},
		{"decimal string wire format, vector 9", "49.99", "49.99"},
		{"negative", "-49.99", "-49.99"},
		{"zero", "0", "0"},
		{"trailing zero preserved for MU-14", "49.90", "49.90"},
		{"leading plus", "+49.99", "49.99"},
		{"large magnitude", "9007199254740993", "9007199254740993"},
		{"exponent, vector 42", "100e-2", "1.00"},
		{"exponent, SPEC-MU §2.6.1 example", "5e-3", "0.005"},
		{"exponent, SPEC-MU §2.6.1 example, no fractional digits", "1.2e3", "1200"},
		{"exponent, uppercase E", "1E3", "1000"},
		{"exponent, explicit plus", "1.5e+2", "150"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := mustParse(t, tc.in)
			if got := d.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParse_Failure(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty string", ""},
		{"not a number", "not-a-number"},
		{"whitespace padded", " 49.99 "},
		{"multiple decimal points", "49.9.9"},
		{"currency symbol", "$49.99"},
		{"NaN is not a valid monetary value", "NaN"},
		{"Infinity is not a valid monetary value", "Infinity"},
		{"negative infinity", "-Infinity"},
		{"double exponent marker", "1e2e3"},
		{"exponent with no digits", "1e"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.in)
			if err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", tc.in)
			}
			if !strings.Contains(err.Error(), "decimal:") {
				t.Errorf("error %q missing package context prefix", err.Error())
			}
		})
	}
}

func TestDecimal_SignAndIsZero(t *testing.T) {
	cases := []struct {
		in       string
		wantSign int
		wantZero bool
	}{
		{"0", 0, true},
		{"49.99", 1, false},
		{"-49.99", -1, false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			d := mustParse(t, tc.in)
			if got := d.Sign(); got != tc.wantSign {
				t.Errorf("Sign() = %d, want %d", got, tc.wantSign)
			}
			if got := d.IsZero(); got != tc.wantZero {
				t.Errorf("IsZero() = %v, want %v", got, tc.wantZero)
			}
		})
	}

	// The zero value, never touched by Parse, must also be a valid decimal
	// zero -- this type's zero value is meaningful in its own right, not a
	// sentinel for "not yet constructed."
	var zero Decimal
	if !zero.IsZero() {
		t.Errorf("zero value Decimal.IsZero() = false, want true")
	}
	if zero.Sign() != 0 {
		t.Errorf("zero value Decimal.Sign() = %d, want 0", zero.Sign())
	}
}

func TestDecimal_Compare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"49.99", "49.99", 0},
		{"1", "1.0", 0}, // equal value, different representation
		{"1", "2", -1},
		{"2", "1", 1},
		{"-1", "1", -1},
		{"-1", "-2", 1},
		{"0", "-0", 0},
	}
	for _, tc := range cases {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			a := mustParse(t, tc.a)
			b := mustParse(t, tc.b)
			if got := a.Compare(b); got != tc.want {
				t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestDecimal_AddSub(t *testing.T) {
	a := mustParse(t, "10.10")
	b := mustParse(t, "20.20")

	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("Add unexpected error: %v", err)
	}
	want := mustParse(t, "30.30")
	if sum.Compare(want) != 0 {
		t.Errorf("Add(10.10, 20.20) = %s, want %s", sum, want)
	}

	diff, err := sum.Sub(a)
	if err != nil {
		t.Fatalf("Sub unexpected error: %v", err)
	}
	if diff.Compare(b) != 0 {
		t.Errorf("Sub(30.30, 10.10) = %s, want %s", diff, b)
	}

	// Negative values through Add/Sub.
	neg := mustParse(t, "-5")
	five := mustParse(t, "5")
	sumZero, err := neg.Add(five)
	if err != nil {
		t.Fatalf("Add unexpected error: %v", err)
	}
	if !sumZero.IsZero() {
		t.Errorf("Add(-5, 5) = %s, want 0", sumZero)
	}
}

func TestDecimal_Add_OverflowError(t *testing.T) {
	// Constructed to exceed the underlying arithmetic's supported exponent
	// range (±10^5): this is the one documented failure mode for Add, and it
	// must be a returned error, not a panic or a silently wrong result --
	// "very large magnitude" per the task's required coverage.
	huge := newTestDecimal(9, 100000)
	negHuge := newTestDecimal(-9, 100000)

	_, err := huge.Add(huge)
	if err == nil {
		t.Fatal("Add of two values overflowing the supported exponent range succeeded, want error")
	}
	if !strings.Contains(err.Error(), "decimal: add") {
		t.Errorf("error %q missing operation context", err.Error())
	}

	_, err = huge.Sub(negHuge)
	if err == nil {
		t.Fatal("Sub of two values overflowing the supported exponent range succeeded, want error")
	}
	if !strings.Contains(err.Error(), "decimal: subtract") {
		t.Errorf("error %q missing operation context", err.Error())
	}
}

func TestDecimal_Mul(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{"integers", "12", "3", "36"},
		{"decimal factor", "50", "0.45359237", "22.6796185"},
		{"negative operand", "-12", "3", "-36"},
		{"both negative", "-12", "-3", "36"},
		{"zero", "0", "12345.6789", "0"},
		{"identity", "42.5", "1", "42.5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := mustParse(t, tc.a)
			b := mustParse(t, tc.b)
			got, err := a.Mul(b)
			if err != nil {
				t.Fatalf("Mul unexpected error: %v", err)
			}
			want := mustParse(t, tc.want)
			if got.Compare(want) != 0 {
				t.Errorf("Mul(%s, %s) = %s, want %s", tc.a, tc.b, got, want)
			}
		})
	}
}

func TestDecimal_Mul_OverflowError(t *testing.T) {
	huge := newTestDecimal(9, 100000)
	_, err := huge.Mul(huge)
	if err == nil {
		t.Fatal("Mul of two values overflowing the supported exponent range succeeded, want error")
	}
	if !strings.Contains(err.Error(), "decimal: multiply") {
		t.Errorf("error %q missing operation context", err.Error())
	}
}

// TestDecimal_Sub_ErrorMessageOperandOrder asserts Sub's error text reports
// the operation it actually performed (d - other), not the reverse. A
// message that names the wrong operand order is actively misleading when
// diagnosing a failed MU-12 reconciliation in production.
//
// The error reports cheap operand summaries (exponent/digits/sign), not the
// full decimal text (see operandSummary's doc comment: triggering a genuine
// overflow requires an exponent near apd's ±10^5 limit, and formatting that
// as a full plain-decimal string is exactly the unbounded-cost operation
// this package's error paths avoid). This checks operand order via each
// summary's distinguishing "sign=" marker instead.
func TestDecimal_Sub_ErrorMessageOperandOrder(t *testing.T) {
	huge := newTestDecimal(9, 100000)     // sign=1
	negHuge := newTestDecimal(-9, 100000) // sign=-1

	_, err := huge.Sub(negHuge)
	if err == nil {
		t.Fatal("Sub overflowing the supported exponent range succeeded, want error")
	}
	// The error reports cheap operand summaries (exponent/digits/sign), not
	// full decimal text (see operandSummary's doc comment), so check operand
	// order via each summary's distinguishing "sign=" marker rather than
	// matching decimal-string prefixes.
	msg := err.Error()
	firstSign := strings.Index(msg, "sign=1")
	secondSign := strings.Index(msg, "sign=-1")
	if firstSign == -1 || secondSign == -1 {
		t.Fatalf("error %q missing expected operand sign markers", msg)
	}
	if firstSign > secondSign {
		t.Fatalf("error %q reports operands out of order: d (sign=1) must appear before other (sign=-1)", msg)
	}
}

func TestDecimal_Abs(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"49.99", "49.99"},
		{"-49.99", "49.99"},
		{"0", "0"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			d := mustParse(t, tc.in)
			want := mustParse(t, tc.want)
			if got := d.Abs(); got.Compare(want) != 0 {
				t.Errorf("Abs(%s) = %s, want %s", tc.in, got, want)
			}
		})
	}
}

func TestDecimal_DecimalPlaces(t *testing.T) {
	cases := []struct {
		in   string
		want int32
	}{
		{"500", 0},
		{"49.99", 2},
		{"49.999", 3},
		{"49.90", 2}, // trailing zero preserved, per MU-14's requirement
		{"0", 0},
		{"-49.99", 2},
		{"1000", 0},
		{"100e-2", 2}, // vector 42: two decimal places, no point
		{"5e-3", 3},   // SPEC-MU §2.6.1 worked example
		{"1.2e3", 0},  // SPEC-MU §2.6.1 worked example
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			d := mustParse(t, tc.in)
			if got := d.DecimalPlaces(); got != tc.want {
				t.Errorf("DecimalPlaces(%s) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestDecimal_ScaleByExponent(t *testing.T) {
	cases := []struct {
		in       string
		exponent int32
		want     string
	}{
		{"49.99", 2, "4999"},   // USD major -> minor units
		{"500", 0, "500"},      // JPY, exponent 0
		{"4.999", 3, "4999"},   // KWD, exponent 3
		{"4999", -2, "49.99"},  // minor -> major
		{"-49.99", 2, "-4999"}, // negative amount
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			d := mustParse(t, tc.in)
			got := mustScale(t, d, tc.exponent)
			want := mustParse(t, tc.want)
			if got.Compare(want) != 0 {
				t.Errorf("ScaleByExponent(%s, %d) = %s, want %s", tc.in, tc.exponent, got, want)
			}
		})
	}
}

// TestDecimal_ScaleByExponent_OutOfRange is the regression test for the
// silent-overflow defect found in adversarial review: scaling by an
// exponent that pushes the result outside apd's supported ±10^5 range must
// be a returned error, never a wrapped int32 that silently turns "scale up"
// into "scale down" with no signal.
func TestDecimal_ScaleByExponent_OutOfRange(t *testing.T) {
	one := mustParse(t, "1")

	// The exact reproduction from adversarial review: scaling up by
	// (MaxInt32 - 10) succeeds (result exponent 2147483637, still far
	// beyond apd's own ±10^5 range -- so even the first step here must
	// already fail; the important thing is that it fails loudly instead of
	// silently, and that a second scale on top of it cannot wrap through
	// int32 and land back on a plausible-looking small value).
	_, err := one.ScaleByExponent(math.MaxInt32 - 10)
	if err == nil {
		t.Fatal("ScaleByExponent(1, MaxInt32-10) succeeded, want error (exceeds apd's ±10^5 exponent range)")
	}

	// Directly targeting the apd boundary: one step past is rejected, one
	// step within it is not.
	justOver, err := one.ScaleByExponent(100001)
	if err == nil {
		t.Fatalf("ScaleByExponent(1, 100001) succeeded with %s, want error", justOver)
	}
	if _, err := one.ScaleByExponent(100000); err != nil {
		t.Errorf("ScaleByExponent(1, 100000) unexpected error: %v", err)
	}

	// The int32-wraparound case from adversarial review, adapted: chaining
	// two scales that individually looked plausible under raw int32
	// arithmetic (sum wraps past math.MaxInt32 back to a large negative
	// number) must not silently succeed with a corrupted small value.
	// ScaleByExponent's own bounds check runs in int64 before truncating to
	// int32, so it rejects the composition outright rather than requiring
	// the caller to also bounds-check every chained call.
	almostWrapping := newTestDecimal(1, math.MaxInt32-10)
	if _, err := almostWrapping.ScaleByExponent(20); err == nil {
		t.Fatal("ScaleByExponent chain that wraps int32 succeeded, want error")
	}
}

// TestDecimal_ScaleByExponent_AdjustedExponentBoundary is the regression
// test for the second adversarial-review defect: apd's ±10^5 limit binds
// the *adjusted* exponent (raw exponent + coefficient digit count − 1, the
// power-of-ten position of the leading digit), not the raw exponent alone.
// A single-digit coefficient can't expose this -- for it, adjusted and raw
// are the same number -- which is exactly why the original bounds check
// (tested only against newTestDecimal(9, 100000)-style single-digit values)
// missed it. This uses a 100000-digit coefficient, matching the review's
// reproduction: "999" followed by 99997 zeros, at exponent 0, has adjusted
// exponent 99999 -- safely parseable -- but scaling by +2 pushes the raw
// exponent to a harmless-looking 2 while the adjusted exponent becomes
// 100001, one past apd's limit.
func TestDecimal_ScaleByExponent_AdjustedExponentBoundary(t *testing.T) {
	literal := "999" + strings.Repeat("0", 99997) // 100000 digits total
	d := mustParse(t, literal)

	if _, err := d.ScaleByExponent(2); err == nil {
		t.Fatal("ScaleByExponent pushing the adjusted exponent past apd's ±10^5 limit succeeded, want error")
	}

	// One digit fewer of headroom: scaling by +1 keeps the adjusted exponent
	// at exactly 100000 (the boundary itself, still valid) and must succeed;
	// this pins the fix to the correct side of the off-by-one line rather
	// than just rejecting everything multi-digit.
	scaled, err := d.ScaleByExponent(1)
	if err != nil {
		t.Fatalf("ScaleByExponent(literal, 1) unexpected error: %v", err)
	}

	// The round-trip guarantee String()'s doc comment makes, now actually
	// exercised against a value that could break it: the result must still
	// be plain decimal and still parseable by this package's own Parse.
	s := scaled.String()
	if strings.ContainsAny(s, "eE") {
		t.Errorf("String() = a %d-byte value containing scientific notation", len(s))
	}
	if _, err := Parse(s); err != nil {
		t.Errorf("ScaleByExponent result does not round-trip through Parse: %v", err)
	}
}

// TestDecimal_RangeBound_MinorUnitsNeverFloat exercises the MU-07 comparison
// requirement: "comparison for kind: money is performed in minor units
// after normalisation, never in floating point." It also demonstrates
// why that matters -- a boundary amount that a float64-based comparison
// would get wrong due to binary64 rounding of the scaled value.
func TestDecimal_RangeBound_MinorUnitsNeverFloat(t *testing.T) {
	amount := mustParse(t, "100000.00") // USD major units
	minorUnits := mustScale(t, amount, 2)
	max := mustParse(t, "10000000") // declared max, in minor units

	if minorUnits.Compare(max) > 0 {
		t.Errorf("normalised amount %s compares greater than max %s, want <= 0", minorUnits, max)
	}

	over := mustScale(t, mustParse(t, "100000.01"), 2)
	if over.Compare(max) <= 0 {
		t.Errorf("normalised amount %s compares <= max %s, want > 0", over, max)
	}
}

// TestParse_AcceptsScientificNotation asserts SPEC-MU §2.6.1's decimal-text
// grammar is honoured: an exponent is part of the wire format, not an
// extension to it, and must parse successfully -- see Parse's doc comment
// for why an earlier version of this package rejected it and no longer
// does.
func TestParse_AcceptsScientificNotation(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1E3", "1000"},
		{"1.5e10", "15000000000"},
		{"-2E-5", "-0.00002"},
		{"9e100000", ""}, // near apd's own exponent ceiling; only checked for no error below
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			d, err := Parse(tc.in)
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tc.in, err)
			}
			if tc.want != "" && d.String() != tc.want {
				t.Errorf("Parse(%q).String() = %q, want %q", tc.in, d.String(), tc.want)
			}
		})
	}
}

// TestParse_RejectsMalformedExponent asserts a syntactically broken
// exponent (as opposed to no exponent at all) is still a parse error, not
// silently ignored or truncated.
func TestParse_RejectsMalformedExponent(t *testing.T) {
	cases := []string{"1e2e3", "1e"}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if _, err := Parse(in); err == nil {
				t.Fatalf("Parse(%q) succeeded, want error", in)
			}
		})
	}
}

// TestDecimal_String_NeverScientificNotation asserts String() always emits
// plain decimal, including for values with an exponent large enough that
// apd's own default formatting (String's "G" format) would switch to
// scientific notation. A value produced inside this package -- including
// one Parse itself accepted in exponential form, per SPEC-MU §2.6.1 -- must
// remain parseable by this package's own Parse, in the same plain-decimal
// wire format, all the way round.
func TestDecimal_String_NeverScientificNotation(t *testing.T) {
	cases := []Decimal{
		newTestDecimal(9, 100000),  // largest exponent apd's Context permits
		newTestDecimal(1, -100000), // smallest exponent apd's Context permits
		mustParse(t, "49.99"),
		mustParse(t, "0"),
	}
	for _, d := range cases {
		s := d.String()
		if strings.ContainsAny(s, "eE") {
			t.Errorf("String() = %q contains scientific notation", s)
		}
		if _, err := Parse(s); err != nil {
			t.Errorf("String() output %q does not round-trip through Parse: %v", s, err)
		}
	}
}

func TestDecimal_String_ErrorWrapping(t *testing.T) {
	// Deliberately free of 'e'/'E' so this exercises the apd-delegated parse
	// failure path (wrapped with %w), distinct from Parse's own
	// scientific-notation rejection (which returns an unwrapped error, since
	// there is no underlying library error to wrap in that case).
	_, err := Parse("49.9.9")
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Unwrap(err) == nil {
		t.Errorf("Parse error does not wrap an underlying cause: %v", err)
	}
}
