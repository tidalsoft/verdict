package field

import "testing"

var _ Declaration = TimestampDeclaration{}

func TestEncoding_String(t *testing.T) {
	tests := []struct {
		name string
		e    Encoding
		want string
	}{
		{"iso8601", EncodingISO8601, "iso8601"},
		{"epoch_s", EncodingEpochSeconds, "epoch_s"},
		{"epoch_ms", EncodingEpochMillis, "epoch_ms"},
		{"unspecified (zero value)", EncodingUnspecified, "unspecified"},
		{"out of range", Encoding(99), "unspecified"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.e.String(); got != tc.want {
				t.Fatalf("Encoding(%d).String() = %q, want %q", tc.e, got, tc.want)
			}
		})
	}
}

func TestTimestampDeclaration_ZeroValue(t *testing.T) {
	d := NewTimestampDeclaration()
	if d.Kind() != KindTimestamp {
		t.Fatalf("Kind() = %v, want %v", d.Kind(), KindTimestamp)
	}
	if _, ok := d.Encoding(); ok {
		t.Fatal("Encoding() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NotBefore(); ok {
		t.Fatal("NotBefore() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NotAfter(); ok {
		t.Fatal("NotAfter() on fresh declaration: ok = true, want false")
	}
	if _, ok := d.NullSemantics(); ok {
		t.Fatal("NullSemantics() on fresh declaration: ok = true, want false")
	}
}

func TestTimestampDeclaration_WithEncoding(t *testing.T) {
	d, err := NewTimestampDeclaration().WithEncoding(EncodingISO8601)
	if err != nil {
		t.Fatalf("WithEncoding: unexpected error: %v", err)
	}
	got, ok := d.Encoding()
	if !ok || got != EncodingISO8601 {
		t.Fatalf("Encoding() = (%v, %v), want (%v, true)", got, ok, EncodingISO8601)
	}
}

func TestTimestampDeclaration_WithEncoding_Invalid(t *testing.T) {
	if _, err := NewTimestampDeclaration().WithEncoding(Encoding(99)); err == nil {
		t.Fatal("WithEncoding(invalid): expected error, got nil")
	}
}

func TestTimestampDeclaration_WithNotBefore(t *testing.T) {
	d, err := NewTimestampDeclaration().WithNotBefore("now-5y")
	if err != nil {
		t.Fatalf("WithNotBefore: unexpected error: %v", err)
	}
	got, ok := d.NotBefore()
	if !ok || got != "now-5y" {
		t.Fatalf("NotBefore() = (%q, %v), want (%q, true)", got, ok, "now-5y")
	}
}

func TestTimestampDeclaration_WithNotBefore_Empty(t *testing.T) {
	if _, err := NewTimestampDeclaration().WithNotBefore(""); err == nil {
		t.Fatal("WithNotBefore(\"\"): expected error, got nil")
	}
}

func TestTimestampDeclaration_WithNotAfter(t *testing.T) {
	d, err := NewTimestampDeclaration().WithNotAfter("now+90d")
	if err != nil {
		t.Fatalf("WithNotAfter: unexpected error: %v", err)
	}
	got, ok := d.NotAfter()
	if !ok || got != "now+90d" {
		t.Fatalf("NotAfter() = (%q, %v), want (%q, true)", got, ok, "now+90d")
	}
}

func TestTimestampDeclaration_WithNotAfter_Empty(t *testing.T) {
	if _, err := NewTimestampDeclaration().WithNotAfter(""); err == nil {
		t.Fatal("WithNotAfter(\"\"): expected error, got nil")
	}
}

func TestTimestampDeclaration_WithNullSemantics(t *testing.T) {
	d, err := NewTimestampDeclaration().WithNullSemantics(NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics: unexpected error: %v", err)
	}
	got, ok := d.NullSemantics()
	if !ok || got != NullSemanticsDistinct {
		t.Fatalf("NullSemantics() = (%v, %v), want (%v, true)", got, ok, NullSemanticsDistinct)
	}
}

func TestTimestampDeclaration_WithNullSemantics_Invalid(t *testing.T) {
	if _, err := NewTimestampDeclaration().WithNullSemantics(NullSemantics(99)); err == nil {
		t.Fatal("WithNullSemantics(invalid): expected error, got nil")
	}
}
