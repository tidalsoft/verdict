package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// mustScale returns d with s applied via WithScale, failing the test on the
// error WithScale returns for an invalid Scale. Every call site in this
// package's tests passes a valid Scale constant, so the error branch is
// never expected to fire; the helper exists so each call site doesn't
// repeat the same three lines of error handling (mirrors mustParse and
// mustRegistry in scaffold_test.go).
func mustScale(t *testing.T, d field.MoneyDeclaration, s field.Scale) field.MoneyDeclaration {
	t.Helper()
	out, err := d.WithScale(s)
	if err != nil {
		t.Fatalf("WithScale(%v) unexpected error: %v", s, err)
	}
	return out
}

// mustCurrencyField returns d with "arguments.currency" applied via
// WithCurrencyField, failing the test on the error WithCurrencyField
// returns for an empty path. See mustScale for why this exists as a
// helper. The path is fixed rather than a parameter because every MU-14
// test declares the same sibling field; a parameter no call site varies is
// dead flexibility, not a real seam (golangci-lint's unparam agrees).
func mustCurrencyField(t *testing.T, d field.MoneyDeclaration) field.MoneyDeclaration {
	t.Helper()
	const path = "arguments.currency"
	out, err := d.WithCurrencyField(path)
	if err != nil {
		t.Fatalf("WithCurrencyField(%q) unexpected error: %v", path, err)
	}
	return out
}

// wantMU01 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU01: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-01's only severity), and Outcome.
func wantMU01(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU01 applicable = false, want true")
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
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

func TestCheckMU01_MU_V1(t *testing.T) {
	// MU-V1: money, minor_units | 4999 | PASS | MU-01
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomePass)
}

func TestCheckMU01_MU_V2(t *testing.T) {
	// MU-V2: money, minor_units | 49.99 | FAIL | MU-01
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomeFail)
}

func TestCheckMU01_MU_V3(t *testing.T) {
	// MU-V3: money, minor_units | 49.00 | FAIL | MU-01 (type contradicts declaration)
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.00"),
		Registry: mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomeFail)
}

func TestCheckMU01_MU_V8(t *testing.T) {
	// MU-V8: money (no scale) | 4999 | INDETERMINATE | MU-01
	decl := field.NewMoneyDeclaration() // no WithScale
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU01_MU_V42(t *testing.T) {
	// MU-V42: money, minor_units | 100e-2 | FAIL | MU-01 (two decimal
	// places, no point). decimal.Parse now accepts exponential notation
	// per SPEC-MU §2.6.1's decimal-text grammar (see decimal.Parse's own
	// doc comment) -- "100e-2" carries two decimal places, the same as
	// "1.00", despite containing no literal decimal point.
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "100e-2"),
		Registry: mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomeFail)
}

func TestCheckMU01_ScaleMajorUnits_Pass(t *testing.T) {
	// scale: major_units → PASS at MU-01 (defer to MU-14)
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	res, applicable, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU01 applicable = false, want true")
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU01_ScaleMajorUnits_WithCurrency_Pass(t *testing.T) {
	// scale: major_units with currency still PASS at MU-01 -- MU-01 defers
	// the exponent bound to MU-14 regardless of whether a currency is even
	// resolvable.
	tbl := tables.NewISO4217Table()
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Tables:   Tables{ISO4217: tbl},
	}
	res, applicable, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU01 applicable = false, want true")
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU01_MinorUnits_IntegerValues_Pass(t *testing.T) {
	// Various integer representations in minor units should PASS, including
	// zero and negative values.
	cases := []struct {
		name  string
		value string
	}{
		{"zero", "0"},
		{"positive", "4999"},
		{"negative", "-4999"},
		{"large", "9007199254740993"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: mustRegistry(t, decl),
			}
			res, applicable, err := checkMU01(in)
			if err != nil {
				t.Fatalf("checkMU01 unexpected error: %v", err)
			}
			if !applicable {
				t.Fatal("checkMU01 applicable = false, want true")
			}
			if res.Outcome() != verdict.OutcomePass {
				t.Errorf("Outcome() = %v, want PASS for %s", res.Outcome(), tc.value)
			}
		})
	}
}

func TestCheckMU01_MinorUnits_FractionalValues_Fail(t *testing.T) {
	// Various fractional representations in minor units should FAIL,
	// including a trailing-zero fractional part and negative values.
	cases := []struct {
		name  string
		value string
	}{
		{"two decimal places", "49.99"},
		{"trailing zeros", "49.00"},
		{"one decimal place", "49.9"},
		{"three decimal places", "49.999"},
		{"negative fractional", "-49.99"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: mustRegistry(t, decl),
			}
			res, applicable, err := checkMU01(in)
			if err != nil {
				t.Fatalf("checkMU01 unexpected error: %v", err)
			}
			if !applicable {
				t.Fatal("checkMU01 applicable = false, want true")
			}
			if res.Outcome() != verdict.OutcomeFail {
				t.Errorf("Outcome() = %v, want FAIL for %s", res.Outcome(), tc.value)
			}
		})
	}
}

func TestCheckMU01_NoDeclaration_NotApplicable(t *testing.T) {
	// No declaration in registry → not applicable
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: field.Registry{}, // empty registry
	}
	_, applicable, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU01 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU01_WrongKind_NotApplicable(t *testing.T) {
	// Non-money kind → not applicable (type assertion fails)
	decl := field.NewDecimalDeclaration()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	_, applicable, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU01 applicable = true, want false (wrong kind)")
	}
}

func TestCheckMU01_MU_V43(t *testing.T) {
	// MU-V43: money, minor_units | "1,234" (not coercible) |
	// INDETERMINATE | MU-01 (value_not_coercible). MU-01 is value-dependent
	// (§2.6.3's table), so a value the coercion gate could not read is
	// reported INDETERMINATE without ever consulting DecimalPlaces() --
	// Value/Provenance must not be read once ValueCoercionFailed is true
	// (see Input's own doc comment), which is why this test leaves Value at
	// its zero value entirely: reading it here would be the defect this
	// test exists to catch.
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	wantMU01(t, in, verdict.OutcomeIndeterminate)
}
