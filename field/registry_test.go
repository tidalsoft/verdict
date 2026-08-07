package field

import "testing"

func TestRegistry_ZeroValue(t *testing.T) {
	var r Registry
	if _, ok := r.Lookup("arguments.amount"); ok {
		t.Fatal("zero-value Registry.Lookup: ok = true, want false")
	}
}

func TestNewRegistry_LookupPresent(t *testing.T) {
	money := NewMoneyDeclaration()
	r, err := NewRegistry(map[string]Declaration{
		"arguments.amount": money,
	})
	if err != nil {
		t.Fatalf("NewRegistry: unexpected error: %v", err)
	}

	got, ok := r.Lookup("arguments.amount")
	if !ok {
		t.Fatal("Lookup(\"arguments.amount\"): ok = false, want true")
	}
	if got.Kind() != KindMoney {
		t.Fatalf("Lookup(\"arguments.amount\").Kind() = %v, want %v", got.Kind(), KindMoney)
	}
}

// TestNewRegistry_LookupAbsent is the field-level half of the
// absent-vs-present distinction the package doc comment describes: a path
// that was never declared must report ok=false, distinguishable from any
// Declaration -- including one with every attribute left unset.
func TestNewRegistry_LookupAbsent(t *testing.T) {
	r, err := NewRegistry(map[string]Declaration{
		"arguments.amount": NewMoneyDeclaration(),
	})
	if err != nil {
		t.Fatalf("NewRegistry: unexpected error: %v", err)
	}

	if _, ok := r.Lookup("arguments.currency"); ok {
		t.Fatal("Lookup(\"arguments.currency\") for an undeclared path: ok = true, want false")
	}
}

// TestAbsentDeclarationVsPresentButEmpty is the attribute-level half: a
// field that does carry a Declaration, but one that leaves a given
// attribute unset, must be distinguishable from a field with no
// Declaration at all -- both at the Registry level (this test) and via
// each accessor's own ok return (exercised per-type in money_test.go,
// quantity_test.go, etc.).
func TestAbsentDeclarationVsPresentButEmpty(t *testing.T) {
	r, err := NewRegistry(map[string]Declaration{
		// Declared, kind: money, but no scale attribute set at all.
		"arguments.amount": NewMoneyDeclaration(),
	})
	if err != nil {
		t.Fatalf("NewRegistry: unexpected error: %v", err)
	}

	declared, ok := r.Lookup("arguments.amount")
	if !ok {
		t.Fatal("Lookup(\"arguments.amount\"): ok = false, want true (field has a declaration)")
	}
	money, ok := declared.(MoneyDeclaration)
	if !ok {
		t.Fatalf("Lookup(\"arguments.amount\") type = %T, want MoneyDeclaration", declared)
	}
	if _, hasScale := money.Scale(); hasScale {
		t.Fatal("Scale() on a declaration that never set it: ok = true, want false")
	}

	// A wholly undeclared field must produce the same "not evaluable"
	// signal as the present-but-empty case above -- but at the Registry
	// level, not by handing back a Declaration masquerading as empty.
	if _, ok := r.Lookup("arguments.undeclared_field"); ok {
		t.Fatal("Lookup(\"arguments.undeclared_field\"): ok = true, want false")
	}
}

func TestNewRegistry_EmptyPath(t *testing.T) {
	_, err := NewRegistry(map[string]Declaration{
		"": NewMoneyDeclaration(),
	})
	if err == nil {
		t.Fatal("NewRegistry with empty field path: expected error, got nil")
	}
}

func TestNewRegistry_NilDeclaration(t *testing.T) {
	_, err := NewRegistry(map[string]Declaration{
		"arguments.amount": nil,
	})
	if err == nil {
		t.Fatal("NewRegistry with nil declaration: expected error, got nil")
	}
}

func TestNewRegistry_CopiesInput(t *testing.T) {
	input := map[string]Declaration{
		"arguments.amount": NewMoneyDeclaration(),
	}
	r, err := NewRegistry(input)
	if err != nil {
		t.Fatalf("NewRegistry: unexpected error: %v", err)
	}

	// Mutating the caller's map after construction must not affect the
	// Registry (immutability -- see the package doc comment).
	input["arguments.amount"] = nil
	delete(input, "arguments.amount")
	input["arguments.new_field"] = NewQuantityDeclaration()

	if _, ok := r.Lookup("arguments.amount"); !ok {
		t.Fatal("Lookup(\"arguments.amount\") after caller mutated its input map: ok = false, want true")
	}
	if _, ok := r.Lookup("arguments.new_field"); ok {
		t.Fatal("Lookup(\"arguments.new_field\") added to caller's map post-construction: ok = true, want false")
	}
}
