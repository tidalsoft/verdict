package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

func wantMU15(t *testing.T, in Input, wantOutcome verdict.Outcome) {
	t.Helper()
	res, err := checkMU15(in)
	if err != nil {
		t.Fatalf("checkMU15 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-15" {
		t.Errorf("CheckID() = %q, want MU-15", res.CheckID())
	}
	if res.Outcome() != wantOutcome {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), wantOutcome)
	}
	// MU-15's default severity is warn for every outcome, not just FAIL --
	// see checkMU15's own doc comment.
	if res.Severity() != verdict.SeverityWarn {
		t.Errorf("Severity() = %v, want SeverityWarn", res.Severity())
	}
}

func TestCheckMU15_Vector_83(t *testing.T) {
	// Vector 83: quantity, mass, canonical_unit kg, max "10" | "12 lb" |
	// PASS | MU-15
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomePass)
}

func TestCheckMU15_Vector_85(t *testing.T) {
	// Vector 85: quantity, mass, canonical_unit kg, max "10" | 12 |
	// INDETERMINATE | MU-15 (nothing to convert from)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Tables:   unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_Vector_97(t *testing.T) {
	// Vector 97: quantity, temperature, canonical_unit K, tolerance "0" |
	// "100 °F" | FAIL @ warn | MU-15 (5/9 does not round-trip in decimal)
	decl, err := mustDimension(t, field.NewQuantityDeclaration(), "temperature").
		WithCanonicalUnit("K")
	if err != nil {
		t.Fatalf("WithCanonicalUnit unexpected error: %v", err)
	}
	decl = decl.WithTolerance(mustParse(t, "0"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "100"),
		EmbeddedUnit:    "°F",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeFail)
}

func TestCheckMU15_Vector_107(t *testing.T) {
	// Vector 107: quantity, mass, canonical_unit kg, unit_field | "12 lb",
	// unit field "kg" | INDETERMINATE | MU-15 (unit_conflict)
	decl := mustUnitField(t,
		mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Vals:            map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_Vector_120(t *testing.T) {
	// Vector 120: quantity, mass, canonical_unit kg | "12 m" |
	// INDETERMINATE | MU-15 (unit recognised, wrong dimension)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "m",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_NoCanonicalUnit_Indeterminate(t *testing.T) {
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass") // no canonical_unit
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_CanonicalUnitNotInRegistry_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "flurbs")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_ValueUnitNotInRegistry_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "flurbs",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU15_DefaultTolerance_Pass(t *testing.T) {
	// No tolerance declared: default is 1e-9 relative, easily satisfied by
	// mass's exact/high-precision constants.
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomePass)
}

func TestCheckMU15_CanonicalUnitItself_RoundTripsExactly(t *testing.T) {
	// A value already in its canonical unit round-trips through the
	// identity transform exactly, even at tolerance "0".
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")
	decl = decl.WithTolerance(mustParse(t, "0"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "kg",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomePass)
}

// TestCheckMU15_RoundTripOverflow_Indeterminate exercises roundTrip's
// error path (an offset-addition step in Unit.ToCanonical overflowing
// apd's exponent range) -- see conversion.go's roundTrip doc comment for
// why this and FromCanonical's own overflow collapse into one branch.
func TestCheckMU15_RoundTripOverflow_Indeterminate(t *testing.T) {
	decl, err := mustDimension(t, field.NewQuantityDeclaration(), "temperature").
		WithCanonicalUnit("K")
	if err != nil {
		t.Fatalf("WithCanonicalUnit unexpected error: %v", err)
	}
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "9e99999"),
		EmbeddedUnit:    "°C",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU15_ToleranceComparisonOverflow_Indeterminate exercises
// exceedsTolerance's error path: an astronomically large declared
// tolerance whose product with abs(value) overflows the supported
// exponent range. Nothing in SPEC-MU §2.4.2 bounds how large a declared
// tolerance may be.
func TestCheckMU15_ToleranceComparisonOverflow_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")
	decl = decl.WithTolerance(mustParse(t, "9e100000"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "20"),
		EmbeddedUnit:    "kg",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU15_ToleranceComparisonSubOverflow_Indeterminate isolates
// exceedsTolerance's Sub step overflowing (as opposed to its Mul step,
// covered by TestCheckMU15_ToleranceComparisonOverflow_Indeterminate
// above): a Fahrenheit value large enough that the round trip itself
// succeeds, but subtracting the round-tripped result from the original
// does not.
func TestCheckMU15_ToleranceComparisonSubOverflow_Indeterminate(t *testing.T) {
	decl, err := mustDimension(t, field.NewQuantityDeclaration(), "temperature").
		WithCanonicalUnit("K")
	if err != nil {
		t.Fatalf("WithCanonicalUnit unexpected error: %v", err)
	}
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "-9e99999"),
		EmbeddedUnit:    "°F",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}
