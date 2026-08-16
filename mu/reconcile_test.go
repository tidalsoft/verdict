package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU12 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU12: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-12's only severity), and Outcome.
func wantMU12(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU12(in)
	if err != nil {
		t.Fatalf("checkMU12 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU12 applicable = false, want true")
	}
	if res.CheckID() != "MU-12" {
		t.Errorf("CheckID() = %q, want MU-12", res.CheckID())
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

// mustReconcile builds a ReconcileDeclaration naming total and the fixed
// components path "arguments.line_items[*].amount" -- every call site in
// this file uses that same components path, so, per this package's
// mustCurrencyField/mustWhenEntry convention (a parameter no call site
// varies is dead flexibility, golangci-lint's unparam agrees), it is a
// constant here rather than a second parameter.
func mustReconcile(t *testing.T, total string) field.ReconcileDeclaration {
	t.Helper()
	const components = "arguments.line_items[*].amount"
	d, err := field.NewReconcileDeclaration(total, components)
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	return d
}

func elem(t *testing.T, v string) SequenceElement {
	t.Helper()
	return SequenceElement{Value: mustParse(t, v)}
}

func TestCheckMU12_MU_V74(t *testing.T) {
	// MU-V74: reconcile, tolerance 0 | items [10.10, "abc"], total 30.30 |
	// INDETERMINATE | MU-12 (uncoercible component)
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "30.30"),
		Components: []SequenceElement{
			elem(t, "10.10"),
			{CoercionFailed: true},
		},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_MU_V95(t *testing.T) {
	// MU-V95: reconcile, tolerance 0, adjustments: [arguments.tax] | items
	// [10.10, 20.20], total 30.30, arguments.tax absent | PASS | MU-12
	// (absent adjustment contributes zero)
	decl := mustReconcile(t, "arguments.total")
	decl, err := decl.WithAdjustments([]string{"arguments.tax"})
	if err != nil {
		t.Fatalf("WithAdjustments unexpected error: %v", err)
	}
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     decl,
		TotalResolved: true,
		TotalValue:    mustParse(t, "30.30"),
		Components:    []SequenceElement{elem(t, "10.10"), elem(t, "20.20")},
		Adjustments:   [][]SequenceElement{nil}, // arguments.tax absent
	}
	wantMU12(t, in, verdict.OutcomePass)
}

func TestCheckMU12_MU_V96(t *testing.T) {
	// MU-V96: reconcile, tolerance 0, components:
	// arguments.line_items[*].amount | line_items: [{amount: 10.10}, {sku:
	// "widget-2"}], total 10.10 | INDETERMINATE | MU-12 (an element
	// matched, its amount did not -- a miss)
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "10.10"),
		Components: []SequenceElement{
			elem(t, "10.10"),
			{Miss: true},
		},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_MU_V109(t *testing.T) {
	// MU-V109: reconcile, tolerance 0, total: arguments.total | items
	// [10.10, 20.20], arguments.total absent | INDETERMINATE | MU-12
	// (total does not resolve)
	in := Input{
		Field:        "arguments.total",
		HasReconcile: true,
		Reconcile:    mustReconcile(t, "arguments.total"),
		// TotalResolved deliberately left false: the total path is absent.
		Components: []SequenceElement{elem(t, "10.10"), elem(t, "20.20")},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_MU_V110(t *testing.T) {
	// MU-V110: reconcile, tolerance 0, components:
	// arguments.line_items[*].amount | line_items: [], total 0 |
	// INDETERMINATE | MU-12 (empty components match against a zero total)
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "0"),
		// Components deliberately left nil: the wildcard matched nothing.
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_MU_V114(t *testing.T) {
	// MU-V114: reconcile, tolerance 0, components:
	// arguments.line_items[*].amount, adjustments:
	// [arguments.fees[*].amount] | line_items: [{amount: 30.30}], fees:
	// [{amount: 5}, {}], total 35.30 | INDETERMINATE | MU-12 (adjustments
	// sequence carries a miss)
	decl := mustReconcile(t, "arguments.total")
	decl, err := decl.WithAdjustments([]string{"arguments.fees[*].amount"})
	if err != nil {
		t.Fatalf("WithAdjustments unexpected error: %v", err)
	}
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     decl,
		TotalResolved: true,
		TotalValue:    mustParse(t, "35.30"),
		Components:    []SequenceElement{elem(t, "30.30")},
		Adjustments: [][]SequenceElement{
			{elem(t, "5"), {Miss: true}},
		},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_MU_V115(t *testing.T) {
	// MU-V115: reconcile, tolerance 0, total: arguments.totals[*],
	// components: arguments.line_items[*].amount | totals: [30.30, 40.00],
	// line_items: [{amount: 10.10}, {amount: 20.20}] | INDETERMINATE |
	// MU-12 (total resolves to a sequence)
	in := Input{
		Field:        "arguments.totals",
		HasReconcile: true,
		Reconcile:    mustReconcile(t, "arguments.totals"),
		// TotalResolved deliberately left false: the total path itself
		// resolved to a sequence, not a single scalar.
		Components: []SequenceElement{elem(t, "10.10"), elem(t, "20.20")},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_TotalCoercionFailed_Indeterminate(t *testing.T) {
	// Not vector-tested directly, but stated in SPEC-MU §5's Indeterminate
	// when clause: "Any path the entry names -- its total, any component,
	// or any adjustment -- carries a value that is not coercible."
	in := Input{
		Field:               "arguments.total",
		HasReconcile:        true,
		Reconcile:           mustReconcile(t, "arguments.total"),
		TotalResolved:       true,
		TotalCoercionFailed: true,
		Components:          []SequenceElement{elem(t, "10.10")},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_AdjustmentCoercionFailed_Indeterminate(t *testing.T) {
	decl := mustReconcile(t, "arguments.total")
	decl, err := decl.WithAdjustments([]string{"arguments.tax"})
	if err != nil {
		t.Fatalf("WithAdjustments unexpected error: %v", err)
	}
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     decl,
		TotalResolved: true,
		TotalValue:    mustParse(t, "10.10"),
		Components:    []SequenceElement{elem(t, "10.10")},
		Adjustments:   [][]SequenceElement{{{CoercionFailed: true}}},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_DeltaExceedsTolerance_Fail(t *testing.T) {
	// The FAIL branch itself: SPEC-MU §8.3's MU-V34 pins the identical
	// arithmetic at decimal.Reconciles' own level (decimal/
	// reconcile_test.go), but no MU-12 vector runs it through checkMU12,
	// so this exercises the check's own FAIL branch directly.
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "30.31"),
		Components:    []SequenceElement{elem(t, "10.10"), elem(t, "20.20")}, // sums to 30.30
	}
	wantMU12(t, in, verdict.OutcomeFail)
}

func TestCheckMU12_ToleranceHonored(t *testing.T) {
	decl := mustReconcile(t, "arguments.total")
	decl, err := decl.WithTolerance(mustParse(t, "1"))
	if err != nil {
		t.Fatalf("WithTolerance unexpected error: %v", err)
	}
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     decl,
		TotalResolved: true,
		TotalValue:    mustParse(t, "30.31"),
		Components:    []SequenceElement{elem(t, "10.10"), elem(t, "20.20")}, // sums to 30.30
	}
	// Delta is 0.01, within the declared tolerance of 1.
	wantMU12(t, in, verdict.OutcomePass)
}

func TestCheckMU12_NotApplicable_NoReconcileEntry(t *testing.T) {
	in := Input{Field: "arguments.total"}
	_, applicable, err := checkMU12(in)
	if err != nil {
		t.Fatalf("checkMU12 unexpected error: %v", err)
	}
	if applicable {
		t.Fatal("checkMU12 applicable = true, want false (HasReconcile false)")
	}
}

func TestCheckMU12_AdjustmentsSumAcrossMultiplePaths(t *testing.T) {
	// Every declared adjustment path contributes its own resolved
	// sequence's sum -- exercises the multi-adjustment loop in checkMU12
	// beyond the single-adjustment vectors above.
	decl := mustReconcile(t, "arguments.total")
	decl, err := decl.WithAdjustments([]string{"arguments.tax", "arguments.shipping", "arguments.discount"})
	if err != nil {
		t.Fatalf("WithAdjustments unexpected error: %v", err)
	}
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     decl,
		TotalResolved: true,
		TotalValue:    mustParse(t, "112"), // 100 + 10 + 5 - 3
		Components:    []SequenceElement{elem(t, "100")},
		Adjustments: [][]SequenceElement{
			{elem(t, "10")},
			{elem(t, "5")},
			{elem(t, "-3")},
		},
	}
	wantMU12(t, in, verdict.OutcomePass)
}

func TestEvaluate_ReconcileWithoutFieldDeclaration(t *testing.T) {
	// SPEC-MU §2.3: a reconcile entry's total path is evaluated even
	// without a field.Registry entry of its own.
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "30.30"),
		Components:    []SequenceElement{elem(t, "10.10"), elem(t, "20.20")},
	}
	results, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Evaluate returned %d results, want 1", len(results))
	}
	if results[0].CheckID() != "MU-12" {
		t.Errorf("CheckID() = %q, want MU-12", results[0].CheckID())
	}
	if results[0].Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", results[0].Outcome())
	}
}

func TestCheckMU12_SumOverflow_Indeterminate(t *testing.T) {
	// decimal.Add's one failure mode -- an exponent-range overflow -- is
	// not vector-tested (every conformance vector's arithmetic stays
	// comfortably in range), but SPEC-MU §2.6 forbids letting one check's
	// arithmetic failure abort evaluation, so checkMU12 must turn it into
	// INDETERMINATE rather than propagate an error. Two components at the
	// edge of the supported exponent range, summed together, overflow
	// sumSequence's own running total.
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "0"),
		Components:    []SequenceElement{elem(t, "9e100000"), elem(t, "9e100000")},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU12_ReconcilesOverflow_Indeterminate(t *testing.T) {
	// decimal.Reconciles' own Sub call can overflow independently of
	// sumSequence's Add calls -- this pins that path specifically: the
	// component sum alone (a single huge value) never overflows, but
	// comparing it against an equally huge, oppositely-signed total does.
	in := Input{
		Field:         "arguments.total",
		HasReconcile:  true,
		Reconcile:     mustReconcile(t, "arguments.total"),
		TotalResolved: true,
		TotalValue:    mustParse(t, "-9e100000"),
		Components:    []SequenceElement{elem(t, "9e100000")},
	}
	wantMU12(t, in, verdict.OutcomeIndeterminate)
}

func TestSumSequence(t *testing.T) {
	zero := mustParse(t, "0")
	cases := []struct {
		name string
		seq  []SequenceElement
		want string
		ok   bool
	}{
		{"empty contributes nothing", nil, "0", true},
		{"sums cleanly", []SequenceElement{elem(t, "1"), elem(t, "2")}, "3", true},
		{"a miss fails", []SequenceElement{elem(t, "1"), {Miss: true}}, "", false},
		{"an uncoercible element fails", []SequenceElement{{CoercionFailed: true}}, "", false},
		{"an overflowing add fails", []SequenceElement{elem(t, "9e100000"), elem(t, "9e100000")}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := sumSequence(tc.seq, zero)
			if ok != tc.ok {
				t.Fatalf("sumSequence ok = %v, want %v", ok, tc.ok)
			}
			if ok && got.Compare(mustParse(t, tc.want)) != 0 {
				t.Errorf("sumSequence = %v, want %v", got, tc.want)
			}
		})
	}
}
