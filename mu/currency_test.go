package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

func mustTargetCurrencyField(t *testing.T, d field.MoneyDeclaration) field.MoneyDeclaration {
	t.Helper()
	const path = "arguments.target_currency"
	out, err := d.WithTargetCurrencyField(path)
	if err != nil {
		t.Fatalf("WithTargetCurrencyField(%q) unexpected error: %v", path, err)
	}
	return out
}

func fullCurrencyDeclaration(t *testing.T) field.MoneyDeclaration {
	t.Helper()
	return mustTargetCurrencyField(t, mustCurrencyField(t, field.NewMoneyDeclaration()))
}

func TestCurrencyCodeShape(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{"upper", "USD", "USD", true},
		{"lower", "usd", "USD", true},
		{"mixed", "Usd", "USD", true},
		// SPEC-MU §2.4.2: shape, not table membership -- a well-shaped code
		// the ISO 4217 table does not carry still resolves as a currency
		// code for MU-03's purposes.
		{"not an ISO 4217 member but right shape", "ZZZ", "ZZZ", true},
		{"too short", "US", "", false},
		{"too long", "USDD", "", false},
		{"contains a digit", "US1", "", false},
		{"empty", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := currencyCodeShape(tc.in)
			if ok != tc.ok {
				t.Fatalf("currencyCodeShape(%q) ok = %v, want %v", tc.in, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("currencyCodeShape(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// wantMU03 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU03: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-03's only severity), and Outcome.
func wantMU03(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.CheckID() != "MU-03" {
		t.Errorf("CheckID() = %q, want MU-03", res.CheckID())
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

func TestCheckMU03_Vector_46(t *testing.T) {
	// Vector 46: source "USD", target "USD" -> PASS
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "USD",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomePass)
}

func TestCheckMU03_Vector_47(t *testing.T) {
	// Vector 47: source "USD", target "EUR" -> FAIL
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "EUR",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomeFail)
}

func TestCheckMU03_Vector_48(t *testing.T) {
	// Vector 48: source "usd", target "USD" -> PASS (case-insensitive)
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "usd",
			"arguments.target_currency": "USD",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomePass)
}

func TestCheckMU03_Vector_49(t *testing.T) {
	// Vector 49: target_currency_field rooted at state., no state envelope
	// -> INDETERMINATE (target_currency_unresolved). This package has no
	// notion of "state" distinct from Vals (see
	// field.MoneyDeclaration.TargetCurrencyField's doc comment), so "no
	// state envelope" is modelled the same way any unresolved sibling is:
	// absent from Vals.
	decl := mustTargetCurrencyField(t, mustCurrencyField(t, field.NewMoneyDeclaration()))
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU03_Vector_50(t *testing.T) {
	// Vector 50: currency_field path absent, target "USD" ->
	// INDETERMINATE (source_currency_unresolved)
	decl := mustTargetCurrencyField(t, mustCurrencyField(t, field.NewMoneyDeclaration()))
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.target_currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU03_Vector_108(t *testing.T) {
	// Vector 108: both currency_field and target_currency_field paths
	// absent -> INDETERMINATE (both_currencies_unresolved)
	decl := fullCurrencyDeclaration(t)
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU03(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU03_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	_, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU03 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU03_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewDecimalDeclaration()),
	}
	_, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU03 applicable = true, want false (wrong kind)")
	}
}

func TestCheckMU03_NoCurrencyField_Indeterminate(t *testing.T) {
	decl := mustTargetCurrencyField(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.target_currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_SourceSiblingAbsent_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals:     stringVals(map[string]string{"arguments.target_currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

// TestCheckMU03_SourceWrongShape_Indeterminate replaces an earlier version
// of this test that used "ZZZ" as its "unresolvable" source currency --
// which is exactly three ASCII letters and, per SPEC-MU §2.4.2's shape
// (not membership) rule, now resolves as a currency code for MU-03's
// purposes even though no ISO 4217 entry carries it. This test uses a
// genuinely wrong-shaped source instead (four letters) to exercise the
// same INDETERMINATE outcome for the reason MU-03 actually requires it.
func TestCheckMU03_SourceWrongShape_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "USDD",
			"arguments.target_currency": "USD",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

// TestCheckMU03_WellShapedButUntabledCurrency_ComparesNormally is the
// positive counterpart: SPEC-MU §2.4.2 says a well-shaped code the ISO
// 4217 table does not carry "resolves as a currency code" -- MU-03 reads
// no table at all, so two untabled-but-identically-shaped codes compare
// exactly like two ordinary ones.
func TestCheckMU03_WellShapedButUntabledCurrency_ComparesNormally(t *testing.T) {
	decl := fullCurrencyDeclaration(t)
	registry := mustRegistry(t, decl)
	tbl := Tables{ISO4217: tables.NewISO4217Table()}

	equal := Input{
		Field:    "arguments.amount",
		Registry: registry,
		Vals: stringVals(map[string]string{
			"arguments.currency":        "ZZZ",
			"arguments.target_currency": "zzz",
		}),
		Tables: tbl,
	}
	res, applicable, err := checkMU03(equal)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS (untabled but matching shape)", res.Outcome())
	}

	unequal := Input{
		Field:    "arguments.amount",
		Registry: registry,
		Vals: stringVals(map[string]string{
			"arguments.currency":        "ZZZ",
			"arguments.target_currency": "USD",
		}),
		Tables: tbl,
	}
	res2, applicable2, err := checkMU03(unequal)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable2 {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res2.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res2.Outcome())
	}
}

func TestCheckMU03_NoTargetCurrencyField_NotApplicable(t *testing.T) {
	// SPEC-MU §2.5.1/§2.5.2: target_currency_field is MU-03's gate --
	// "off by default, enabled per field" -- not a required-input gap.
	decl := mustCurrencyField(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	_, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU03 applicable = true, want false (target_currency_field not declared)")
	}
}

func TestCheckMU03_TargetSiblingAbsent_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_TargetWrongShape_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "EU",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

// TestCheckMU03_SourceResolvesToNonString_Indeterminate exercises
// resolveCurrencyCodeShape's "resolved but not a string" branch: the
// sibling path resolves to a comparable-but-non-string field.Value (a
// JSON number), which is not a currency code under any reading of
// SPEC-MU §2.4.2.
func TestCheckMU03_SourceResolvesToNonString_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: map[string]field.Value{
			"arguments.currency":        field.NewNumberValue(mustParse(t, "840")),
			"arguments.target_currency": field.NewStringValue("USD"),
		},
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, applicable, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU03 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

// TestCheckMU14_CurrencyResolvesToNonString_Indeterminate exercises
// resolveDeclaredCurrency's "resolved but not a string" branch.
func TestCheckMU14_CurrencyResolvesToNonString_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.currency": field.NewBoolValue(true)},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU14(t, in, verdict.OutcomeIndeterminate)
}
