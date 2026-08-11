package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

func mustDimension(t *testing.T, d field.QuantityDeclaration, dim string) field.QuantityDeclaration {
	t.Helper()
	out, err := d.WithDimension(dim)
	if err != nil {
		t.Fatalf("WithDimension(%q) unexpected error: %v", dim, err)
	}
	return out
}

// mustUnitField declares "arguments.unit" as d's unit_field -- the only
// sibling path this package's tests ever condition unit_field on; a
// parameter no call site varies is dead flexibility (golangci-lint's
// unparam agrees; see mustCurrencyField in scale_test.go for the same
// reasoning).
func mustUnitField(t *testing.T, d field.QuantityDeclaration) field.QuantityDeclaration {
	t.Helper()
	const path = "arguments.unit"
	out, err := d.WithUnitField(path)
	if err != nil {
		t.Fatalf("WithUnitField(%q) unexpected error: %v", path, err)
	}
	return out
}

func mustCanonicalUnit(t *testing.T, d field.QuantityDeclaration, unit string) field.QuantityDeclaration {
	t.Helper()
	out, err := d.WithCanonicalUnit(unit)
	if err != nil {
		t.Fatalf("WithCanonicalUnit(%q) unexpected error: %v", unit, err)
	}
	return out
}

func unitTables() Tables {
	return Tables{Units: tables.NewUnitRegistry()}
}

// wantMU04 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU04: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-04's only severity), and Outcome.
func wantMU04(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU04(in)
	if err != nil {
		t.Fatalf("checkMU04 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU04 applicable = false, want true")
	}
	if res.CheckID() != "MU-04" {
		t.Errorf("CheckID() = %q, want MU-04", res.CheckID())
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

func TestCheckMU04_Vector_22(t *testing.T) {
	// Vector 22: quantity, mass, kg | "12 lb" | PASS -> 5.443 kg | MU-04
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomePass)
}

func TestCheckMU04_Vector_23(t *testing.T) {
	// Vector 23: quantity, mass, kg | "12 m" | FAIL | MU-04 (dimension)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "m",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeFail)
}

func TestCheckMU04_Vector_25(t *testing.T) {
	// Vector 25: quantity, mass, kg | "12 flurbs" | FAIL | MU-04 (unknown unit)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "flurbs",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeFail)
}

func TestCheckMU04_Vector_26(t *testing.T) {
	// Vector 26: temperature, K | "50 °F" | PASS -> 283.15 K | MU-04 (affine)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "temperature")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "50"),
		EmbeddedUnit:    "°F",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomePass)
}

func TestCheckMU04_Vector_56(t *testing.T) {
	// Vector 56: quantity, mass, kg, unit_required | 12 | INDETERMINATE |
	// MU-04 (unit absent)
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithUnitRequired()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Tables:   unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU04_Vector_57(t *testing.T) {
	// Vector 57: quantity, canonical_unit kg (no dimension) | "12 kg" |
	// INDETERMINATE | MU-04
	decl := mustCanonicalUnit(t, field.NewQuantityDeclaration(), "kg")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "kg",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU04_Vector_106(t *testing.T) {
	// Vector 106: quantity, mass, unit_field | "12 lb", unit field "kg" |
	// INDETERMINATE | MU-04 (unit_conflict)
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Vals:            map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU04_Vector_121(t *testing.T) {
	// Vector 121: quantity, mass, unit_field | "12 kg", unit field
	// "flurbs" | INDETERMINATE | MU-04 (unit_conflict; the conflicting
	// unit_field string need not itself be registry-recognised)
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "kg",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Vals:            map[string]field.Value{"arguments.unit": field.NewStringValue("flurbs")},
		Tables:          unitTables(),
	}
	wantMU04(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU04_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	_, applicable, err := checkMU04(in)
	if err != nil {
		t.Fatalf("checkMU04 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU04 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU04_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	_, applicable, err := checkMU04(in)
	if err != nil {
		t.Fatalf("checkMU04 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU04 applicable = true, want false (wrong kind)")
	}
}

// TestCheckMU04_UnitFieldOnly_Resolves confirms a quantity value whose
// number arrived without an embedded unit (a JSON number, or a string
// decomposition that carried no unit) still resolves via unit_field
// alone.
func TestCheckMU04_UnitFieldOnly_Resolves(t *testing.T) {
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
		Tables:   unitTables(),
	}
	wantMU04(t, in, verdict.OutcomePass)
}
