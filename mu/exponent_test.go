package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// usdInput builds an Input for a money/major_units/currency_field-declared
// field with the given value and currency code, injecting tbl as the
// resolvable ISO 4217 table. Shared by every MU-14 test below since they
// all vary only the value, the sibling currency code, and (sometimes) the
// declaration itself.
func usdInput(t *testing.T, tbl tables.CurrencyTable, value, currencyCode string) Input {
	t.Helper()
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits))
	return Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, value),
		Registry: mustRegistry(t, decl),
		Tables:   Tables{ISO4217: tbl},
		Vals:     stringVals(map[string]string{"arguments.currency": currencyCode}),
	}
}

func wantMU14(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, err := checkMU14(in)
	if err != nil {
		t.Fatalf("checkMU14 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-14" {
		t.Errorf("CheckID() = %q, want MU-14", res.CheckID())
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

func TestCheckMU14_Vector_04(t *testing.T) {
	// Vector 4: money, major_units, USD | 49.99 | PASS | MU-14
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.99", "USD"), verdict.OutcomePass)
}

func TestCheckMU14_Vector_05(t *testing.T) {
	// Vector 5: money, major_units, USD | 49.999 | FAIL | MU-14
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.999", "USD"), verdict.OutcomeFail)
}

func TestCheckMU14_Vector_06(t *testing.T) {
	// Vector 6: money, major_units, JPY | 500.5 | FAIL | MU-14 (exponent 0)
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "500.5", "JPY"), verdict.OutcomeFail)
}

func TestCheckMU14_Vector_07(t *testing.T) {
	// Vector 7: money, major_units, KWD | 4.999 | PASS | MU-14 (exponent 3)
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "4.999", "KWD"), verdict.OutcomePass)
}

func TestCheckMU14_TrailingZeros(t *testing.T) {
	// Trailing zeros count as decimal places, per DecimalPlaces()'s
	// contract: "49.90" is 2 places (PASS under USD's exponent 2); "49.900"
	// is 3 places (FAIL). Neither reduces to a canonical "2 significant
	// fractional digits" reading.
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.90", "USD"), verdict.OutcomePass)
	wantMU14(t, usdInput(t, tbl, "49.900", "USD"), verdict.OutcomeFail)
}

func TestCheckMU14_NegativeAndZero(t *testing.T) {
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "-49.99", "USD"), verdict.OutcomePass)
	wantMU14(t, usdInput(t, tbl, "-49.999", "USD"), verdict.OutcomeFail)
	wantMU14(t, usdInput(t, tbl, "0", "USD"), verdict.OutcomePass)
	wantMU14(t, usdInput(t, tbl, "0.00", "USD"), verdict.OutcomePass)
}

func TestCheckMU14_Vector_101(t *testing.T) {
	// Vector 101: money, USD (no scale) | 49.999 | INDETERMINATE | MU-14
	decl := mustCurrencyField(t, field.NewMoneyDeclaration()) // no WithScale
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.999"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_Vector_102(t *testing.T) {
	// Vector 102: money, major_units, no currency_field | 49.999 |
	// INDETERMINATE | MU-14 (no exponent)
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits) // no WithCurrencyField
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.999"),
		Registry: mustRegistry(t, decl),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: field.Registry{},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_WrongKind_Indeterminate(t *testing.T) {
	decl := field.NewDecimalDeclaration()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_NoScale_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_MinorUnits_Indeterminate(t *testing.T) {
	// scale: minor_units is MU-01's territory, not MU-14's.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration(), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_NoCurrencyField_Indeterminate(t *testing.T) {
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_NoSiblingValue_Indeterminate(t *testing.T) {
	// currency_field declared, but Input.Vals carries nothing at that path.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU14_UnresolvableCurrencyCode_Indeterminate(t *testing.T) {
	// Sibling value present but not a code the injected table recognizes.
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.99", "ZZZ"), verdict.OutcomeIndeterminate)
}

func TestCheckMU14_CurrencyLowercase_Resolves(t *testing.T) {
	// resolveCurrency upper-cases before lookup; a lowercase sibling value
	// still resolves.
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.99", "usd"), verdict.OutcomePass)
}

func TestCheckMU14_NoMinorUnitExponent_Indeterminate(t *testing.T) {
	// XAU is a real ISO 4217 code with no minor unit exponent at all --
	// MinorUnitExponent() reports (_, false). The exponent must never be
	// defaulted to guess an answer here.
	tbl := tables.NewISO4217Table()
	wantMU14(t, usdInput(t, tbl, "49.99", "XAU"), verdict.OutcomeIndeterminate)
}

func TestCheckMU14_EmptyTables_Indeterminate(t *testing.T) {
	// A caller that never injected a CurrencyTable at all (Tables{} zero
	// value) misses every lookup, same as an unresolvable code.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}
