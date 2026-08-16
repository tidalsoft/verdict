package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// hugeAtMaxExponent is decimal text for the exact decimal 9 * 10^100000.
// Its order of magnitude sits exactly at the boundary the underlying
// arithmetic library supports (decimal.Decimal.Add/Sub/Mul's own doc
// comments: "the range this package's underlying arithmetic supports,
// ±10^5" digits of exponent): it parses successfully on its own, but
// combining it with another value of comparable magnitude -- by Add, by
// Sub against an opposite sign, or by multiplying it by a factor whose
// own leading digit is 2 or more -- pushes the result's own magnitude
// past that boundary and returns an error. Every arithmetic-overflow test
// in this file, scaleshift_test.go, and transposition_test.go builds its
// window around this one constant.
const hugeAtMaxExponent = "9e100000"

// wantMU20 asserts every field SPEC-MU §8.3 constrains for a MU-20
// conformance vector: CheckID, Class (ClassS), Severity (SeverityWarn --
// MU-20's only severity this package ever constructs; promotion to block
// happens outside it, see statResult's doc comment), and Outcome.
func wantMU20(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU20(in)
	if err != nil {
		t.Fatalf("checkMU20 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU20 applicable = false, want true")
	}
	if res.CheckID() != "MU-20" {
		t.Errorf("CheckID() = %q, want MU-20", res.CheckID())
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

// observationsOf builds an []Observation of len(values) entries, none
// carrying an entity -- the shape every MU-20 test needs, since MU-20
// never reads Observation.Entity (statistics.go's observationValues
// discards it).
func observationsOf(t *testing.T, values ...string) []Observation {
	t.Helper()
	out := make([]Observation, len(values))
	for i, v := range values {
		out[i] = Observation{Value: mustParse(t, v)}
	}
	return out
}

// repeat returns n copies of v, for building a window dominated by one
// value.
func repeat(v string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// mu20WorkedWindow builds the 300-observation window MU-V86 and MU-V87
// both declare ("300 obs, median 5000, MAD 100"): 150 observations at
// 4900 and 150 at 5100. SPEC-MU §8.2 says a vector's declared observation
// history is supplied as given, so the count matters, not only the
// resulting statistics -- 150+150 is the window this package actually
// evaluates for those two vectors, not a same-statistics stand-in of a
// different size. Sorted, the two middle (150th and 151st) entries are
// 4900 and 5100, averaging to a median of exactly 5000; every one of the
// 300 absolute deviations from that median is exactly 100 (|4900-5000|
// and |5100-5000| both equal 100), so the MAD -- the median of an
// all-identical list -- is 100 regardless of which entries the even-count
// average picks.
func mu20WorkedWindow(t *testing.T) []Observation {
	t.Helper()
	values := append(repeat("4900", 150), repeat("5100", 150)...)
	return observationsOf(t, values...)
}

func TestCheckMU20_MU_V38(t *testing.T) {
	// MU-V38: outlier, 199 observations (below the floor) | any | INDETERMINATE
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "100"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, repeat("100", 199)...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU20_MU_V39(t *testing.T) {
	// MU-V39: outlier, MAD = 0 (every observation identical) | any
	// differing value | INDETERMINATE, never FAIL -- SPEC-MU §6's
	// Degenerate case: "an engine dividing by zero here will flag every
	// non-identical value, which is the worst possible behaviour."
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "500"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, repeat("100", 250)...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU20_MU_V86(t *testing.T) {
	// MU-V86: MU-20, 300 obs, median 5000, MAD 100 | 50 | FAIL @ warn
	// (far below the median -- the two-sided test SPEC-MU §6 requires).
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "50"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: mu20WorkedWindow(t),
	}
	wantMU20(t, in, verdict.OutcomeFail)
}

func TestCheckMU20_MU_V87(t *testing.T) {
	// MU-V87: MU-20, 300 obs, median 5000, MAD 100 | 5300 | PASS
	// modified_z = 0.6745 * 300 / 100 = 2.0235, within the default
	// threshold of 8.
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "5300"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: mu20WorkedWindow(t),
	}
	wantMU20(t, in, verdict.OutcomePass)
}

func TestCheckMU20_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "100"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU20(in)
	if err != nil {
		t.Fatalf("checkMU20 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU20 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU20_TimestampKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "100"),
		Registry: mustRegistry(t, field.NewTimestampDeclaration()),
	}
	_, applicable, err := checkMU20(in)
	if err != nil {
		t.Fatalf("checkMU20 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU20 applicable = true, want false (wrong kind)")
	}
}

func TestCheckMU20_ValueNotCoercible_Indeterminate(t *testing.T) {
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, field.NewDecimalDeclaration()),
		Observations:        mu20WorkedWindow(t),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU20_AllFourApplicableKinds pins SPEC-MU §2.5.1's Applies-to
// set directly: money, decimal, percentage, and quantity all evaluate
// MU-20 for real (not merely "not applicable"), each against the same
// worked window.
func TestCheckMU20_AllFourApplicableKinds(t *testing.T) {
	cases := []struct {
		name string
		decl field.Declaration
	}{
		{"money", mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)},
		{"decimal", field.NewDecimalDeclaration()},
		{"percentage", field.NewPercentageDeclaration()},
		{"quantity", field.NewQuantityDeclaration()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:        "arguments.amount",
				Value:        mustParse(t, "50"),
				Registry:     mustRegistry(t, tc.decl),
				Observations: mu20WorkedWindow(t),
			}
			wantMU20(t, in, verdict.OutcomeFail)
		})
	}
}

// TestCheckMU20_MedianOverflow_Indeterminate exercises the branch median()
// itself cannot reach through this package's other tests: an even-count
// window whose two middle order statistics are both huge enough that
// averaging them (Add) overflows. All 200 observations are identical, so
// both middle entries are the same value and the Add is exactly the
// doubling decimal.Decimal.Add's own overflow test uses.
func TestCheckMU20_MedianOverflow_Indeterminate(t *testing.T) {
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, repeat(hugeAtMaxExponent, mu20MinObservations)...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU20_MADOverflow_Indeterminate reaches medianAbsoluteDeviation's
// own overflow branch: an odd-count window (so the *overall* median is a
// direct pick, not an Add) whose median is huge and positive, with one
// observation of the opposite huge sign -- that single observation's
// deviation from the median (a Sub of two same-magnitude, opposite-signed
// huge values) overflows.
func TestCheckMU20_MADOverflow_Indeterminate(t *testing.T) {
	values := append([]string{"-" + hugeAtMaxExponent}, repeat("0", 99)...)
	values = append(values, repeat(hugeAtMaxExponent, 101)...) // 1 + 99 + 101 = 201, odd
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, values...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU20_ValueDiffOverflow_Indeterminate reaches checkMU20's own
// diff := in.Value.Sub(med) overflow branch. The window is three equal
// groups (67 each, 201 total, odd) of three huge values one order of
// magnitude apart at the top of the supported range -- 7e100000, 8e100000,
// 9e100000 -- whose median (8e100000) and MAD (1e100000, both picked
// directly with no Add, so neither computation itself overflows) are both
// huge but nonzero. Input.Value is the opposite-signed huge value, whose
// Sub against the median overflows.
func TestCheckMU20_ValueDiffOverflow_Indeterminate(t *testing.T) {
	values := append(repeat("7e100000", 67), repeat("8e100000", 67)...)
	values = append(values, repeat(hugeAtMaxExponent, 67)...)
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "-"+hugeAtMaxExponent),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, values...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU20_ZScoreLHSUnderflow_Indeterminate reaches checkMU20's own
// lhs := mu20ZConstant().Mul(diff.Abs()) branch at the *low* end of the
// supported exponent range, not the high end: mu20ZConstant (0.6745) has
// adjusted exponent -1, so multiplying by it shifts a diff already at the
// exponent floor one place past it. The window (100 copies of -1, one 0,
// 100 copies of 1) has median 0 and MAD 1, neither of which is huge, so
// nothing about computing them overflows; only the final multiplication
// against Input.Value's own extreme diff does.
func TestCheckMU20_ZScoreLHSUnderflow_Indeterminate(t *testing.T) {
	values := append(append(repeat("-1", 100), "0"), repeat("1", 100)...)
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1e-100000"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, values...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU20_ThresholdMadOverflow_Indeterminate reaches checkMU20's own
// rhs := mu20Threshold().Mul(mad) overflow branch. The window is three
// equal groups (67 each) at -9e100000, 0, and 9e100000: the overall
// median is 0 (picked directly, the middle group), and the MAD -- the
// median of the 201 per-observation deviations from 0, which are 67
// zeros and 134 copies of 9e100000 -- is 9e100000, also picked directly
// (both without an Add, so neither construction step itself overflows).
// Input.Value is 0, so diff and lhs are trivially 0; only mad's own
// multiplication by the threshold (8) overflows -- 8 has a leading digit
// large enough, unlike mu20ZConstant's 0.6745, to push a max-magnitude
// operand's coefficient to a 2-digit carry.
func TestCheckMU20_ThresholdMadOverflow_Indeterminate(t *testing.T) {
	values := append(repeat("-"+hugeAtMaxExponent, 67), repeat("0", 67)...)
	values = append(values, repeat(hugeAtMaxExponent, 67)...)
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "0"),
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: observationsOf(t, values...),
	}
	wantMU20(t, in, verdict.OutcomeIndeterminate)
}
