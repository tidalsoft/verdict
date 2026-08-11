package field

import (
	"testing"

	"github.com/tidalsoft/verdict/decimal"
)

func mustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse(s)
	if err != nil {
		t.Fatalf("decimal.Parse(%q) unexpected error: %v", s, err)
	}
	return d
}

func TestValueKind_String(t *testing.T) {
	cases := []struct {
		name string
		k    ValueKind
		want string
	}{
		{"string", ValueKindString, "string"},
		{"number", ValueKindNumber, "number"},
		{"bool", ValueKindBool, "bool"},
		{"null", ValueKindNull, "null"},
		{"non_comparable", ValueKindNonComparable, "non_comparable"},
		{"unspecified (zero value)", ValueKindUnspecified, "unspecified"},
		{"out of range", ValueKind(99), "unspecified"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.k.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValue_ZeroValue(t *testing.T) {
	var v Value
	if v.Kind() != ValueKindUnspecified {
		t.Errorf("Kind() = %v, want ValueKindUnspecified", v.Kind())
	}
	if v.Comparable() {
		t.Error("Comparable() on zero Value = true, want false")
	}
	if v.IsNull() {
		t.Error("IsNull() on zero Value = true, want false")
	}
}

func TestValue_Accessors(t *testing.T) {
	sv := NewStringValue("refund")
	if got, ok := sv.StringValue(); !ok || got != "refund" {
		t.Errorf("StringValue() = (%q, %v), want (%q, true)", got, ok, "refund")
	}
	if _, ok := sv.NumberValue(); ok {
		t.Error("NumberValue() on a string Value: ok = true, want false")
	}
	if _, ok := sv.BoolValue(); ok {
		t.Error("BoolValue() on a string Value: ok = true, want false")
	}
	if !sv.Comparable() {
		t.Error("Comparable() on a string Value = false, want true")
	}

	nv := NewNumberValue(mustDecimal(t, "500"))
	got, ok := nv.NumberValue()
	if !ok || got.Compare(mustDecimal(t, "500")) != 0 {
		t.Errorf("NumberValue() = (%v, %v), want (500, true)", got, ok)
	}
	if _, ok := nv.StringValue(); ok {
		t.Error("StringValue() on a number Value: ok = true, want false")
	}

	bv := NewBoolValue(true)
	if got, ok := bv.BoolValue(); !ok || got != true {
		t.Errorf("BoolValue() = (%v, %v), want (true, true)", got, ok)
	}
	if _, ok := bv.NumberValue(); ok {
		t.Error("NumberValue() on a bool Value: ok = true, want false")
	}

	nullV := NewNullValue()
	if !nullV.IsNull() {
		t.Error("IsNull() on NewNullValue() = false, want true")
	}
	if !nullV.Comparable() {
		t.Error("Comparable() on null Value = false, want true")
	}
	if _, ok := nullV.StringValue(); ok {
		t.Error("StringValue() on a null Value: ok = true, want false")
	}

	nc := NewNonComparableValue()
	if nc.Comparable() {
		t.Error("Comparable() on NewNonComparableValue() = true, want false")
	}
	if nc.IsNull() {
		t.Error("IsNull() on NewNonComparableValue() = true, want false")
	}
}

func TestValue_Equal(t *testing.T) {
	cases := []struct {
		name string
		a    Value
		b    Value
		want bool
	}{
		{"equal strings", NewStringValue("refund"), NewStringValue("refund"), true},
		{"different strings", NewStringValue("refund"), NewStringValue("Refund"), false},
		{"equal numbers, different transported form", NewNumberValue(mustDecimal(t, "500")), NewNumberValue(mustDecimal(t, "500.0")), true},
		{"different numbers", NewNumberValue(mustDecimal(t, "500")), NewNumberValue(mustDecimal(t, "501")), false},
		{"equal bools", NewBoolValue(true), NewBoolValue(true), true},
		{"different bools", NewBoolValue(true), NewBoolValue(false), false},
		{"null equals null", NewNullValue(), NewNullValue(), true},
		{"number does not match string", NewNumberValue(mustDecimal(t, "500")), NewStringValue("500"), false},
		{"string does not match number", NewStringValue("500"), NewNumberValue(mustDecimal(t, "500")), false},
		{"non-comparable never equal, even to itself", NewNonComparableValue(), NewNonComparableValue(), false},
		{"comparable vs non-comparable", NewStringValue("x"), NewNonComparableValue(), false},
		{"non-comparable vs comparable", NewNonComparableValue(), NewStringValue("x"), false},
		{"zero value vs zero value", Value{}, Value{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.Equal(tc.b); got != tc.want {
				t.Errorf("Equal() = %v, want %v", got, tc.want)
			}
		})
	}
}
