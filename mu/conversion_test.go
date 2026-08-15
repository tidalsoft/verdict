package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

func wantMU15(t *testing.T, in Input, wantOutcome verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU15(in)
	if err != nil {
		t.Fatalf("checkMU15 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU15 applicable = false, want true")
	}
	if res.CheckID() != "MU-15" {
		t.Errorf("CheckID() = %q, want MU-15", res.CheckID())
	}
	if res.Class() != verdict.ClassD {
		t.Errorf("Class() = %v, want ClassD", res.Class())
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

// wantMU15NotApplicable asserts checkMU15 reports applicable == false for
// in: the field's kind is outside MU-15's Applies to set, or the field
// never declared canonical_unit -- MU-15's own §2.5.1 gate. MU-15's warn
// default severity governs the Result it builds once applicable; it has no
// bearing on whether one is built at all (see checkMU15's own doc comment).
func wantMU15NotApplicable(t *testing.T, in Input) {
	t.Helper()
	_, applicable, err := checkMU15(in)
	if err != nil {
		t.Fatalf("checkMU15 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU15 applicable = true, want false")
	}
}

func TestCheckMU15_MU_V83(t *testing.T) {
	// MU-V83: quantity, mass, canonical_unit kg, max "10" | "12 lb" |
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

func TestCheckMU15_MU_V85(t *testing.T) {
	// MU-V85: quantity, mass, canonical_unit kg, max "10" | 12 |
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

func TestCheckMU15_MU_V97(t *testing.T) {
	// MU-V97: quantity, temperature, canonical_unit K, tolerance "0" |
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

func TestCheckMU15_MU_V107(t *testing.T) {
	// MU-V107: quantity, mass, canonical_unit kg, unit_field | "12 lb",
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

func TestCheckMU15_MU_V120(t *testing.T) {
	// MU-V120: quantity, mass, canonical_unit kg | "12 m" |
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

func TestCheckMU15_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{Field: "arguments.amount", Registry: field.Registry{}}
	wantMU15NotApplicable(t, in)
}

func TestCheckMU15_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU15NotApplicable(t, in)
}

func TestCheckMU15_NoCanonicalUnit_NotApplicable(t *testing.T) {
	// SPEC-MU §2.5.1: canonical_unit is MU-15's gate -- its absence is not
	// applicable, not a required-input INDETERMINATE.
	decl := mustDimension(t, field.NewQuantityDeclaration(), "mass") // no canonical_unit
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU15NotApplicable(t, in)
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
	// identity transform exactly, even at tolerance "0". kg is also
	// mass's registry canonical, so this alone does not distinguish a
	// correct round trip through the *declared* canonical_unit from one
	// that (incorrectly) always routes through the registry canonical --
	// see TestCheckMU15_CanonicalUnitNotRegistryCanonical_IdentityRoundTrip
	// below for the case that does.
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

// TestCheckMU15_CanonicalUnitNotRegistryCanonical_IdentityRoundTrip is a
// regression test: canonical_unit is declared as lb -- not mass's registry
// canonical, kg -- and the value already arrives in lb. Because the
// declared canonical_unit and the value's own unit are literally the same
// unit, no conversion is actually occurring, and the round trip must be
// exact -- PASS even at tolerance "0". lb's own registry factors (its
// kg-per-lb scale and lb-per-kg scale) are independently truncated
// decimal approximations that are not exact reciprocals of each other
// (see tables.Unit's doc comment), so a defect that routes "lb to lb"
// through the registry's kg canonical anyway -- lb.ToCanonical then
// lb.FromCanonical, applying that imprecision where no conversion was
// needed -- wrongly FAILs.
func TestCheckMU15_CanonicalUnitNotRegistryCanonical_IdentityRoundTrip(t *testing.T) {
	decl, err := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithCanonicalUnit("lb")
	if err != nil {
		t.Fatalf("WithCanonicalUnit unexpected error: %v", err)
	}
	decl = decl.WithTolerance(mustParse(t, "0"))
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

func TestCheckMU15_ValueNotCoercible_Indeterminate(t *testing.T) {
	// SPEC-MU §2.6.3: MU-15 is value-dependent, so once its own gate
	// (canonical_unit declared) is satisfied and the canonical unit
	// resolves, a value the coercion gate could not read is INDETERMINATE
	// before the round trip ever runs -- Value must not be read once
	// ValueCoercionFailed is true.
	decl, err := mustDimension(t, field.NewQuantityDeclaration(), "mass").WithCanonicalUnit("kg")
	if err != nil {
		t.Fatalf("WithCanonicalUnit unexpected error: %v", err)
	}
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		EmbeddedUnit:        "lb",
		HasEmbeddedUnit:     true,
		Registry:            mustRegistry(t, decl),
		Tables:              unitTables(),
	}
	wantMU15(t, in, verdict.OutcomeIndeterminate)
}
