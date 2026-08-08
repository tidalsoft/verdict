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
	// field.QuantityDeclaration's Max is reserved for MU-15, not this
	// check's range bound -- there is no min/max to consult here.
	in := Input{
		Field:    "arguments.weight",
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
		Vals:     map[string]string{},
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
		Vals:     map[string]string{"arguments.currency": "ZZZ"},
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
		Vals:     map[string]string{"arguments.currency": "XAU"},
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
	vals := map[string]string{"arguments.currency": "USD"}
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
		Vals:     map[string]string{"arguments.currency": "USD"},
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
		Vals:     map[string]string{"arguments.currency": "USD"},
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
	vals := map[string]string{"arguments.currency": "JPY"}
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
		Vals:     map[string]string{"arguments.currency": "USD"},
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
		Vals:     map[string]string{"arguments.currency": "USD"},
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
		Vals:     map[string]string{"arguments.currency": "USD"},
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
