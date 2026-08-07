package field

import "testing"

func TestNullSemantics_String(t *testing.T) {
	tests := []struct {
		name string
		n    NullSemantics
		want string
	}{
		{"distinct", NullSemanticsDistinct, "distinct"},
		{"unspecified (zero value)", NullSemanticsUnspecified, "unspecified"},
		{"out of range", NullSemantics(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.n.String(); got != tc.want {
				t.Fatalf("NullSemantics(%d).String() = %q, want %q", tc.n, got, tc.want)
			}
		})
	}
}

func TestCommon_WithNullSemantics(t *testing.T) {
	var c common
	got, err := c.withNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("withNullSemantics(distinct): unexpected error: %v", err)
	}
	if ns, ok := got.NullSemantics(); !ok || ns != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", ns, ok, NullSemanticsDistinct)
	}
}

func TestCommon_WithNullSemantics_Invalid(t *testing.T) {
	var c common
	if _, err := c.withNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("withNullSemantics(invalid): expected error, got nil")
	}
}

func TestCommon_NullSemantics_Absent(t *testing.T) {
	var c common
	if _, ok := c.NullSemantics(); ok {
		t.Fatal("zero-value common.NullSemantics(): ok = true, want false")
	}
}
