package mu

import (
	"strings"
	"testing"
	"time"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU22 asserts every field SPEC-MU §8.3 constrains for a MU-22
// conformance vector: CheckID, Class (ClassS), Severity (SeverityWarn),
// and Outcome.
func wantMU22(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU22(in)
	if err != nil {
		t.Fatalf("checkMU22 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU22 applicable = false, want true")
	}
	if res.CheckID() != "MU-22" {
		t.Errorf("CheckID() = %q, want MU-22", res.CheckID())
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

// entityObservation builds a single-entry Observation window entry
// carrying entity -- the shape MU-22 needs (unlike observationsOf, which
// deliberately never sets an entity, since MU-20/MU-21 never read one).
func entityObservation(t *testing.T, value, entity string) Observation {
	t.Helper()
	return Observation{Value: mustParse(t, value), Entity: entity, HasEntity: true}
}

func TestCheckMU22_MU_V88(t *testing.T) {
	// MU-V88: MU-22, prior "1234.00" same entity in window | 12340.00 |
	// FAIL @ warn (extra trailing zero, 10x).
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "12340.00"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{entityObservation(t, "1234.00", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeFail)
}

func TestCheckMU22_MU_V89(t *testing.T) {
	// MU-V89: MU-22, entity identified, no prior value in window |
	// 12340.00 | INDETERMINATE
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "12340.00"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{entityObservation(t, "1234.00", "cust-2")}, // different entity
	}
	wantMU22(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU22_MU_V90(t *testing.T) {
	// MU-V90: MU-22, caller identifies no entity | any | INDETERMINATE
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "12340.00"),
		HasEntity:    false,
		Registry:     mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{entityObservation(t, "1234.00", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU22_EmptyStringEntity_HasEntityFalsePriorNeverMatches pins
// matchingEntityObservations' own documented rule directly (mu.go's
// Observation doc comment, matchingEntityObservations' own doc comment
// below): a prior with HasEntity false never matches Input.Entity, not
// even the empty string. Input.Entity is deliberately "" here, with
// HasEntity true (a caller that identified an entity and it happened to
// be the empty string, not a caller that identified none at all -- see
// Input's own HasEntity doc comment for why those are different states).
// The one prior in the window has no entity of its own (HasEntity false)
// but carries a value ("1234.00") that would pattern-match and FAIL
// against Input.Value ("12340.00", an extra trailing zero at 10x) if it
// were compared at all. Without this test, a comparison as loose as
// `obs.Entity == entity` (dropping HasEntity entirely, since Go's zero
// value for Entity is also "") would pass every other test in this file
// and only diverge here.
func TestCheckMU22_EmptyStringEntity_HasEntityFalsePriorNeverMatches(t *testing.T) {
	in := Input{
		Field:     "arguments.amount",
		Value:     mustParse(t, "12340.00"),
		Entity:    "",
		HasEntity: true,
		Registry:  mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{
			{Value: mustParse(t, "1234.00"), HasEntity: false},
		},
	}
	wantMU22(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU22_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "100"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU22(in)
	if err != nil {
		t.Fatalf("checkMU22 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU22 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU22_QuantityKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "100"),
		Registry: mustRegistry(t, field.NewQuantityDeclaration()),
	}
	_, applicable, err := checkMU22(in)
	if err != nil {
		t.Fatalf("checkMU22 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU22 applicable = true, want false (quantity is not in MU-22's Applies-to set)")
	}
}

func TestCheckMU22_ValueNotCoercible_Indeterminate(t *testing.T) {
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Entity:              "cust-1",
		HasEntity:           true,
		Registry:            mustRegistry(t, field.NewMoneyDeclaration()),
		Observations:        []Observation{entityObservation(t, "1234.00", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU22_NoPatternMatch_Pass(t *testing.T) {
	// An ordinary, unrelated prior value: no digit pattern, no FAIL.
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "77.50"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{entityObservation(t, "1234.00", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomePass)
}

// TestCheckMU22_ExtraTrailingZero_PriorLarger_Fail is vector MU-V88's
// mirror image: the *prior* value carries the extra trailing zero and the
// *current* value is the smaller one (1234 against a recorded 12340).
// digitPatternMatch checks both directions (isExtraTrailingZero(a, b) ||
// isExtraTrailingZero(b, a)), and this is also the only test in this file
// where the current value is the smaller of the pair, so it is the one
// that exercises exceedsMagnitudeRatio's swap branch (the prior, not the
// current value, is "larger").
func TestCheckMU22_ExtraTrailingZero_PriorLarger_Fail(t *testing.T) {
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "1234"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{entityObservation(t, "12340", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeFail)
}

func TestCheckMU22_RepeatedDigitPattern_Fail(t *testing.T) {
	// "12234" against "1234": a repeated "2" inserted, magnitude ratio
	// 12234/1234 ~= 9.9x, past the 5x bound.
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "12234"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{entityObservation(t, "1234", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeFail)
}

// TestCheckMU22_AdjacentTranspositionPattern_MagnitudeTooSmall_Pass pins a
// real property of an adjacent-digit swap rather than an arbitrary
// example: swapping two adjacent digits, each 1-9 (a real transported
// number can never carry a leading zero, on either side of the swap),
// changes an equal-length number by at most a factor of 91/19 (~4.79) --
// achieved at the two leading digits, 9 and 1, with every remaining digit
// zero, and *smaller* for any other digit pair or any position further
// from the front. That maximum is still under MU-22's 5x bound, so a
// pure single-digit transposition, on its own, can never satisfy MU-22's
// magnitude requirement -- proven directly here at that maximum rather
// than asserted in prose.
//
// This is a specification defect, not an implementation gap. SPEC-MU §6
// names "a single-digit transposition" as one of three independent
// patterns that can trigger MU-22, but isAdjacentTransposition requires
// equal-length operands, and an equal-length adjacent-digit swap can
// change a value by at most 91/19 (~4.79) -- under MU-22's own 5x floor
// in every case, and it cannot combine with either of the other two named
// patterns (both of which require unequal-length operands) to clear it
// either. No correct implementation of MU-22 as specified can make this
// pattern fire. The gap is recorded against SPEC-MU; it is not fixed
// here. isAdjacentTransposition's
// own detection logic is verified directly in TestIsAdjacentTransposition
// below, independent of the magnitude gate this test is about.
func TestCheckMU22_AdjacentTranspositionPattern_MagnitudeTooSmall_Pass(t *testing.T) {
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "91"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{entityObservation(t, "19", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomePass)
}

func TestIsAdjacentTransposition(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"swap at front", "9134", "1934", true},
		{"swap in middle", "19340", "19430", true},
		{"identical", "1234", "1234", false},
		{"different length", "123", "1234", false},
		{"three differing positions", "111", "222", false},
		{"two differing but not adjacent", "1234", "3214", false},
		{"two adjacent positions, not a swap", "1234", "1564", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAdjacentTransposition(tc.a, tc.b); got != tc.want {
				t.Errorf("isAdjacentTransposition(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestIsRepeatedDigitInsertion(t *testing.T) {
	cases := []struct {
		name        string
		long, short string
		want        bool
	}{
		{"duplicate to the right", "12234", "1234", true},
		{"duplicate earlier in the string", "12214", "1214", true},
		{"duplicate digit at the very end", "12344", "1234", true},
		{"insertion is not a duplicate of either neighbour", "12934", "1234", false},
		{"insertion at the very front, not a duplicate", "1234", "234", false},
		{"more than a single character's worth of difference", "19934", "1234", false},
		{"same length", "1234", "1234", false},
		{"length differs by more than one", "123456", "1234", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRepeatedDigitInsertion(tc.long, tc.short); got != tc.want {
				t.Errorf("isRepeatedDigitInsertion(%q, %q) = %v, want %v", tc.long, tc.short, got, tc.want)
			}
		})
	}
}

// TestIsRepeatedDigitInsertion_LinearTime is a regression guard for the
// quadratic form this function's own doc comment describes replacing: at
// a ~100,000-digit input (an extreme-exponent value's decimal.String()),
// the quadratic form measured over a second for a single call. This scan
// is linear and must stay well under that on any machine.
func TestIsRepeatedDigitInsertion_LinearTime(t *testing.T) {
	short := strings.Repeat("1", 100000)
	long := short[:50000] + "1" + short[50000:] // a duplicate '1' inserted at the midpoint
	start := time.Now()
	got := isRepeatedDigitInsertion(long, short)
	elapsed := time.Since(start)
	if !got {
		t.Fatal("isRepeatedDigitInsertion(long, short) = false, want true")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("isRepeatedDigitInsertion took %v for a 100,000-digit input, want well under 200ms (linear, not quadratic)", elapsed)
	}
}

func TestIsExtraTrailingZero(t *testing.T) {
	cases := []struct {
		name        string
		long, short string
		want        bool
	}{
		{"extra zero", "12340", "1234", true},
		{"same length", "1234", "1234", false},
		{"extra digit is not zero", "12341", "1234", false},
		{"length differs by more than one", "123400", "1234", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExtraTrailingZero(tc.long, tc.short); got != tc.want {
				t.Errorf("isExtraTrailingZero(%q, %q) = %v, want %v", tc.long, tc.short, got, tc.want)
			}
		})
	}
}

func TestDigitsOnly(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1234", "1234"},
		{"-1234.00", "123400"},
		{"0.5", "05"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := digitsOnly(tc.in); got != tc.want {
			t.Errorf("digitsOnly(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCheckMU22_OppositeSign_Pass(t *testing.T) {
	// A sign flip is not a digit-transposition typo: -1234.00 against
	// 12340.00 shares the extra-trailing-zero digit shape but must not
	// match, since the two values differ in sign.
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, "12340.00"),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{entityObservation(t, "-1234.00", "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomePass)
}

func TestCheckMU22_MultipleEntityObservations_AnyMatchFails(t *testing.T) {
	// Several prior observations for the same entity, only one of which
	// matches a pattern: the FAIL must not depend on which position the
	// matching entry occupies in the window.
	in := Input{
		Field:     "arguments.amount",
		Value:     mustParse(t, "12340.00"),
		Entity:    "cust-1",
		HasEntity: true,
		Registry:  mustRegistry(t, field.NewMoneyDeclaration()),
		Observations: []Observation{
			entityObservation(t, "77.50", "cust-1"),
			entityObservation(t, "1234.00", "cust-1"),
			entityObservation(t, "999.99", "cust-1"),
		},
	}
	wantMU22(t, in, verdict.OutcomeFail)
}

// TestCheckMU22_ExceedsMagnitudeRatioOverflow_Indeterminate reaches
// exceedsMagnitudeRatio's own factor.Mul(smaller) overflow branch. "92"
// and "29" followed by 99999 zeros apiece are a genuine adjacent-digit
// transposition (the leading two digits swapped) at the maximum digit
// count this package's arithmetic supports, so digitPatternMatch matches
// before the magnitude check ever runs. The smaller of the two (the one
// starting "29...") has a two-digit leading run that carries into a third
// digit once multiplied by MU-22's factor of 5, pushing the product's
// digit count past the supported boundary -- unlike vector MU-V88's
// pair, whose smaller value (1234.00) is nowhere near that boundary.
func TestCheckMU22_ExceedsMagnitudeRatioOverflow_Indeterminate(t *testing.T) {
	prior := "92" + strings.Repeat("0", 99999)
	value := "29" + strings.Repeat("0", 99999)
	in := Input{
		Field:        "arguments.amount",
		Value:        mustParse(t, value),
		Entity:       "cust-1",
		HasEntity:    true,
		Registry:     mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{entityObservation(t, prior, "cust-1")},
	}
	wantMU22(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU22_OrderIndependent_FailSurvivesAnErroringPrior pins
// checkMU22's own order-independence directly: the same two same-entity
// priors, in both orders, must produce the same verdict. priorOverflow
// pairs with value the same way
// TestCheckMU22_ExceedsMagnitudeRatioOverflow_Indeterminate's prior does
// -- an adjacent-digit-transposition match whose own magnitude comparison
// overflows. priorFail is value with one fewer trailing zero: an
// ordinary, non-overflowing extra-trailing-zero match at exactly 10x,
// which FAILs on its own (mirroring vector MU-V88's own ratio). An
// earlier version of checkMU22 returned on the first prior that decided
// anything, so evaluating priorOverflow before priorFail returned
// INDETERMINATE without ever looking at priorFail, while the reverse
// order returned FAIL -- two different verdicts from the same set of
// priors, differing only in the order Input.Observations happened to
// list them. Both orders below must return FAIL: a real match must not
// be hidden by an unrelated prior's arithmetic failure, regardless of
// which one Observations lists first.
func TestCheckMU22_OrderIndependent_FailSurvivesAnErroringPrior(t *testing.T) {
	value := "29" + strings.Repeat("0", 99999)
	priorOverflow := "92" + strings.Repeat("0", 99999)
	priorFail := "29" + strings.Repeat("0", 99998)

	errorFirst := Input{
		Field:     "arguments.amount",
		Value:     mustParse(t, value),
		Entity:    "cust-1",
		HasEntity: true,
		Registry:  mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{
			entityObservation(t, priorOverflow, "cust-1"),
			entityObservation(t, priorFail, "cust-1"),
		},
	}
	wantMU22(t, errorFirst, verdict.OutcomeFail)

	matchFirst := Input{
		Field:     "arguments.amount",
		Value:     mustParse(t, value),
		Entity:    "cust-1",
		HasEntity: true,
		Registry:  mustRegistry(t, field.NewDecimalDeclaration()),
		Observations: []Observation{
			entityObservation(t, priorFail, "cust-1"),
			entityObservation(t, priorOverflow, "cust-1"),
		},
	}
	wantMU22(t, matchFirst, verdict.OutcomeFail)
}
