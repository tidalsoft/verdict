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

func TestCheckMU01_Vector_01(t *testing.T) {
	// Vector 1: money, minor_units | 4999 | PASS | MU-01
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU01_Vector_02(t *testing.T) {
	// Vector 2: money, minor_units | 49.99 | FAIL | MU-01
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU01_Vector_03(t *testing.T) {
	// Vector 3: money, minor_units | 49.00 | FAIL | MU-01 (type contradicts declaration)
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.00"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU01_Vector_08(t *testing.T) {
	// Vector 8: money (no scale) | 4999 | INDETERMINATE | MU-01
	decl := field.NewMoneyDeclaration() // no WithScale
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU01_Vector_42(t *testing.T) {
	// Vector 42: money, minor_units | 100e-2 | FAIL | MU-01 (two decimal
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
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU01_ScaleMajorUnits_Pass(t *testing.T) {
	// scale: major_units → PASS at MU-01 (defer to MU-14)
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
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
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
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
			res, err := checkMU01(in)
			if err != nil {
				t.Fatalf("checkMU01 unexpected error: %v", err)
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
			res, err := checkMU01(in)
			if err != nil {
				t.Fatalf("checkMU01 unexpected error: %v", err)
			}
			if res.Outcome() != verdict.OutcomeFail {
				t.Errorf("Outcome() = %v, want FAIL for %s", res.Outcome(), tc.value)
			}
		})
	}
}

func TestCheckMU01_NoDeclaration_Indeterminate(t *testing.T) {
	// No declaration in registry → INDETERMINATE
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: field.Registry{}, // empty registry
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU01_WrongKind_Indeterminate(t *testing.T) {
	// Non-money kind → INDETERMINATE (type assertion fails)
	decl := field.NewDecimalDeclaration()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU01(in)
	if err != nil {
		t.Fatalf("checkMU01 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
