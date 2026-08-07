package field

import "testing"

var _ Declaration = DecimalDeclaration{}

func TestDecimalDeclaration_ZeroValue(t *testing.T) {
	d := NewDecimalDeclaration()
	if d.Kind() != KindDecimal {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindDecimal)
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestDecimalDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewDecimalDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestDecimalDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewDecimalDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
