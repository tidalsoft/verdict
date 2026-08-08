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

func TestCheckMU03_BothResolvedEqual_Pass(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "usd",
		},
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-03" {
		t.Errorf("CheckID() = %q, want MU-03", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU03_BothResolvedUnequal_Fail(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "EUR",
		},
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU03_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewDecimalDeclaration()),
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_NoCurrencyField_Indeterminate(t *testing.T) {
	decl := mustTargetCurrencyField(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.target_currency": "USD"},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_SourceSiblingAbsent_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals:     map[string]string{"arguments.target_currency": "USD"},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_SourceUnresolvable_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: map[string]string{
			"arguments.currency":        "ZZZ",
			"arguments.target_currency": "USD",
		},
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_NoTargetCurrencyField_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.currency": "USD"},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_TargetSiblingAbsent_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals:     map[string]string{"arguments.currency": "USD"},
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU03_TargetUnresolvable_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, fullCurrencyDeclaration(t)),
		Vals: map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "ZZZ",
		},
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU03(in)
	if err != nil {
		t.Fatalf("checkMU03 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
