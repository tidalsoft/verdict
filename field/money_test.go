package field

import (
	"testing"

	"github.com/tidalsoft/verdict/decimal"
)

var _ Declaration = MoneyDeclaration{}

func TestScale_String(t *testing.T) {
	tests := []struct {
		name string
		s    Scale
		want string
	}{
		{"minor_units", ScaleMinorUnits, "minor_units"},
		{"major_units", ScaleMajorUnits, "major_units"},
		{"unspecified (zero value)", ScaleUnspecified, "unspecified"},
		{"out of range", Scale(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Fatalf("Scale(%d).String() = %q, want %q", tc.s, got, tc.want)
			}
		})
	}
}

func TestSign_String(t *testing.T) {
	tests := []struct {
		name string
		s    Sign
		want string
	}{
		{"positive", SignPositive, "positive"},
		{"negative", SignNegative, "negative"},
		{"any", SignAny, "any"},
		{"unspecified (zero value)", SignUnspecified, "unspecified"},
		{"out of range", Sign(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.s.String(); got != tc.want {
				t.Fatalf("Sign(%d).String() = %q, want %q", tc.s, got, tc.want)
			}
		})
	}
}

func TestNewConditionalSign(t *testing.T) {
	c, err := NewConditionalSign("arguments.type", "refund", SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign: unexpected error: %v", err)
	}
	if got := c.WhenField(); got != "arguments.type" {
		t.Fatalf("WhenField() = %q, want %q", got, "arguments.type")
	}
	if got := c.WhenValue(); got != "refund" {
		t.Fatalf("WhenValue() = %q, want %q", got, "refund")
	}
	if got := c.Sign(); got != SignNegative {
		t.Fatalf("Sign() = %v, want %v", got, SignNegative)
	}
}

func TestNewConditionalSign_EmptyField(t *testing.T) {
	if _, err := NewConditionalSign("", "refund", SignNegative); err == nil {
		t.Fatal("NewConditionalSign with empty whenField: expected error, got nil")
	}
}

func TestNewConditionalSign_InvalidSign(t *testing.T) {
	if _, err := NewConditionalSign("arguments.type", "refund", Sign(99)); err == nil {
		t.Fatal("NewConditionalSign with invalid sign: expected error, got nil")
	}
}

func TestMoneyDeclaration_ZeroValue(t *testing.T) {
	d := NewMoneyDeclaration()
	if d.Kind() != KindMoney {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindMoney)
	}
	if _, ok := d.CurrencyField(); ok {
		t.Fatal("CurrencyField() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.TargetCurrencyField(); ok {
		t.Fatal("TargetCurrencyField() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.Scale(); ok {
		t.Fatal("Scale() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.Sign(); ok {
		t.Fatal("Sign() on fresh declaration: ok = true, want false")
	}
	if got := d.SignWhen(); got != nil {
		t.Fatalf("SignWhen() on fresh declaration = %v, want nil", got)
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
	if d.Nonzero() {
		t.Fatal("Nonzero() on fresh declaration = true, want false")
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestMoneyDeclaration_WithCurrencyField(t *testing.T) {
	d, err := NewMoneyDeclaration().WithCurrencyField("arguments.currency")
	if err != nil {
		t.Fatalf("WithCurrencyField: unexpected error: %v", err)
	}
	got, ok := d.CurrencyField()
	if !ok || got != "arguments.currency" {
		t.Fatalf("CurrencyField() = (%q, %v), want (%q, true)", got, ok, "arguments.currency")
	}
}

func TestMoneyDeclaration_WithCurrencyField_Empty(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithCurrencyField(""); err == nil {
		t.Fatal("WithCurrencyField(\"\"): expected error, got nil")
	}
}

func TestMoneyDeclaration_WithTargetCurrencyField(t *testing.T) {
	d, err := NewMoneyDeclaration().WithTargetCurrencyField("arguments.target_currency")
	if err != nil {
		t.Fatalf("WithTargetCurrencyField: unexpected error: %v", err)
	}
	got, ok := d.TargetCurrencyField()
	if !ok || got != "arguments.target_currency" {
		t.Fatalf("TargetCurrencyField() = (%q, %v), want (%q, true)", got, ok, "arguments.target_currency")
	}
}

func TestMoneyDeclaration_WithTargetCurrencyField_Empty(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithTargetCurrencyField(""); err == nil {
		t.Fatal("WithTargetCurrencyField(\"\"): expected error, got nil")
	}
}

func TestMoneyDeclaration_TargetCurrencyField_RejectsSelfReference(t *testing.T) {
	// A currency_field and target_currency_field naming the same path
	// would make MU-03 compare a currency against itself and PASS
	// unconditionally -- rejected regardless of which is declared first.
	t.Run("currency_field declared first", func(t *testing.T) {
		d, err := NewMoneyDeclaration().WithCurrencyField("arguments.currency")
		if err != nil {
			t.Fatalf("WithCurrencyField: unexpected error: %v", err)
		}
		if _, err := d.WithTargetCurrencyField("arguments.currency"); err == nil {
			t.Fatal("WithTargetCurrencyField(same path as currency_field): expected error, got nil")
		}
	})
	t.Run("target_currency_field declared first", func(t *testing.T) {
		d, err := NewMoneyDeclaration().WithTargetCurrencyField("arguments.currency")
		if err != nil {
			t.Fatalf("WithTargetCurrencyField: unexpected error: %v", err)
		}
		if _, err := d.WithCurrencyField("arguments.currency"); err == nil {
			t.Fatal("WithCurrencyField(same path as target_currency_field): expected error, got nil")
		}
	})
	t.Run("distinct paths still succeed", func(t *testing.T) {
		d, err := NewMoneyDeclaration().WithCurrencyField("arguments.currency")
		if err != nil {
			t.Fatalf("WithCurrencyField: unexpected error: %v", err)
		}
		d, err = d.WithTargetCurrencyField("arguments.target_currency")
		if err != nil {
			t.Fatalf("WithTargetCurrencyField(distinct path): unexpected error: %v", err)
		}
		got, ok := d.TargetCurrencyField()
		if !ok || got != "arguments.target_currency" {
			t.Fatalf("TargetCurrencyField() = (%q, %v), want (%q, true)", got, ok, "arguments.target_currency")
		}
	})
}

func TestMoneyDeclaration_WithScale(t *testing.T) {
	d, err := NewMoneyDeclaration().WithScale(ScaleMinorUnits)
	if err != nil {
		t.Fatalf("WithScale: unexpected error: %v", err)
	}
	got, ok := d.Scale()
	if !ok || got != ScaleMinorUnits {
		t.Fatalf("Scale() = (%v, %v), want (%v, true)", got, ok, ScaleMinorUnits)
	}
}

func TestMoneyDeclaration_WithScale_Invalid(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithScale(Scale(99)); err == nil {
		t.Fatal("WithScale(invalid): expected error, got nil")
	}
}

func TestMoneyDeclaration_WithSign(t *testing.T) {
	d, err := NewMoneyDeclaration().WithSign(SignPositive)
	if err != nil {
		t.Fatalf("WithSign: unexpected error: %v", err)
	}
	got, ok := d.Sign()
	if !ok || got != SignPositive {
		t.Fatalf("Sign() = (%v, %v), want (%v, true)", got, ok, SignPositive)
	}
}

func TestMoneyDeclaration_WithSign_Invalid(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithSign(Sign(99)); err == nil {
		t.Fatal("WithSign(invalid): expected error, got nil")
	}
}

func TestMoneyDeclaration_WithSignWhen(t *testing.T) {
	refund, err := NewConditionalSign("arguments.type", "refund", SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign: unexpected error: %v", err)
	}
	charge, err := NewConditionalSign("arguments.type", "charge", SignPositive)
	if err != nil {
		t.Fatalf("NewConditionalSign: unexpected error: %v", err)
	}
	conds := []ConditionalSign{refund, charge}

	d, err := NewMoneyDeclaration().WithSignWhen(conds)
	if err != nil {
		t.Fatalf("WithSignWhen: unexpected error: %v", err)
	}

	// Mutating the slice passed in must not affect the stored declaration
	// (defensive copy on the way in).
	conds[0] = ConditionalSign{}

	got := d.SignWhen()
	if len(got) != 2 || got[0] != refund || got[1] != charge {
		t.Fatalf("SignWhen() = %+v, want [%+v %+v]", got, refund, charge)
	}

	// Mutating the returned slice must not affect the stored declaration
	// (defensive copy on the way out).
	got[0] = ConditionalSign{}
	again := d.SignWhen()
	if again[0] != refund {
		t.Fatalf("SignWhen() after mutating a prior result = %+v, want unchanged %+v", again[0], refund)
	}
}

func TestMoneyDeclaration_WithSignWhen_Empty(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithSignWhen(nil); err == nil {
		t.Fatal("WithSignWhen(nil): expected error, got nil")
	}
	if _, err := NewMoneyDeclaration().WithSignWhen([]ConditionalSign{}); err == nil {
		t.Fatal("WithSignWhen([]ConditionalSign{}): expected error, got nil")
	}
}

func TestMoneyDeclaration_MinMax(t *testing.T) {
	min, err := decimal.Parse("1")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}
	max, err := decimal.Parse("100000000")
	if err != nil {
		t.Fatalf("decimal.Parse: %v", err)
	}

	d := NewMoneyDeclaration().WithMin(min).WithMax(max)

	gotMin, ok := d.Min()
	if !ok || gotMin.Compare(min) != 0 {
		t.Fatalf("Min() = (%v, %v), want (%v, true)", gotMin, ok, min)
	}
	gotMax, ok := d.Max()
	if !ok || gotMax.Compare(max) != 0 {
		t.Fatalf("Max() = (%v, %v), want (%v, true)", gotMax, ok, max)
	}

	if d.ExclusiveMin() {
		t.Fatal("ExclusiveMin() before WithExclusiveMin = true, want false")
	}
	if d.ExclusiveMax() {
		t.Fatal("ExclusiveMax() before WithExclusiveMax = true, want false")
	}

	d = d.WithExclusiveMin().WithExclusiveMax()
	if !d.ExclusiveMin() {
		t.Fatal("ExclusiveMin() after WithExclusiveMin = false, want true")
	}
	if !d.ExclusiveMax() {
		t.Fatal("ExclusiveMax() after WithExclusiveMax = false, want true")
	}
}

func TestMoneyDeclaration_Nonzero(t *testing.T) {
	d := NewMoneyDeclaration()
	if d.Nonzero() {
		t.Fatal("Nonzero() before WithNonzero = true, want false")
	}
	d = d.WithNonzero()
	if !d.Nonzero() {
		t.Fatal("Nonzero() after WithNonzero = false, want true")
	}
}

func TestMoneyDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewMoneyDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestMoneyDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewMoneyDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
