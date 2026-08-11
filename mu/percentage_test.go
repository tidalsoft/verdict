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

// wantMU13 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU13: CheckID, Class (ClassD), Severity, and
// Outcome. Severity is a parameter, unlike this package's other want*
// helpers, because MU-13 is the one check in this package whose severity
// is not fixed: its domain: hundred low-magnitude branch is warn while
// every other branch (including its own INDETERMINATE) is block -- see
// checkMU13's own doc comment.
func wantMU13(t *testing.T, in Input, want verdict.Outcome, wantSeverity verdict.Severity) {
	t.Helper()
	res, applicable, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU13 applicable = false, want true")
	}
	if res.CheckID() != "MU-13" {
		t.Errorf("CheckID() = %q, want MU-13", res.CheckID())
	}
	if res.Class() != verdict.ClassD {
		t.Errorf("Class() = %v, want ClassD", res.Class())
	}
	if res.Severity() != wantSeverity {
		t.Errorf("Severity() = %v, want %v", res.Severity(), wantSeverity)
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

func TestCheckMU13_Vector_27(t *testing.T) {
	// Vector 27: percentage, unit_interval | 50 | FAIL | MU-13
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU13_Vector_28(t *testing.T) {
	// Vector 28: percentage, unit_interval | 0.5 | PASS | MU-13
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU13_Vector_29(t *testing.T) {
	// Vector 29: percentage, hundred | 0.5 | FAIL @ warn | MU-13 (asymmetric)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomeFail, verdict.SeverityWarn)
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
	wantMU13(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
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
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU13_Vector_77(t *testing.T) {
	// Vector 77: percentage, hundred | 250 | PASS | MU-13 (no upper bound)
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "250"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU13_Vector_78(t *testing.T) {
	// Vector 78: percentage (no domain) | 0.5 | INDETERMINATE | MU-13
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewPercentageDeclaration()),
	}
	wantMU13(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
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
	wantMU13(t, in, verdict.OutcomeFail, verdict.SeverityWarn)
}

func TestCheckMU13_UnitInterval_Boundary_Pass(t *testing.T) {
	// value == 1 is not > 1: inclusive boundary passes.
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainUnitInterval)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU13_Hundred_AboveOne_Pass(t *testing.T) {
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
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
	wantMU13(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU13_Hundred_BoundaryOne_Fail(t *testing.T) {
	decl := mustDomain(t, field.NewPercentageDeclaration(), field.DomainHundred)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomeFail, verdict.SeverityWarn)
}

func TestCheckMU13_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU13 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU13_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	_, applicable, err := checkMU13(in)
	if err != nil {
		t.Fatalf("checkMU13 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU13 applicable = true, want false (wrong kind)")
	}
}

func TestCheckMU13_NoDomain_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, field.NewPercentageDeclaration()),
	}
	wantMU13(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}

func TestCheckMU13_ValueNotCoercible_Indeterminate(t *testing.T) {
	// SPEC-MU §2.6.3: MU-13 is value-dependent, so a value the coercion
	// gate could not read is INDETERMINATE before domain is even
	// consulted -- Value must not be read once ValueCoercionFailed is true.
	decl, err := field.NewPercentageDeclaration().WithDomain(field.DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain unexpected error: %v", err)
	}
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	wantMU13(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}
