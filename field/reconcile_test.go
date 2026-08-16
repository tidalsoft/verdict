package field

import (
	"testing"

	"github.com/tidalsoft/verdict/decimal"
)

func mustReconcileParse(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse(s)
	if err != nil {
		t.Fatalf("decimal.Parse(%q) unexpected error: %v", s, err)
	}
	return d
}

func TestNewReconcileDeclaration_Valid(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	if got := d.Total(); got != "arguments.total" {
		t.Errorf("Total() = %q, want arguments.total", got)
	}
	if got := d.Components(); got != "arguments.line_items[*].amount" {
		t.Errorf("Components() = %q, want arguments.line_items[*].amount", got)
	}
	if got := d.Adjustments(); len(got) != 0 {
		t.Errorf("Adjustments() = %v, want empty", got)
	}
	if got := d.Tolerance(); got.Compare(decimal.Decimal{}) != 0 {
		t.Errorf("Tolerance() = %v, want the exact zero decimal (SPEC-MU §2.4.3 default)", got)
	}
}

func TestNewReconcileDeclaration_EmptyTotal(t *testing.T) {
	if _, err := NewReconcileDeclaration("", "arguments.line_items[*].amount"); err == nil {
		t.Fatal("NewReconcileDeclaration with empty total succeeded, want error")
	}
}

func TestNewReconcileDeclaration_EmptyComponents(t *testing.T) {
	if _, err := NewReconcileDeclaration("arguments.total", ""); err == nil {
		t.Fatal("NewReconcileDeclaration with empty components succeeded, want error")
	}
}

func TestReconcileDeclaration_WithAdjustments(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	paths := []string{"arguments.tax", "arguments.shipping", "arguments.discount"}
	d, err = d.WithAdjustments(paths)
	if err != nil {
		t.Fatalf("WithAdjustments unexpected error: %v", err)
	}
	got := d.Adjustments()
	if len(got) != len(paths) {
		t.Fatalf("Adjustments() = %v, want %v", got, paths)
	}
	for i, p := range paths {
		if got[i] != p {
			t.Errorf("Adjustments()[%d] = %q, want %q", i, got[i], p)
		}
	}

	// Mutating the returned slice must not affect d -- defensive copy.
	got[0] = "mutated"
	if again := d.Adjustments(); again[0] != paths[0] {
		t.Errorf("Adjustments() after mutating a previous return = %q, want %q (defensive copy broken)", again[0], paths[0])
	}

	// Mutating the caller's own input slice after the call must not
	// affect d either.
	paths[1] = "mutated"
	if again := d.Adjustments(); again[1] != "arguments.shipping" {
		t.Errorf("Adjustments()[1] = %q after mutating caller's slice, want arguments.shipping (defensive copy broken)", again[1])
	}
}

func TestReconcileDeclaration_WithAdjustments_EmptyPath(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	if _, err := d.WithAdjustments([]string{"arguments.tax", ""}); err == nil {
		t.Fatal("WithAdjustments with an empty path succeeded, want error")
	}
}

func TestReconcileDeclaration_WithTolerance(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	d, err = d.WithTolerance(mustReconcileParse(t, "1"))
	if err != nil {
		t.Fatalf("WithTolerance unexpected error: %v", err)
	}
	if got := d.Tolerance(); got.Compare(mustReconcileParse(t, "1")) != 0 {
		t.Errorf("Tolerance() = %v, want 1", got)
	}
}

func TestReconcileDeclaration_WithTolerance_Zero(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	if _, err := d.WithTolerance(mustReconcileParse(t, "0")); err != nil {
		t.Fatalf("WithTolerance(0) unexpected error: %v", err)
	}
}

// TestReconcileDeclaration_WithTolerance_Negative pins this task's decision
// on MU-12's undefined negative-tolerance case: SPEC-MU never states what a
// negative tolerance means, and leaving it unvalidated would let a ruleset
// silently disable reconciliation (an arbitrarily negative tolerance makes
// every delta "exceed" it, or -- read the other way -- makes Reconciles'
// abs(delta) <= tolerance test always false, either way a silent behavior
// nobody asked for). This package's constructors validate everything, so a
// negative tolerance is rejected here rather than reaching MU-12 at all.
func TestReconcileDeclaration_WithTolerance_Negative(t *testing.T) {
	d, err := NewReconcileDeclaration("arguments.total", "arguments.line_items[*].amount")
	if err != nil {
		t.Fatalf("NewReconcileDeclaration unexpected error: %v", err)
	}
	if _, err := d.WithTolerance(mustReconcileParse(t, "-1")); err == nil {
		t.Fatal("WithTolerance(-1) succeeded, want error")
	}
}
