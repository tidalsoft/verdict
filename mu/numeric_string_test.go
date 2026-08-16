package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU09 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU09: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-09's only severity), and Outcome.
func wantMU09(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU09(in)
	if err != nil {
		t.Fatalf("checkMU09 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU09 applicable = false, want true")
	}
	if res.CheckID() != "MU-09" {
		t.Errorf("CheckID() = %q, want MU-09", res.CheckID())
	}
	if res.Class() != verdict.ClassD {
		t.Errorf("Class() = %v, want ClassD", res.Class())
	}
	if res.Severity() != verdict.SeverityBlock {
		t.Errorf("Severity() = %v, want SeverityBlock", res.Severity())
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

func numericStringInput(s string) Input {
	return Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(field.NewMoneyDeclaration()),
		HasRawValue: true,
		RawValue:    field.NewStringValue(s),
	}
}

// mustRegistryT is mustRegistry without a *testing.T receiver, for use in
// table-driven literals built outside a subtest's own t. Every call site
// in this file immediately hands the result to a function that does have
// a *testing.T and can report a construction failure -- but
// field.NewRegistry's only failure modes (empty path, nil declaration)
// never occur here, since every caller passes the fixed literal
// "arguments.amount" and a real field.Declaration.
func mustRegistryT(decl field.Declaration) field.Registry {
	r, err := field.NewRegistry(map[string]field.Declaration{"arguments.amount": decl})
	if err != nil {
		panic(err)
	}
	return r
}

func TestCheckMU09_MU_V12(t *testing.T) {
	// MU-V12: numeric field | "1,234" | FAIL | MU-09 (ambiguous)
	wantMU09(t, numericStringInput("1,234"), verdict.OutcomeFail)
}

func TestCheckMU09_MU_V13(t *testing.T) {
	// MU-V13: numeric field | "1,23" | PASS -> 1.23 | MU-09
	wantMU09(t, numericStringInput("1,23"), verdict.OutcomePass)
}

func TestCheckMU09_MU_V14(t *testing.T) {
	// MU-V14: numeric field | "1.234,56" | PASS -> 1234.56 | MU-09
	wantMU09(t, numericStringInput("1.234,56"), verdict.OutcomePass)
}

func TestCheckMU09_MU_V15(t *testing.T) {
	// MU-V15: numeric field | "$49.99" | FAIL | MU-09 (symbol present)
	wantMU09(t, numericStringInput("$49.99"), verdict.OutcomeFail)
}

func TestCheckMU09_MU_V51(t *testing.T) {
	// MU-V51: numeric field | "abc" | INDETERMINATE | MU-09 (no enumerated shape)
	wantMU09(t, numericStringInput("abc"), verdict.OutcomeIndeterminate)
}

func TestCheckMU09_CleanDecimal_Pass(t *testing.T) {
	wantMU09(t, numericStringInput("49.99"), verdict.OutcomePass)
}

func TestCheckMU09_USStyleGrouping_Pass(t *testing.T) {
	// "1,234.56": comma groups, dot decimal -- the mirror image of vector
	// 14's German-style "1.234,56", pinning the other branch of "last
	// separator is the decimal."
	wantMU09(t, numericStringInput("1,234.56"), verdict.OutcomePass)
}

func TestCheckMU09_MixedSeparators_CandidateStillMalformed_Fail(t *testing.T) {
	// Exactly one '.' and one ',' -- the "unambiguous" shape this
	// package's reading accepts -- but stripping the grouping separator
	// and re-pointing the decimal one still does not produce valid
	// decimal text ("a1.2" is not a number), so the re-parse itself must
	// fail and this must FAIL rather than PASS.
	wantMU09(t, numericStringInput("a,1.2"), verdict.OutcomeFail)
}

func TestCheckMU09_RepeatedSeparators_Fail(t *testing.T) {
	// This package's own disclosed reading of SPEC-MU §5's undefined
	// "unambiguous": more than one of either separator is ambiguous and
	// FAILs outright (classifyMixedSeparators' own doc comment explains
	// why, and this task's report flags the reading as an open question).
	wantMU09(t, numericStringInput("1,234,567.89"), verdict.OutcomeFail)
}

func TestCheckMU09_SpaceInString_Fail(t *testing.T) {
	wantMU09(t, numericStringInput("49.99 USD"), verdict.OutcomeFail)
}

func TestCheckMU09_NonBreakingSpace_Fail(t *testing.T) {
	// U+00A0 NO-BREAK SPACE, explicitly named in SPEC-MU §5's
	// Evaluation clause alongside plain spaces and currency symbols.
	wantMU09(t, numericStringInput("49.99 USD"), verdict.OutcomeFail)
}

func TestCheckMU09_SingleCommaNotDigitShaped_Indeterminate(t *testing.T) {
	// "a,bcd" has exactly one comma and no dot, but neither side is a
	// digit run -- classifySingleComma must not match this shape, and the
	// string carries no currency symbol or whitespace either, so this
	// falls all the way through to INDETERMINATE.
	wantMU09(t, numericStringInput("a,bcd"), verdict.OutcomeIndeterminate)
}

func TestCheckMU09_NotApplicable(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{
			"no declaration",
			Input{Field: "arguments.amount"},
		},
		{
			"declaration kind MU-09 does not apply to",
			Input{
				Field:       "arguments.amount",
				Registry:    mustRegistryT(field.NewQuantityDeclaration()),
				HasRawValue: true,
				RawValue:    field.NewStringValue("1,234"),
			},
		},
		{
			"value did not arrive as a string",
			Input{
				Field:       "arguments.amount",
				Registry:    mustRegistryT(field.NewMoneyDeclaration()),
				HasRawValue: true,
				RawValue:    field.NewNumberValue(mustParse(t, "49.99")),
			},
		},
		{
			"value absent",
			Input{
				Field:    "arguments.amount",
				Registry: mustRegistryT(field.NewMoneyDeclaration()),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, applicable, err := checkMU09(tc.in)
			if err != nil {
				t.Fatalf("checkMU09 unexpected error: %v", err)
			}
			if applicable {
				t.Fatal("checkMU09 applicable = true, want false")
			}
		})
	}
}

func TestCheckMU09_DecimalAndPercentageKinds(t *testing.T) {
	// MU-09 applies to decimal and percentage exactly as to money
	// (SPEC-MU §2.5.1).
	cases := []struct {
		name string
		decl field.Declaration
	}{
		{"decimal", field.NewDecimalDeclaration()},
		{"percentage", field.NewPercentageDeclaration()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:       "arguments.amount",
				Registry:    mustRegistryT(tc.decl),
				HasRawValue: true,
				RawValue:    field.NewStringValue("1,234"),
			}
			wantMU09(t, in, verdict.OutcomeFail)
		})
	}
}

func TestIsASCIIDigits(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"123", true},
		{"12a", false},
		{"-1", false},
	}
	for _, tc := range cases {
		if got := isASCIIDigits(tc.in); got != tc.want {
			t.Errorf("isASCIIDigits(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsSignedDigits(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"123", true},
		{"+123", true},
		{"-123", true},
		{"-", false},
		{"1a", false},
	}
	for _, tc := range cases {
		if got := isSignedDigits(tc.in); got != tc.want {
			t.Errorf("isSignedDigits(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestCheckMU09_TrailingPointIsIndeterminate covers a string that looks like
// it parses but is not decimal text under SPEC-MU §2.6.1: the grammar
// requires a digit after any point, and the prose says "no trailing point".
// The underlying arbitrary-precision library accepts "5." and returns 5, so
// before decimal.Parse enforced the grammar this check reported PASS on the
// strength of it parsing. "5." matches none of MU-09's enumerated shapes, so
// the answer it owes is INDETERMINATE -- a wrong PASS here is the failure
// this product exists to prevent. The leading-point and trailing-zero forms
// are decimal text and must keep passing.
func TestCheckMU09_TrailingPointIsIndeterminate(t *testing.T) {
	wantMU09(t, numericStringInput("5."), verdict.OutcomeIndeterminate)
	wantMU09(t, numericStringInput(".5"), verdict.OutcomePass)
	wantMU09(t, numericStringInput("5.0"), verdict.OutcomePass)
}
