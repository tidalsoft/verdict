package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

func mustDomain(t *testing.T, d field.PercentageDeclaration, dom field.Domain) field.PercentageDeclaration {
	t.Helper()
	out, err := d.WithDomain(dom)
	if err != nil {
		t.Fatalf("WithDomain(%v) unexpected error: %v", dom, err)
	}
	return out
}

func TestCheckMU13_Vector_27(t *testing.T) {
	// Vector 27: percentage, unit_interval | 50 | FAIL | MU-13
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-13" {
		t.Errorf("CheckID() = %q, want MU-13", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
	if res.Severity() != verdict.SeverityBlock {
		t.Errorf("Severity() = %v, want block", res.Severity())
	}
}

func TestCheckMU13_Vector_28(t *testing.T) {
	// Vector 28: percentage, unit_interval | 0.5 | PASS | MU-13
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Vector_29(t *testing.T) {
	// Vector 29: percentage, hundred | 0.5 | FAIL @ warn | MU-13 (asymmetric)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
	if res.Severity() != verdict.SeverityWarn {
		t.Errorf("Severity() = %v, want warn", res.Severity())
	}
}

func TestCheckMU13_Vector_75(t *testing.T) {
	// Vector 75: percentage, unit_interval | -50 | FAIL | MU-13
	// (magnitude above the domain)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-50"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU13_Vector_76(t *testing.T) {
	// Vector 76: percentage, unit_interval | -0.5 | PASS | MU-13 (a signed
	// rate; sign is not this check's)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-0.5"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Vector_77(t *testing.T) {
	// Vector 77: percentage, hundred | 250 | PASS | MU-13 (no upper bound)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "250"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Vector_78(t *testing.T) {
	// Vector 78: percentage (no domain) | 0.5 | INDETERMINATE | MU-13
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewPercentageDeclaration()),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU13_Vector_94(t *testing.T) {
	// Vector 94: percentage, hundred | -0.5 | FAIL @ warn | MU-13
	// (magnitude in the ambiguous band)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-0.5"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
	if res.Severity() != verdict.SeverityWarn {
		t.Errorf("Severity() = %v, want warn", res.Severity())
	}
}

func TestCheckMU13_UnitInterval_Boundary_Pass(t *testing.T) {
	// value == 1 is not > 1: inclusive boundary passes.
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Hundred_AboveOne_Pass(t *testing.T) {
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Hundred_ZeroExcepted_Pass(t *testing.T) {
	// SPEC-MU §3: "value ≤ 1 and value ≠ 0" -- zero is explicitly excepted
	// from the asymmetric FAIL branch.
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU13_Hundred_BoundaryOne_Fail(t *testing.T) {
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
	if res.Severity() != verdict.SeverityWarn {
		t.Errorf("Severity() = %v, want warn", res.Severity())
	}
}

func TestCheckMU13_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: field.Registry{},
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU13_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU13_NoDomain_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewPercentageDeclaration()),
	}
	res, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
