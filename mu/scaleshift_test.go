package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU21 asserts every field SPEC-MU §8.3 constrains for a MU-21
// conformance vector: CheckID, Class (ClassS), Severity (SeverityWarn),
// and Outcome.
func wantMU21(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU21(in)
	if err != nil {
		t.Fatalf("checkMU21 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU21 applicable = false, want true")
	}
	if res.CheckID() != "MU-21" {
		t.Errorf("CheckID() = %q, want MU-21", res.CheckID())
	}
	if res.Class() != verdict.ClassS {
		t.Errorf("Class() = %v, want ClassS", res.Class())
	}
	if res.Severity() != verdict.SeverityWarn {
		t.Errorf("Severity() = %v, want SeverityWarn", res.Severity())
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

// mu21WindowAt builds a 300-observation window every entry of which is
// exactly median -- the simplest window whose median is median and whose
// interquartile range is the single point [median, median], so any value
// other than median itself is "outside the interquartile range" by
// construction. That isolates MU-21's own 100x/1-in-100 relative test,
// which every vector below exists to exercise, from the separate
// question of how quartiles behave on a window that is not degenerate --
// covered instead by TestCheckMU21_WithinInterquartileRange_Pass, which
// builds a spread window specifically to hold that branch.
func mu21WindowAt(t *testing.T, median string) []Observation {
	t.Helper()
	return observationsOf(t, repeat(median, 300)...)
}

func TestCheckMU21_MU_V40(t *testing.T) {
	// MU-V40: MU-21, 300 obs, median 5000 | 500000 | FAIL @ warn
	// (exactly 100x the median, and outside the -- degenerate, single-point
	// -- interquartile range).
	decl := field.NewMoneyDeclaration() // no scale declared
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500000"),
		Registry:     mustRegistry(t, decl),
		Observations: mu21WindowAt(t, "5000"),
	}
	wantMU21(t, in, verdict.OutcomeFail)
}

func TestCheckMU21_MU_V111(t *testing.T) {
	// MU-V111: MU-21, 199 observations (below the floor) | 5000 |
	// INDETERMINATE
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "5000"),
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, repeat("5000", 199)...),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU21_MU_V112(t *testing.T) {
	// MU-V112: MU-21, 300 obs, median 0 (degenerate case) | 500 |
	// INDETERMINATE, never PASS -- "a test that cannot separate a scale
	// shift from an ordinary value has not looked for one."
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500"),
		Registry:     mustRegistry(t, decl),
		Observations: mu21WindowAt(t, "0"),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU21_ScaleDeclared_NotApplicable(t *testing.T) {
	// SPEC-MU §2.5.1: MU-21 applies to money only where scale is NOT
	// declared -- "a field that declares its scale is MU-01's, and this
	// check does not run on it." Both scale values are tested, since the
	// gate is on declaredness, not on which value was declared.
	for _, s := range []field.Scale{field.ScaleMajorUnits, field.ScaleMinorUnits} {
		decl := mustScale(t, field.NewMoneyDeclaration(), s)
		in := Input{
			Field:        "arguments.amount",
			Value:        mustParse(t, "500000"),
			Registry:     mustRegistry(t, decl),
			Observations: mu21WindowAt(t, "5000"),
		}
		_, applicable, err := checkMU21(in)
		if err != nil {
			t.Fatalf("checkMU21 unexpected error: %v", err)
		}
		if applicable {
			t.Errorf("checkMU21 applicable = true for scale %v, want false", s)
		}
	}
}

func TestCheckMU21_NonMoneyKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500000"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: mu21WindowAt(t, "5000"),
	}
	_, applicable, err := checkMU21(in)
	if err != nil {
		t.Fatalf("checkMU21 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU21 applicable = true, want false (decimal, not money)")
	}
}

func TestCheckMU21_ValueNotCoercible_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
		Observations:        mu21WindowAt(t, "5000"),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU21_NotNearShift_Pass(t *testing.T) {
	// A value nowhere near 100x or 1/100 of the median, and not degenerate,
	// must PASS regardless of where it falls relative to the IQR.
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "5000"),
		Registry:     mustRegistry(t, decl),
		Observations: mu21WindowAt(t, "5000"),
	}
	wantMU21(t, in, verdict.OutcomePass)
}

// TestCheckMU21_WithinInterquartileRange_Pass builds a spread window (not
// every entry identical) whose 100x-median value nonetheless falls inside
// its own interquartile range, so MU-21's second, independent condition
// -- "and outside the interquartile range" -- is the one this test
// isolates: a value can be within 10% of a 100x/1-in-100 shift and still
// PASS, provided it is not also outside the window's own normal spread.
func TestCheckMU21_WithinInterquartileRange_Pass(t *testing.T) {
	// 300 observations: 100 at 4000, 100 at 5000, 100 at 500000. Sorted,
	// the median (observation 150 of 300, averaging observations 149/150
	// zero-indexed) is 5000: the lower and upper halves split evenly at
	// 150 each. Q1 (median of the lower 150 -- all 4000/5000 split 100/50)
	// and Q3 (median of the upper 150 -- all 5000/500000 split 50/100) are
	// wide enough that 500000 itself, exactly 100x the median, falls
	// inside [Q1, Q3].
	values := append(append(repeat("4000", 100), repeat("5000", 100)...), repeat("500000", 100)...)
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500000"),
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, values...),
	}
	wantMU21(t, in, verdict.OutcomePass)
}

func TestCheckMU21_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500000"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU21(in)
	if err != nil {
		t.Fatalf("checkMU21 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU21 applicable = true, want false (no declaration)")
	}
}

// majorityWindow builds a 201-observation (odd) window of value repeated
// 101 times alongside "0" repeated 100 times, so the window's own median
// -- a direct index pick, never an Add -- is exactly value, regardless of
// value's own magnitude. Every overflow test below needs a window whose
// median lands on a value it chooses precisely, without the median
// computation's own arithmetic being what fails.
func majorityWindow(t *testing.T, value string) []Observation {
	t.Helper()
	values := append(repeat(value, 101), repeat("0", 100)...)
	return observationsOf(t, values...)
}

// TestCheckMU21_MedianOverflow_Indeterminate mirrors
// TestCheckMU20_MedianOverflow_Indeterminate (outlier_test.go): an
// even-count window of 200 identical huge observations, so median's own
// Add(sorted[99], sorted[100]) overflows.
func TestCheckMU21_MedianOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration() // no scale declared
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1"),
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, repeat(hugeAtMaxExponent, mu21MinObservations)...),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_HundredXOverflow_Indeterminate reaches checkMU21's own
// hundredX := med.Mul(mu21Hundred()) overflow branch: a median at the
// supported exponent range's boundary (9e100000), picked directly via
// majorityWindow so the median computation itself does not overflow, but
// multiplying by 100 (a 3-digit coefficient) pushes the product's digit
// count past the boundary.
func TestCheckMU21_HundredXOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1"),
		Registry:     mustRegistry(t, decl),
		Observations: majorityWindow(t, hugeAtMaxExponent),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_OneHundredthOverflow_Indeterminate reaches checkMU21's
// own oneHundredth := med.ScaleByExponent(-2) overflow branch: a median
// at the supported exponent range's *negative* boundary (1e-99999), so
// shifting its exponent two further places negative exceeds the
// supported minimum.
func TestCheckMU21_OneHundredthOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1"),
		Registry:     mustRegistry(t, decl),
		Observations: majorityWindow(t, "1e-99999"),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_WithinToleranceHundredXOverflow_Indeterminate reaches
// checkMU21's own error branch on its first withinRelativeTolerance call
// (against hundredX) -- and, in the same call, withinRelativeTolerance's
// own internal value.Sub(target) overflow branch. The median (9e99998)
// is two orders of magnitude below the supported boundary, so hundredX
// (median * 100) lands exactly at the boundary without itself
// overflowing; Input.Value is the opposite-signed value at that same
// boundary, so comparing the two overflows.
func TestCheckMU21_WithinToleranceHundredXOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "-"+hugeAtMaxExponent),
		Registry:     mustRegistry(t, decl),
		Observations: majorityWindow(t, "9e99998"),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_QuartilesUpperOverflow_Indeterminate reaches checkMU21's
// own quartiles error branch, and within it, quartiles' q3 (upper half)
// computation. The 400-observation window is 299 copies of an ordinary
// value (5000, covering both the overall median and the entire lower
// half, so neither of those overflows), followed by 101 copies of the
// boundary value -- ensuring the upper half's own middle pair (positions
// 299 and 300 of the full sorted window) are both at the boundary, so
// their Add overflows. Input.Value is chosen equal to hundredX exactly,
// so the near-100x check passes trivially and quartiles is reached at
// all.
func TestCheckMU21_QuartilesUpperOverflow_Indeterminate(t *testing.T) {
	values := append(repeat("5000", 299), repeat(hugeAtMaxExponent, 101)...)
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500000"), // exactly median (5000) * 100
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, values...),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_WithinToleranceOneHundredthSubOverflow_Indeterminate
// reaches checkMU21's own second withinRelativeTolerance call (against
// oneHundredth) failing where the first call (against hundredX)
// succeeds: an in.Value pinned at the exponent ceiling has a *larger*
// exponent gap against oneHundredth than against hundredX, since the two
// targets sit four decimal places apart from each other (hundredX is
// med*100, oneHundredth is med/100) and hundredX is the nearer one. The
// window (201 copies of "1", an odd count, so the median is a direct
// pick and its own computation cannot overflow) makes both targets
// ordinary values in their own right; only comparing each against
// Input.Value exposes the gap, and only the larger of the two gaps
// overflows.
func TestCheckMU21_WithinToleranceOneHundredthSubOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1e100000"),
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, repeat("1", 201)...),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_WithinToleranceOneHundredthBoundOverflow_Indeterminate
// reaches checkMU21's own second withinRelativeTolerance call (against
// oneHundredth), and within it, that call's own bound Mul overflow --
// independently of the first call (against hundredX), which succeeds
// outright, and independently of either call's Sub, which also succeeds.
// The median (1e-99998, picked directly via majorityWindow, so its own
// computation cannot overflow) puts hundredX two decimal places above the
// exponent floor and oneHundredth two places *below* it, at the floor
// itself -- multiplying oneHundredth's magnitude by the 10% tolerance
// constant (adjusted exponent -1) then pushes the product one place past
// the floor, the same low-end failure mu20ZConstant's own multiplication
// can hit (outlier.go). Input.Value equals the median exactly, so both
// Subs land on ordinary, nearby results and neither one is what fails.
func TestCheckMU21_WithinToleranceOneHundredthBoundOverflow_Indeterminate(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1e-99998"),
		Registry:     mustRegistry(t, decl),
		Observations: majorityWindow(t, "1e-99998"),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU21_QuartilesLowerOverflow_Indeterminate reaches checkMU21's
// own quartiles error branch, and within it, quartiles' q1 (lower half)
// computation -- the branch SPEC-MU imposes no same-signed requirement
// against, since a money field legitimately carries refunds alongside
// charges. The 200-observation window is 51 copies of a huge negative
// value, 48 zeros, two ordinary positive values, and 99 copies of another
// ordinary positive value: the overall median (averaging the two ordinary
// values) does not overflow, but the lower half's own two middle entries
// both land inside the huge-negative run, so their Add does. Input.Value
// is chosen equal to hundredX exactly, so the near-100x check passes
// trivially and quartiles is reached at all.
func TestCheckMU21_QuartilesLowerOverflow_Indeterminate(t *testing.T) {
	values := append(repeat("-"+hugeAtMaxExponent, 51), repeat("0", 48)...)
	values = append(values, repeat("1", 2)...)
	values = append(values, repeat("2", 99)...) // 51 + 48 + 2 + 99 = 200
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "100"), // exactly median (1) * 100
		Registry:     mustRegistry(t, decl),
		Observations: observationsOf(t, values...),
	}
	wantMU21(t, in, verdict.OutcomeIndeterminate)
}
