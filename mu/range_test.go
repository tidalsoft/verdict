package mu

import (
	"strings"
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// wantMU07 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU07: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-07's only severity), and Outcome. One helper serves
// all four of checkMU07's per-kind branches (money, decimal, percentage,
// quantity): CheckID/Class/Severity are identical across every branch.
func wantMU07(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU07 applicable = false, want true")
	}
	if res.CheckID() != "MU-07" {
		t.Errorf("CheckID() = %q, want MU-07", res.CheckID())
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

func TestCheckMU07_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU07_QuantityDeclaration_NotApplicable(t *testing.T) {
	// No min/max declared at all on this quantity field -- the "Requires
	// min and/or max" gate every MU-07 branch shares (§2.5.2: unbounded is
	// a coherent declaration, not a gap).
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, field.NewQuantityDeclaration()),
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (no min/max declared)")
	}
}

func TestCheckMU07_NoBoundsDeclared_NotApplicable(t *testing.T) {
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (no min/max declared)")
	}
}

func TestCheckMU07_NoScaleDeclared_Indeterminate(t *testing.T) {
	// Bounds are declared but the field's own scale is not: without
	// knowing whether the value is major or minor units, there is no safe
	// way to compare it against a major-units bound.
	decl := field.NewMoneyDeclaration().WithMin(mustParse(t, "0"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU07_MajorUnits_NoCurrencyNeeded proves this directly:
// a scale: major_units field with bounds produces a real verdict with no
// currency_field, no Vals, and a zero-value Tables -- min/max (always
// major units) already match the value's own unit, so no currency lookup
// happens at all.
func TestCheckMU07_MajorUnits_NoCurrencyNeeded(t *testing.T) {
	cases := []struct {
		name  string
		decl  field.MoneyDeclaration
		value string
		want  verdict.Outcome
	}{
		{
			name:  "below min -> FAIL",
			decl:  field.NewMoneyDeclaration().WithMin(mustParse(t, "50")),
			value: "49.99",
			want:  verdict.OutcomeFail,
		},
		{
			name:  "above max -> FAIL",
			decl:  field.NewMoneyDeclaration().WithMax(mustParse(t, "50")),
			value: "50.01",
			want:  verdict.OutcomeFail,
		},
		{
			name:  "within bounds -> PASS",
			decl:  field.NewMoneyDeclaration().WithMin(mustParse(t, "0")).WithMax(mustParse(t, "100")),
			value: "49.99",
			want:  verdict.OutcomePass,
		},
		{
			name:  "inclusive min equality -> PASS",
			decl:  field.NewMoneyDeclaration().WithMin(mustParse(t, "49.99")),
			value: "49.99",
			want:  verdict.OutcomePass,
		},
		{
			name:  "exclusive min equality -> FAIL",
			decl:  field.NewMoneyDeclaration().WithMin(mustParse(t, "49.99")).WithExclusiveMin(),
			value: "49.99",
			want:  verdict.OutcomeFail,
		},
		{
			name:  "inclusive max equality -> PASS",
			decl:  field.NewMoneyDeclaration().WithMax(mustParse(t, "49.99")),
			value: "49.99",
			want:  verdict.OutcomePass,
		},
		{
			name:  "exclusive max equality -> FAIL",
			decl:  field.NewMoneyDeclaration().WithMax(mustParse(t, "49.99")).WithExclusiveMax(),
			value: "49.99",
			want:  verdict.OutcomeFail,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decl := mustScale(t, tc.decl, field.ScaleMajorUnits)
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: mustRegistry(t, decl),
				// Deliberately no Vals, no Tables: major_units needs
				// neither.
			}
			wantMU07(t, in, tc.want)
		})
	}
}

func TestCheckMU07_MinorUnits_NoCurrencyField_Indeterminate(t *testing.T) {
	decl := mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "10")), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MinorUnits_CurrencyFieldSiblingAbsent_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "10")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{}),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MinorUnits_CurrencyUnresolvable_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "10")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "ZZZ"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MinorUnits_CurrencyNoExponent_Indeterminate(t *testing.T) {
	// XAU (gold) has no ISO 4217 minor-unit exponent at all -- nothing to
	// scale the major-units bound by.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "10")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "XAU"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

// TestCheckMU07_MinorUnits_USD_WorkedExample is the worked example for
// major-units bounds: a scale: minor_units USD field with min: 10, max: 100
// (declared in major units -- $10-$100) compared against values supplied
// in minor units (cents). $10 -> 1000 cents, $100 -> 10000 cents.
func TestCheckMU07_MinorUnits_USD_WorkedExample(t *testing.T) {
	decl := mustCurrencyField(t, mustScale(t,
		field.NewMoneyDeclaration().WithMin(mustParse(t, "10")).WithMax(mustParse(t, "100")),
		field.ScaleMinorUnits))
	registry := mustRegistry(t, decl)
	vals := stringVals(map[string]string{"arguments.currency": "USD"})
	tbl := Tables{ISO4217: tables.NewISO4217Table()}

	cases := []struct {
		name  string
		value string
		want  verdict.Outcome
	}{
		{"999 cents ($9.99) below $10 min -> FAIL", "999", verdict.OutcomeFail},
		{"1000 cents (exactly $10 min, inclusive) -> PASS", "1000", verdict.OutcomePass},
		// 5000 minor units is $50.00 -- squarely inside $10-$100. A
		// same-units comparison (the earlier scale-both-sides no-op) would have compared
		// raw 5000 against raw bounds 10/100 and wrongly FAILed as "above
		// max". Scaling the bounds to minor units (1000-10000) is what
		// makes this PASS correctly.
		{"5000 cents ($50.00) within $10-$100 -> PASS", "5000", verdict.OutcomePass},
		{"10000 cents (exactly $100 max, inclusive) -> PASS", "10000", verdict.OutcomePass},
		{"10001 cents ($100.01) above $100 max -> FAIL", "10001", verdict.OutcomeFail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: registry,
				Vals:     vals,
				Tables:   tbl,
			}
			wantMU07(t, in, tc.want)
		})
	}
}

func TestCheckMU07_MinorUnits_ExclusiveBoundary(t *testing.T) {
	minDecl := mustCurrencyField(t, mustScale(t,
		field.NewMoneyDeclaration().WithMin(mustParse(t, "10")).WithExclusiveMin(),
		field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"), // exactly $10, the excluded boundary
		Registry: mustRegistry(t, minDecl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeFail)

	maxDecl := mustCurrencyField(t, mustScale(t,
		field.NewMoneyDeclaration().WithMax(mustParse(t, "100")).WithExclusiveMax(),
		field.ScaleMinorUnits))
	in2 := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "10000"), // exactly $100, the excluded boundary
		Registry: mustRegistry(t, maxDecl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in2, verdict.OutcomeFail)
}

func TestCheckMU07_MinorUnits_JPY_ZeroExponent(t *testing.T) {
	// JPY's minor-unit exponent is 0: scaling a major-units bound by 10^0
	// is a no-op, so "minor units" and "major units" coincide numerically
	// for this currency.
	decl := mustCurrencyField(t, mustScale(t,
		field.NewMoneyDeclaration().WithMin(mustParse(t, "500")),
		field.ScaleMinorUnits))
	registry := mustRegistry(t, decl)
	vals := stringVals(map[string]string{"arguments.currency": "JPY"})
	tbl := Tables{ISO4217: tables.NewISO4217Table()}

	below := Input{Field: "arguments.amount", Value: mustParse(t, "499"), Registry: registry, Vals: vals, Tables: tbl}
	wantMU07(t, below, verdict.OutcomeFail)

	atBound := Input{Field: "arguments.amount", Value: mustParse(t, "500"), Registry: registry, Vals: vals, Tables: tbl}
	wantMU07(t, atBound, verdict.OutcomePass)
}

// TestCheckMU07_MinorUnits_BoundScaleOverflow_Indeterminate replaces the
// old TestCheckMU07_BoundedError_Propagates: a bound overflowing
// ScaleByExponent must produce INDETERMINATE for MU-07 alone, never an
// error -- an OnFunc error aborts evaluateChecks' whole batch (SPEC-MU
// §2.4 does not permit one check's failure to discard every other check's
// result).
func TestCheckMU07_MinorUnits_BoundScaleOverflow_Indeterminate(t *testing.T) {
	huge := mustParse(t, "999"+strings.Repeat("0", 99997))
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(huge), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MinorUnits_MaxBoundScaleOverflow_Indeterminate(t *testing.T) {
	huge := mustParse(t, "999"+strings.Repeat("0", 99997))
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(huge), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MinorUnits_EvaluateDoesNotAbortSiblingChecks(t *testing.T) {
	// End-to-end through Evaluate (not just checkMU07 directly): a scaling
	// failure on MU-07 must not discard MU-01/MU-02/MU-06's results for the
	// same field. MU-03 and MU-14 are not applicable to this declaration
	// (no target_currency_field; scale is minor_units, not major_units), so
	// they contribute no entry at all -- four results, not six.
	huge := mustParse(t, "999"+strings.Repeat("0", 99997))
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(huge), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "4999"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	results, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate unexpected error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("Evaluate returned %d results, want 4 (MU-01, MU-02, MU-06, MU-07 -- MU-03 and MU-14 not applicable)", len(results))
	}
	var sawMU07 bool
	for _, res := range results {
		if res.CheckID() == "MU-07" {
			sawMU07 = true
			if res.Outcome() != verdict.OutcomeIndeterminate {
				t.Errorf("MU-07 Outcome() = %v, want INDETERMINATE", res.Outcome())
			}
		}
	}
	if !sawMU07 {
		t.Fatal("Evaluate results missing MU-07")
	}
}

// ---- MU-07: decimal branch ----

func TestCheckMU07_Decimal_NoBounds_NotApplicable(t *testing.T) {
	// SPEC-MU §2.5.2: neither min nor max declared means the field is
	// simply unbounded -- not applicable, not INDETERMINATE.
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5"),
		Registry: mustRegistry(t, field.NewDecimalDeclaration()),
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (no min/max declared)")
	}
}

func TestCheckMU07_MU_V68(t *testing.T) {
	// MU-V68: decimal, max "10", exclusive_max | 10 | FAIL | MU-07
	// (exclusive bound)
	decl := field.NewDecimalDeclaration().WithMax(mustParse(t, "10")).WithExclusiveMax()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "10"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

func TestCheckMU07_Decimal_WithinBounds_Pass(t *testing.T) {
	decl := field.NewDecimalDeclaration().WithMin(mustParse(t, "0")).WithMax(mustParse(t, "10"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomePass)
}

// ---- MU-07: percentage branch ----

func TestCheckMU07_MU_V66(t *testing.T) {
	// MU-V66: percentage, max "0.5" (no domain) | 0.25 | INDETERMINATE
	// | MU-07
	decl := field.NewPercentageDeclaration().WithMax(mustParse(t, "0.5"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.25"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_Percentage_NoBounds_NotApplicable(t *testing.T) {
	// No min/max declared -- the gate, not a required-input gap.
	decl, err := field.NewPercentageDeclaration().WithDomain(field.DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain unexpected error: %v", err)
	}
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.25"),
		Registry: mustRegistry(t, decl),
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (no min/max declared)")
	}
}

func TestCheckMU07_Percentage_WithDomain_Pass(t *testing.T) {
	decl, err := field.NewPercentageDeclaration().WithDomain(field.DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain unexpected error: %v", err)
	}
	decl = decl.WithMin(mustParse(t, "0")).WithMax(mustParse(t, "1"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.5"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomePass)
}

// ---- MU-07: quantity branch ----

func TestCheckMU07_MU_V67(t *testing.T) {
	// MU-V67: quantity, mass, canonical_unit kg, max "10" | 12 |
	// INDETERMINATE | MU-07 (value's unit unresolvable)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Tables:   unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MU_V84(t *testing.T) {
	// MU-V84: quantity, mass, canonical_unit kg, max "10" | "50 lb" |
	// FAIL | MU-07 (22.68 kg exceeds the bound)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "50"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

func TestCheckMU07_MU_V98(t *testing.T) {
	// MU-V98: quantity, mass, unit_field, max "10", no canonical_unit |
	// "12 lb" | INDETERMINATE | MU-07 (bounds have no stated units)
	decl := mustUnitField(t, mustDimension(t, field.NewQuantityDeclaration(), "mass")).WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MU_V105(t *testing.T) {
	// MU-V105: quantity, mass, canonical_unit kg, unit_field, max "10"
	// | "12 lb", unit field "kg" | INDETERMINATE | MU-07 (unit_conflict)
	decl := mustUnitField(t,
		mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg")).WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Vals:            map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MU_V119(t *testing.T) {
	// MU-V119: quantity, mass, canonical_unit kg, max "10" | "12 m" |
	// INDETERMINATE | MU-07 (unit recognised, wrong dimension)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "m",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_Quantity_CanonicalUnitNotInRegistry_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "flurbs").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "kg",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_Quantity_ValueUnitNotInRegistry_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "flurbs",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_Quantity_WithinBounds_Pass(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").
		WithMin(mustParse(t, "0")).WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomePass)
}

// TestCheckMU07Quantity_CanonicalUnitNotRegistryCanonical_BoundEnforced is a
// regression test: mass's registry canonical is kg, but canonical_unit here
// is declared as g -- a different, still-recognised unit in the same
// dimension. SPEC-MU §3 MU-07's Evaluation clause states the bound in the
// *declared* canonical_unit, so 12 g against max: "10" must FAIL (12 > 10,
// no conversion needed at all since the value already arrived in g). A
// defect that instead converted both sides through the registry's kg
// canonical -- reading the bound as already being in kg -- computed
// 12 × 0.001 = 0.012 kg against an unconverted bound of 10 and wrongly
// PASSed.
func TestCheckMU07Quantity_CanonicalUnitNotRegistryCanonical_BoundEnforced(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "g").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "g",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

// TestCheckMU07Quantity_CanonicalUnitLb_BoundEnforced is the same
// regression pinned against lb instead of g -- a unit whose registry
// to/from conversion factors are not exact reciprocals of each other
// (unlike g/kg's exact 1000), so this also guards against a fix that
// special-cases only units with exact round-trip factors. 12 lb against
// max: "10" must FAIL in lb, the declared unit; the pre-fix defect
// converted 12 lb to ~5.443 kg, compared that against an unconverted
// bound of 10, and wrongly PASSed.
func TestCheckMU07Quantity_CanonicalUnitLb_BoundEnforced(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "lb").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

// TestCheckMU07Quantity_TwoHopConversion_BoundEnforced pins the second
// conversion hop, which no other test asserts.
//
// Every other quantity test either declares canonical_unit equal to the
// value's own unit (taking convertBetweenUnits' same-symbol shortcut) or
// declares kg or K, whose FromCanonical is ×1 + 0. In both cases the hop
// from the registry canonical to the *declared* canonical is arithmetically
// the identity, so transposing it to to.ToCanonical -- or deleting it --
// passes the whole suite. Here all three units differ: the value arrives in
// lb, the registry canonical for mass is kg, and the declared canonical_unit
// is g. 12 lb is 5443.10844 g, which must FAIL max: "10".
//
// This is the same blind spot that hid the defect the two tests above now
// guard: a suite that only exercises inputs where two readings coincide
// cannot tell you which one you implemented.
func TestCheckMU07Quantity_TwoHopConversion_BoundEnforced(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "g").WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "12"),
		EmbeddedUnit:    "lb",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

// TestCheckMU07Quantity_TwoHopConversion_WithinBound is the two-hop case in
// the PASS direction, so the test above cannot be satisfied by a fix that
// simply fails everything it touches. 77 °F is 24.99999999999999999975 °C
// under the registry's truncated factors -- comfortably inside max: "30",
// and comfortably outside max: "20", which the sibling assertion pins.
func TestCheckMU07Quantity_TwoHopConversion_WithinBound(t *testing.T) {
	base := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "temperature"), "°C")
	for _, tc := range []struct {
		name string
		max  string
		want verdict.Outcome
	}{
		{"within", "30", verdict.OutcomePass},
		{"exceeded", "20", verdict.OutcomeFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decl := base.WithMax(mustParse(t, tc.max))
			in := Input{
				Field:           "arguments.amount",
				Value:           mustParse(t, "77"),
				EmbeddedUnit:    "°F",
				HasEmbeddedUnit: true,
				Registry:        mustRegistry(t, decl),
				Tables:          unitTables(),
			}
			wantMU07(t, in, tc.want)
		})
	}
}

// TestCheckMU07Quantity_SameUnitBoundary_NoSpuriousFail pins why
// convertBetweenUnits' same-symbol shortcut is load-bearing rather than an
// optimisation.
//
// lb's stored factors are not exact reciprocals: toScale is 0.45359237
// (exact, by definition of the pound) while fromScale is the 20-digit
// truncation of 1/0.45359237. Routing 12 lb through kg and back therefore
// yields 12 − 5.3e-20. Against an inclusive min of exactly 12 that is
// *below* the bound, so MU-07 would FAIL at block severity on a value equal
// to its own declared minimum -- a Class D false positive, which SPEC-MU
// §2.2 says cannot exist by construction. The shortcut returns the value
// untouched and it PASSes.
//
// The exclusive_max arm holds the boundary from the other side, so a fix
// that made the shortcut over-permissive would not slip through.
func TestCheckMU07Quantity_SameUnitBoundary_NoSpuriousFail(t *testing.T) {
	base := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "lb")
	for _, tc := range []struct {
		name string
		decl field.QuantityDeclaration
		want verdict.Outcome
	}{
		{"inclusive min, value equals bound", base.WithMin(mustParse(t, "12")), verdict.OutcomePass},
		{"exclusive max, value equals bound", base.WithMax(mustParse(t, "12")).WithExclusiveMax(), verdict.OutcomeFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:           "arguments.amount",
				Value:           mustParse(t, "12"),
				EmbeddedUnit:    "lb",
				HasEmbeddedUnit: true,
				Registry:        mustRegistry(t, tc.decl),
				Tables:          unitTables(),
			}
			wantMU07(t, in, tc.want)
		})
	}
}

// TestCheckMU07_TimestampKind_NotApplicable exercises checkMU07's default
// switch arm: a declared kind (timestamp) this check does not apply to at
// all.
func TestCheckMU07_TimestampKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, field.NewTimestampDeclaration()),
	}
	_, applicable, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU07 applicable = true, want false (wrong kind)")
	}
}

// TestCheckMU07_Quantity_ConversionOverflow_Indeterminate exercises
// checkMU07Quantity's own ToCanonical error branch: a value large enough
// that converting it into the canonical unit overflows the supported
// exponent range.
func TestCheckMU07_Quantity_ConversionOverflow_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "temperature"), "K").
		WithMax(mustParse(t, "10"))
	in := Input{
		Field:           "arguments.amount",
		Value:           mustParse(t, "9e99999"),
		EmbeddedUnit:    "°C",
		HasEmbeddedUnit: true,
		Registry:        mustRegistry(t, decl),
		Tables:          unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

// ---- MU-07 money branch: explicit vectors 60-65 ----

func TestCheckMU07_MU_V60(t *testing.T) {
	// MU-V60: money, minor_units, USD, max "100.00" | 5000 | PASS |
	// MU-07 (bounds in major units)
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomePass)
}

func TestCheckMU07_MU_V61(t *testing.T) {
	// MU-V61: money, minor_units, USD, max "100.00" | 10001 | FAIL |
	// MU-07
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "10001"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

func TestCheckMU07_MU_V62(t *testing.T) {
	// MU-V62: money, minor_units, JPY, max "100" | 101 | FAIL | MU-07
	// (exponent 0) -- catches an implementation that hardcodes exponent 2.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "101"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "JPY"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	wantMU07(t, in, verdict.OutcomeFail)
}

func TestCheckMU07_MU_V63(t *testing.T) {
	// MU-V63: money, major_units, max "100.00", no currency_field |
	// 50.00 | PASS | MU-07 (no exponent read)
	decl := mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50.00"),
		Registry: mustRegistry(t, decl),
		// Deliberately no Vals, no Tables.
	}
	wantMU07(t, in, verdict.OutcomePass)
}

func TestCheckMU07_MU_V64(t *testing.T) {
	// MU-V64: money, minor_units, max "100.00", no currency_field |
	// 5000 | INDETERMINATE | MU-07 (currency unresolvable)
	decl := mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5000"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07_MU_V65(t *testing.T) {
	// MU-V65: money (no scale), max "100.00" | 50 | INDETERMINATE |
	// MU-07
	decl := field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

// ---- MU-07: value_not_coercible, one per branch (SPEC-MU §2.6.3) ----
//
// MU-07 is value-dependent on every kind it applies to. Each of these
// exercises the coercion-gate branch in its own per-kind function
// (checkMU07Money, checkMU07Decimal, checkMU07Percentage,
// checkMU07Quantity) once that branch's own gate (min or max declared) is
// already satisfied -- confirming the value is never read once
// ValueCoercionFailed is true, per Input's own doc comment.

func TestCheckMU07Money_ValueNotCoercible_Indeterminate(t *testing.T) {
	decl := mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "0")), field.ScaleMajorUnits)
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07Decimal_ValueNotCoercible_Indeterminate(t *testing.T) {
	decl := field.NewDecimalDeclaration().WithMin(mustParse(t, "0"))
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07Percentage_ValueNotCoercible_Indeterminate(t *testing.T) {
	decl := field.NewPercentageDeclaration().WithMin(mustParse(t, "0"))
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU07Quantity_ValueNotCoercible_Indeterminate(t *testing.T) {
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").
		WithMin(mustParse(t, "0"))
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
		Tables:              unitTables(),
	}
	wantMU07(t, in, verdict.OutcomeIndeterminate)
}
