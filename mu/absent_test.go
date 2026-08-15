package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU05 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU05: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-05's only severity), and Outcome.
func wantMU05(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU05 applicable = false, want true")
	}
	if res.CheckID() != "MU-05" {
		t.Errorf("CheckID() = %q, want MU-05", res.CheckID())
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

func TestCheckMU05_MU_V24(t *testing.T) {
	// MU-V24: quantity, mass, kg, unit_required | 12 | FAIL | MU-05
	// (unit required)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithUnitRequired()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
	}
	wantMU05(t, in, verdict.OutcomeFail)
}

func TestCheckMU05_MU_V58(t *testing.T) {
	// MU-V58: quantity, mass, unit_required, unit_field | "12 lb", unit
	// field "kg" | INDETERMINATE | MU-05 (unit_conflict)
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass")).WithUnitRequired()
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Vals:            map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
	}
	wantMU05(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU05_MU_V59(t *testing.T) {
	// MU-V59: quantity, mass, unit_required, no bounds | "12 lb" | PASS
	// | MU-05 (decomposed with no number read)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithUnitRequired()
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
	}
	wantMU05(t, in, verdict.OutcomePass)
}

func TestCheckMU05_UnitFromFieldOnly_Pass(t *testing.T) {
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass")).WithUnitRequired()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
	}
	wantMU05(t, in, verdict.OutcomePass)
}

func TestCheckMU05_UnitRequiredFalse_NotApplicable(t *testing.T) {
	// SPEC-MU §2.5.1/§2.5.2: unit_required defaults to false, an actual
	// substituted value, so its absence is a coherent "bare number is
	// fine" declaration -- not applicable, not INDETERMINATE.
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass") // no WithUnitRequired
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
	}
	_, applicable, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU05 applicable = true, want false (unit_required not declared)")
	}
}

func TestCheckMU05_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	_, applicable, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU05 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU05_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	_, applicable, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU05 applicable = true, want false (wrong kind)")
	}
}
