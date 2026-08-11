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
	if _, ok := d.Sign(); ok {
		t.Fatal("Sign() on fresh declaration: ok = true, want false")
	}
	if got := d.SignWhen(); got != nil {
		t.Fatalf("SignWhen() on fresh declaration = %v, want nil", got)
	}
	if d.Nonzero() {
		t.Fatal("Nonzero() on fresh declaration = true, want false")
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

func TestDecimalDeclaration_WithSign(t *testing.T) {
	d, err := NewDecimalDeclaration().WithSign(SignPositive)
	if err != nil {
		t.Fatalf("WithSign: unexpected error: %v", err)
	}
	got, ok := d.Sign()
	if !ok || got != SignPositive {
		t.Fatalf("Sign() = (%v, %v), want (%v, true)", got, ok, SignPositive)
	}
}

func TestDecimalDeclaration_WithSign_Invalid(t *testing.T) {
	if _, err := NewDecimalDeclaration().WithSign(Sign(99)); err == nil {
		t.Fatal("WithSign(invalid): expected error, got nil")
	}
}

func TestDecimalDeclaration_WithSignWhen(t *testing.T) {
	entry := mustWhenEntry(t, NewStringValue("refund"))
	cond, err := NewConditionalSign([]WhenEntry{entry}, SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign: unexpected error: %v", err)
	}
	conds := []ConditionalSign{cond}

	d, err := NewDecimalDeclaration().WithSignWhen(conds)
	if err != nil {
		t.Fatalf("WithSignWhen: unexpected error: %v", err)
	}

	// Defensive copy on the way in.
	conds[0] = ConditionalSign{}

	got := d.SignWhen()
	if len(got) != 1 || got[0].Sign() != SignNegative {
		t.Fatalf("SignWhen() = %+v, want one entry with sign negative", got)
	}

	// Defensive copy on the way out.
	got[0] = ConditionalSign{}
	again := d.SignWhen()
	if again[0].Sign() != SignNegative {
		t.Fatalf("SignWhen() after mutating a prior result: Sign() = %v, want unchanged %v", again[0].Sign(), SignNegative)
	}
}

func TestDecimalDeclaration_WithSignWhen_Empty(t *testing.T) {
	if _, err := NewDecimalDeclaration().WithSignWhen(nil); err == nil {
		t.Fatal("WithSignWhen(nil): expected error, got nil")
	}
	if _, err := NewDecimalDeclaration().WithSignWhen([]ConditionalSign{}); err == nil {
		t.Fatal("WithSignWhen([]ConditionalSign{}): expected error, got nil")
	}
}

func TestDecimalDeclaration_WithNonzero(t *testing.T) {
	d := NewDecimalDeclaration()
	if d.Nonzero() {
		t.Fatal("Nonzero() before WithNonzero = true, want false")
	}
	d = d.WithNonzero()
	if !d.Nonzero() {
		t.Fatal("Nonzero() after WithNonzero = false, want true")
	}
}

func TestDecimalDeclaration_MinMax(t *testing.T) {
	min := mustDecimal(t, "0")
	max := mustDecimal(t, "10")

	d := NewDecimalDeclaration().WithMin(min).WithMax(max)

	gotMin, ok := d.Min()
	if !ok || gotMin.Compare(min) != 0 {
		t.Fatalf("Min() = (%v, %v), want (%v, true)", gotMin, ok, min)
	}
	gotMax, ok := d.Max()
	if !ok || gotMax.Compare(max) != 0 {
		t.Fatalf("Max() = (%v, %v), want (%v, true)", gotMax, ok, max)
	}
}

func TestDecimalDeclaration_ExclusiveMinMax(t *testing.T) {
	d := NewDecimalDeclaration()
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
