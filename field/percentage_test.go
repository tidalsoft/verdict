package field

import "testing"

var _ Declaration = PercentageDeclaration{}

func TestDomain_String(t *testing.T) {
	tests := []struct {
		name string
		d    Domain
		want string
	}{
		{"unit_interval", DomainUnitInterval, "unit_interval"},
		{"hundred", DomainHundred, "hundred"},
		{"unspecified (zero value)", DomainUnspecified, "unspecified"},
		{"out of range", Domain(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.d.String(); got != tc.want {
				t.Fatalf("Domain(%d).String() = %q, want %q", tc.d, got, tc.want)
			}
		})
	}
}

func TestPercentageDeclaration_ZeroValue(t *testing.T) {
	d := NewPercentageDeclaration()
	if d.Kind() != KindPercentage {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindPercentage)
	}
	if _, ok := d.Domain(); ok {
		t.Fatal("Domain() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestPercentageDeclaration_WithDomain(t *testing.T) {
	d, err := NewPercentageDeclaration().WithDomain(DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain: unexpected error: %v", err)
	}
	got, ok := d.Domain()
	if !ok || got != DomainUnitInterval {
		t.Fatalf("Domain() = (%v, %v), want (%v, true)", got, ok, DomainUnitInterval)
	}
}

func TestPercentageDeclaration_WithDomain_Invalid(t *testing.T) {
	if _, err := NewPercentageDeclaration().WithDomain(Domain(99)); err == nil {
		t.Fatal("WithDomain(invalid): expected error, got nil")
	}
}

func TestPercentageDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewPercentageDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestPercentageDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewPercentageDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
