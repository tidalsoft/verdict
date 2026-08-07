package field

import (
	"testing"

	"github.com/evanisnor/gatepost/engine/decimal"
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
	if _, ok := d.Max(); ok {
		t.Fatal("Max() on fresh declaration: ok = true, want false")
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

func TestQuantityDeclaration_MaxAndTolerance(t *testing.T) {
	max, err := decimal.Parse("1000")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	tolerance, err := decimal.Parse("0.000000001")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}

	d := NewQuantityDeclaration().WithMax(max).WithTolerance(tolerance)

	gotMax, ok := d.Max()
	if !ok || gotMax.Compare(max) != 0 {
		t.Fatalf("Max() = (%v, %v), want (%v, true)", gotMax, ok, max)
	}
	gotTolerance, ok := d.Tolerance()
	if !ok || gotTolerance.Compare(tolerance) != 0 {
		t.Fatalf("Tolerance() = (%v, %v), want (%v, true)", gotTolerance, ok, tolerance)
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
