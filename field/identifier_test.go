package field

import "testing"

var _ Declaration = IdentifierDeclaration{}

func TestIdentifierDeclaration_ZeroValue(t *testing.T) {
	d := NewIdentifierDeclaration()
	if d.Kind() != KindIdentifier {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindIdentifier)
	}
	if _, ok := d.Scheme(); ok {
		t.Fatal("Scheme() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestIdentifierDeclaration_WithScheme_Known(t *testing.T) {
	tests := []Scheme{
		SchemeIBAN, SchemeLuhn, SchemeISBN13, SchemeISBN10,
		SchemeGTIN8, SchemeGTIN12, SchemeGTIN13, SchemeGTIN14,
		SchemeLEI, SchemeBIC, SchemeISO4217, SchemeISO3166Alpha2,
	}
	for _, s := range tests {
		t.Run(string(s), func(t *testing.T) {
			d, err := NewIdentifierDeclaration().WithScheme(s)
			if err != nil {
				t.Fatalf("WithScheme(%q): unexpected error: %v", s, err)
			}
			got, ok := d.Scheme()
			if !ok || got != s {
				t.Fatalf("Scheme() = (%q, %v), want (%q, true)", got, ok, s)
			}
		})
	}
}

func TestIdentifierDeclaration_WithScheme_Unrecognised(t *testing.T) {
	// MU-16 requires an unrecognised scheme to evaluate as INDETERMINATE,
	// never to be rejected here -- so an arbitrary non-empty string must be
	// accepted at the declaration layer.
	d, err := NewIdentifierDeclaration().WithScheme(Scheme("not_a_real_scheme"))
	if err != nil {
		t.Fatalf("WithScheme(unrecognised): unexpected error: %v", err)
	}
	got, ok := d.Scheme()
	if !ok || got != Scheme("not_a_real_scheme") {
		t.Fatalf("Scheme() = (%q, %v), want (%q, true)", got, ok, "not_a_real_scheme")
	}
}

func TestIdentifierDeclaration_WithScheme_Empty(t *testing.T) {
	if _, err := NewIdentifierDeclaration().WithScheme(""); err == nil {
		t.Fatal("WithScheme(\"\"): expected error, got nil")
	}
}

func TestIdentifierDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewIdentifierDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestIdentifierDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewIdentifierDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
