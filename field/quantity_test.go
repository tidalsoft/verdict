package field

import (
	"testing"

	"github.com/tidalsoft/verdict/decimal"
)

var _ Declaration = QuantityDeclaration{}

func TestQuantityDeclaration_ZeroValue(t *testing.T) {
	d := NewQuantityDeclaration()
	if d.Kind() != KindQuantity {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindQuantity)
	}
	if _, ok := d.Dimension(); ok {
		t.Fatal("Dimension() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.UnitField(); ok {
		t.Fatal("UnitField() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.CanonicalUnit(); ok {
		t.Fatal("CanonicalUnit() on fresh declaration: ok = true, want false")
	}
	if d.UnitRequired() {
		t.Fatal("UnitRequired() on fresh declaration = true, want false")
	}
	if _, ok := d.Min(); ok {
		t.Fatal("Min() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.Max(); ok {
		t.Fatal("Max() on fresh declaration: ok = true, want false")
	}
	if d.ExclusiveMin() {
		t.Fatal("ExclusiveMin() on fresh declaration = true, want false")
	}
	if d.ExclusiveMax() {
		t.Fatal("ExclusiveMax() on fresh declaration = true, want false")
	}
	if _, ok := d.Tolerance(); ok {
		t.Fatal("Tolerance() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestQuantityDeclaration_WithDimension(t *testing.T) {
	d, err := NewQuantityDeclaration().WithDimension("mass")
	if err != nil {
		t.Fatalf("WithDimension: unexpected error: %v", err)
	}
	got, ok := d.Dimension()
	if !ok || got != "mass" {
		t.Fatalf("Dimension() = (%q, %v), want (%q, true)", got, ok, "mass")
	}
}

func TestQuantityDeclaration_WithDimension_Empty(t *testing.T) {
	if _, err := NewQuantityDeclaration().WithDimension(""); err == nil {
		t.Fatal("WithDimension(\"\"): expected error, got nil")
	}
}

// TestQuantityDeclaration_WithDimension_Unrecognised is the regression for
// the defect SPEC-MU §2.2 forbids: an unrecognised dimension string (not
// merely empty) must be rejected at construction, the same way an invalid
// Scale/Sign/Domain already is, rather than reaching MU-04 at evaluation
// and manufacturing a FAIL against a value that contradicts nothing.
// "weight" is deliberately plausible-sounding text, not an obvious typo,
// and "Mass" pins that matching is exact (no case-folding) against
// tables.ParseDimension's own closed set.
func TestQuantityDeclaration_WithDimension_Unrecognised(t *testing.T) {
	cases := []string{"weight", "Mass", "digital_storage", "currency_per_unit", "kg"}
	for _, dim := range cases {
		t.Run(dim, func(t *testing.T) {
			if _, err := NewQuantityDeclaration().WithDimension(dim); err == nil {
				t.Fatalf("WithDimension(%q): expected error, got nil", dim)
			}
		})
	}
}

// TestQuantityDeclaration_WithDimension_SpecSpelling pins SPEC-MU §4's own
// "Supported dimensions" spelling for the two tokens that differ from this
// package's internal one (tables.Dimension.String()): "digital storage" (a
// space) and "currency-per-unit" (a hyphen), where String() renders
// "digital_storage" and "currency_per_unit". A ruleset spelling them the
// spec's way must be accepted, and Dimension() reports back the internal
// spelling MU-04 compares against (mu.checkMU04), not the string the
// ruleset supplied -- see WithDimension's own doc comment.
func TestQuantityDeclaration_WithDimension_SpecSpelling(t *testing.T) {
	cases := []struct {
		specSpelling string
		wantStored   string
	}{
		{"digital storage", "digital_storage"},
		{"currency-per-unit", "currency_per_unit"},
	}
	for _, tc := range cases {
		t.Run(tc.specSpelling, func(t *testing.T) {
			d, err := NewQuantityDeclaration().WithDimension(tc.specSpelling)
			if err != nil {
				t.Fatalf("WithDimension(%q): unexpected error: %v", tc.specSpelling, err)
			}
			got, ok := d.Dimension()
			if !ok || got != tc.wantStored {
				t.Fatalf("Dimension() = (%q, %v), want (%q, true)", got, ok, tc.wantStored)
			}
		})
	}
}

func TestQuantityDeclaration_WithUnitField(t *testing.T) {
	d, err := NewQuantityDeclaration().WithUnitField("arguments.weight_unit")
	if err != nil {
		t.Fatalf("WithUnitField: unexpected error: %v", err)
	}
	got, ok := d.UnitField()
	if !ok || got != "arguments.weight_unit" {
		t.Fatalf("UnitField() = (%q, %v), want (%q, true)", got, ok, "arguments.weight_unit")
	}
}

func TestQuantityDeclaration_WithUnitField_Empty(t *testing.T) {
	if _, err := NewQuantityDeclaration().WithUnitField(""); err == nil {
		t.Fatal("WithUnitField(\"\"): expected error, got nil")
	}
}

func TestQuantityDeclaration_WithCanonicalUnit(t *testing.T) {
	d, err := NewQuantityDeclaration().WithCanonicalUnit("kg")
	if err != nil {
		t.Fatalf("WithCanonicalUnit: unexpected error: %v", err)
	}
	got, ok := d.CanonicalUnit()
	if !ok || got != "kg" {
		t.Fatalf("CanonicalUnit() = (%q, %v), want (%q, true)", got, ok, "kg")
	}
}

func TestQuantityDeclaration_WithCanonicalUnit_Empty(t *testing.T) {
	if _, err := NewQuantityDeclaration().WithCanonicalUnit(""); err == nil {
		t.Fatal("WithCanonicalUnit(\"\"): expected error, got nil")
	}
}

func TestQuantityDeclaration_WithUnitRequired(t *testing.T) {
	d := NewQuantityDeclaration()
	if d.UnitRequired() {
		t.Fatal("UnitRequired() before WithUnitRequired = true, want false")
	}
	d = d.WithUnitRequired()
	if !d.UnitRequired() {
		t.Fatal("UnitRequired() after WithUnitRequired = false, want true")
	}
}

func TestQuantityDeclaration_MinMaxAndTolerance(t *testing.T) {
	min, err := decimal.Parse("0")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	max, err := decimal.Parse("1000")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	tolerance, err := decimal.Parse("0.000000001")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}

	d := NewQuantityDeclaration().WithMin(min).WithMax(max).WithTolerance(tolerance)

	gotMin, ok := d.Min()
	if !ok || gotMin.Compare(min) != 0 {
		t.Fatalf("Min() = (%v, %v), want (%v, true)", gotMin, ok, min)
	}
	gotMax, ok := d.Max()
	if !ok || gotMax.Compare(max) != 0 {
		t.Fatalf("Max() = (%v, %v), want (%v, true)", gotMax, ok, max)
	}
	gotTolerance, ok := d.Tolerance()
	if !ok || gotTolerance.Compare(tolerance) != 0 {
		t.Fatalf("Tolerance() = (%v, %v), want (%v, true)", gotTolerance, ok, tolerance)
	}
}

func TestQuantityDeclaration_ExclusiveMinMax(t *testing.T) {
	d := NewQuantityDeclaration()
	if d.ExclusiveMin() || d.ExclusiveMax() {
		t.Fatal("ExclusiveMin()/ExclusiveMax() before declaration = true, want false")
	}
	d = d.WithExclusiveMin().WithExclusiveMax()
	if !d.ExclusiveMin() {
		t.Fatal("ExclusiveMin() after WithExclusiveMin = false, want true")
	}
	if !d.ExclusiveMax() {
		t.Fatal("ExclusiveMax() after WithExclusiveMax = false, want true")
	}
}

func TestQuantityDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewQuantityDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestQuantityDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewQuantityDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
