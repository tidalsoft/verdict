package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

func TestCheckMU05_Vector_24(t *testing.T) {
	// Vector 24: quantity, mass, kg, unit_required | 12 | FAIL | MU-05
	// (unit required)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithUnitRequired()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-05" {
		t.Errorf("CheckID() = %q, want MU-05", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU05_Vector_58(t *testing.T) {
	// Vector 58: quantity, mass, unit_required, unit_field | "12 lb", unit
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
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU05_Vector_59(t *testing.T) {
	// Vector 59: quantity, mass, unit_required, no bounds | "12 lb" | PASS
	// | MU-05 (decomposed with no number read)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithUnitRequired()
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
	}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU05_UnitFromFieldOnly_Pass(t *testing.T) {
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass")).WithUnitRequired()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
	}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU05_UnitRequiredFalse_Indeterminate(t *testing.T) {
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass") // no WithUnitRequired
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU05_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU05_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU05(in)
	if err != nil {
		t.Fatalf("checkMU05 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
