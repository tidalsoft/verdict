package field

import "testing"

func TestKind_String(t *testing.T) {
	tests := []struct {
		name string
		k    Kind
		want string
	}{
		{"money", KindMoney, "money"},
		{"quantity", KindQuantity, "quantity"},
		{"timestamp", KindTimestamp, "timestamp"},
		{"percentage", KindPercentage, "percentage"},
		{"decimal", KindDecimal, "decimal"},
		{"identifier", KindIdentifier, "identifier"},
		{"unspecified (zero value)", KindUnspecified, "unspecified"},
		{"out of range", Kind(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.k.String(); got != tc.want {
				t.Fatalf("Kind(%d).String() = %q, want %q", tc.k, got, tc.want)
			}
		})
	}
}
