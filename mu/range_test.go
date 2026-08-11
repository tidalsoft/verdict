package mu

import (
	"strings"
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

func TestCheckMU07_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: field.Registry{},
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-07" {
		t.Errorf("CheckID() = %q, want MU-07", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_QuantityDeclaration_Indeterminate(t *testing.T) {
	// No min/max declared at all on this quantity field -- the "Requires
	// min and/or max" gate every MU-07 branch shares.
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, field.NewQuantityDeclaration()),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_NoBoundsDeclared_Indeterminate(t *testing.T) {
	decl := mustScale(t, field.NewMoneyDeclaration(), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "49.99"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
			res, err := checkMU07(in)
			if err != nil {
				t.Fatalf("checkMU07 unexpected error: %v", err)
			}
			if res.Outcome() != tc.want {
				t.Errorf("Outcome() = %v, want %v", res.Outcome(), tc.want)
			}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_MinorUnits_CurrencyFieldSiblingAbsent_Indeterminate(t *testing.T) {
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMin(mustParse(t, "10")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "1000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{}),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
			res, err := checkMU07(in)
			if err != nil {
				t.Fatalf("checkMU07 unexpected error: %v", err)
			}
			if res.Outcome() != tc.want {
				t.Errorf("value %s: Outcome() = %v, want %v", tc.value, res.Outcome(), tc.want)
			}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}

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
	res2, err := checkMU07(in2)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res2.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res2.Outcome())
	}
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
	res, err := checkMU07(below)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}

	atBound := Input{Field: "arguments.amount", Value: mustParse(t, "500"), Registry: registry, Vals: vals, Tables: tbl}
	res2, err := checkMU07(atBound)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res2.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res2.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v (want no error, want INDETERMINATE)", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v (want no error, want INDETERMINATE)", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_MinorUnits_EvaluateDoesNotAbortSiblingChecks(t *testing.T) {
	// End-to-end through Evaluate (not just checkMU07 directly): a scaling
	// failure on MU-07 must not discard MU-01/MU-02/MU-03/MU-06/MU-14's
	// results for the same field.
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
	if len(results) != 6 {
		t.Fatalf("Evaluate returned %d results, want 6 (one per money check)", len(results))
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

func TestCheckMU07_Decimal_NoBounds_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5"),
		Registry: mustRegistry(t, field.NewDecimalDeclaration()),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Vector_68(t *testing.T) {
	// Vector 68: decimal, max "10", exclusive_max | 10 | FAIL | MU-07
	// (exclusive bound)
	decl := field.NewDecimalDeclaration().WithMax(mustParse(t, "10")).WithExclusiveMax()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "10"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU07_Decimal_WithinBounds_Pass(t *testing.T) {
	decl := field.NewDecimalDeclaration().WithMin(mustParse(t, "0")).WithMax(mustParse(t, "10"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

// ---- MU-07: percentage branch ----

func TestCheckMU07_Vector_66(t *testing.T) {
	// Vector 66: percentage, max "0.5" (no domain) | 0.25 | INDETERMINATE
	// | MU-07
	decl := field.NewPercentageDeclaration().WithMax(mustParse(t, "0.5"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.25"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Percentage_NoBounds_Indeterminate(t *testing.T) {
	decl, err := field.NewPercentageDeclaration().WithDomain(field.DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain unexpected error: %v", err)
	}
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.25"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

// ---- MU-07: quantity branch ----

func TestCheckMU07_Vector_67(t *testing.T) {
	// Vector 67: quantity, mass, canonical_unit kg, max "10" | 12 |
	// INDETERMINATE | MU-07 (value's unit unresolvable)
	decl := mustCanonicalUnit(t, mustDimension(t, field.NewQuantityDeclaration(), "mass"), "kg").WithMax(mustParse(t, "10"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, decl),
		Tables:   unitTables(),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Vector_84(t *testing.T) {
	// Vector 84: quantity, mass, canonical_unit kg, max "10" | "50 lb" |
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU07_Vector_98(t *testing.T) {
	// Vector 98: quantity, mass, unit_field, max "10", no canonical_unit |
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Vector_105(t *testing.T) {
	// Vector 105: quantity, mass, canonical_unit kg, unit_field, max "10"
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Vector_119(t *testing.T) {
	// Vector 119: quantity, mass, canonical_unit kg, max "10" | "12 m" |
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

// TestCheckMU07_TimestampKind_Indeterminate exercises checkMU07's default
// switch arm: a declared kind (timestamp) this check does not apply to at
// all.
func TestCheckMU07_TimestampKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "12"),
		Registry: mustRegistry(t, field.NewTimestampDeclaration()),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
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
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

// ---- MU-07 money branch: explicit vectors 60-65 ----

func TestCheckMU07_Vector_60(t *testing.T) {
	// Vector 60: money, minor_units, USD, max "100.00" | 5000 | PASS |
	// MU-07 (bounds in major units)
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5000"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU07_Vector_61(t *testing.T) {
	// Vector 61: money, minor_units, USD, max "100.00" | 10001 | FAIL |
	// MU-07
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "10001"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU07_Vector_62(t *testing.T) {
	// Vector 62: money, minor_units, JPY, max "100" | 101 | FAIL | MU-07
	// (exponent 0) -- catches an implementation that hardcodes exponent 2.
	decl := mustCurrencyField(t, mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100")), field.ScaleMinorUnits))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "101"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.currency": "JPY"}),
		Tables:   Tables{ISO4217: tables.NewISO4217Table()},
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU07_Vector_63(t *testing.T) {
	// Vector 63: money, major_units, max "100.00", no currency_field |
	// 50.00 | PASS | MU-07 (no exponent read)
	decl := mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMajorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50.00"),
		Registry: mustRegistry(t, decl),
		// Deliberately no Vals, no Tables.
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU07_Vector_64(t *testing.T) {
	// Vector 64: money, minor_units, max "100.00", no currency_field |
	// 5000 | INDETERMINATE | MU-07 (currency unresolvable)
	decl := mustScale(t, field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00")), field.ScaleMinorUnits)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "5000"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU07_Vector_65(t *testing.T) {
	// Vector 65: money (no scale), max "100.00" | 50 | INDETERMINATE |
	// MU-07
	decl := field.NewMoneyDeclaration().WithMax(mustParse(t, "100.00"))
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "50"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU07(in)
	if err != nil {
		t.Fatalf("checkMU07 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
