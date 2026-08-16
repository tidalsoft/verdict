package mu

import (
	"testing"
	"time"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU11 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU11: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-11's only severity), and Outcome.
func wantMU11(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU11(in)
	if err != nil {
		t.Fatalf("checkMU11 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU11 applicable = false, want true")
	}
	if res.CheckID() != "MU-11" {
		t.Errorf("CheckID() = %q, want MU-11", res.CheckID())
	}
	if res.Class() != verdict.ClassD {
		t.Errorf("Class() = %v, want ClassD", res.Class())
	}
	if res.Severity() != verdict.SeverityBlock {
		t.Errorf("Severity() = %v, want SeverityBlock", res.Severity())
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

func mustNotAfter(t *testing.T, d field.TimestampDeclaration, bound string) field.TimestampDeclaration {
	t.Helper()
	out, err := d.WithNotAfter(bound)
	if err != nil {
		t.Fatalf("WithNotAfter(%q) unexpected error: %v", bound, err)
	}
	return out
}

func mustNotBefore(t *testing.T, d field.TimestampDeclaration, bound string) field.TimestampDeclaration {
	t.Helper()
	out, err := d.WithNotBefore(bound)
	if err != nil {
		t.Fatalf("WithNotBefore(%q) unexpected error: %v", bound, err)
	}
	return out
}

// referenceEvaluatedAt is the fixed evaluation timestamp SPEC-MU §8.3's
// MU-11 vectors declare: 2026-08-05T14:22:07Z.
func referenceEvaluatedAt(t *testing.T) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, "2026-08-05T14:22:07Z")
	if err != nil {
		t.Fatalf("time.Parse unexpected error: %v", err)
	}
	return ts
}

func TestCheckMU11_MU_V71(t *testing.T) {
	// MU-V71: timestamp, iso8601, not_after: now, evaluated_at
	// 2026-08-05T14:22:07Z | "2030-01-01T00:00:00Z" | FAIL | MU-11
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "now")
	in := Input{
		Field:          "arguments.amount",
		Registry:       mustRegistryT(decl),
		HasRawValue:    true,
		RawValue:       field.NewStringValue("2030-01-01T00:00:00Z"),
		EvaluatedAt:    referenceEvaluatedAt(t),
		HasEvaluatedAt: true,
	}
	wantMU11(t, in, verdict.OutcomeFail)
}

func TestCheckMU11_MU_V72(t *testing.T) {
	// MU-V72: timestamp, iso8601, not_before: now-5y, same evaluated_at |
	// "2026-08-05T00:00:00Z" | PASS | MU-11
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotBefore(t, decl, "now-5y")
	in := Input{
		Field:          "arguments.amount",
		Registry:       mustRegistryT(decl),
		HasRawValue:    true,
		RawValue:       field.NewStringValue("2026-08-05T00:00:00Z"),
		EvaluatedAt:    referenceEvaluatedAt(t),
		HasEvaluatedAt: true,
	}
	wantMU11(t, in, verdict.OutcomePass)
}

func TestCheckMU11_MU_V73(t *testing.T) {
	// MU-V73: timestamp, iso8601, not_after: "2030-01-01T00:00:00Z" |
	// "not a date" | INDETERMINATE | MU-11 (does not parse)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "2030-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("not a date"),
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_MU_V99(t *testing.T) {
	// MU-V99: timestamp, not_after: "2030-01-01T00:00:00Z", no encoding |
	// "2026-08-05T00:00:00Z" | INDETERMINATE | MU-11 (encoding not declared)
	decl := mustNotAfter(t, field.NewTimestampDeclaration(), "2030-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("2026-08-05T00:00:00Z"),
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_MU_V100(t *testing.T) {
	// MU-V100: timestamp, iso8601, not_after: now+90d, no evaluated_at |
	// "2026-08-05T00:00:00Z" | INDETERMINATE | MU-11 (relative bound unresolvable)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "now+90d")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("2026-08-05T00:00:00Z"),
		// HasEvaluatedAt deliberately left false.
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_NotApplicable(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{"no declaration", Input{Field: "arguments.amount"}},
		{
			"declaration kind MU-11 does not apply to",
			Input{Field: "arguments.amount", Registry: mustRegistryT(field.NewMoneyDeclaration())},
		},
		{
			"neither not_before nor not_after declared",
			Input{
				Field:    "arguments.amount",
				Registry: mustRegistryT(mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, applicable, err := checkMU11(tc.in)
			if err != nil {
				t.Fatalf("checkMU11 unexpected error: %v", err)
			}
			if applicable {
				t.Fatal("checkMU11 applicable = true, want false")
			}
		})
	}
}

func TestCheckMU11_EpochSeconds_WithinBounds_Pass(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	decl = mustNotBefore(t, decl, "2020-01-01T00:00:00Z")
	decl = mustNotAfter(t, decl, "2030-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, "1754404927")), // 2025-08-05T14:22:07Z
	}
	wantMU11(t, in, verdict.OutcomePass)
}

func TestCheckMU11_EpochMillis_OutsideBounds_Fail(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	decl = mustNotBefore(t, decl, "2020-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, "0")), // 1970-01-01T00:00:00Z
	}
	wantMU11(t, in, verdict.OutcomeFail)
}

func TestCheckMU11_EpochValue_FractionalDoesNotParse_Indeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	decl = mustNotBefore(t, decl, "2020-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, "1754404927.5")),
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_ValueNotAResolvableType_Indeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotBefore(t, decl, "2020-01-01T00:00:00Z")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewBoolValue(true),
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_AbsoluteBoundDoesNotParse_Indeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "not a bound")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("2026-08-05T00:00:00Z"),
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_MalformedRelativeBound_Indeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "now+abc")
	in := Input{
		Field:          "arguments.amount",
		Registry:       mustRegistryT(decl),
		HasRawValue:    true,
		RawValue:       field.NewStringValue("2026-08-05T00:00:00Z"),
		EvaluatedAt:    referenceEvaluatedAt(t),
		HasEvaluatedAt: true,
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU11_NotBeforeUnresolvable_Indeterminate(t *testing.T) {
	// The mirror of TestCheckMU11_MU_V100 on the not_before side: every
	// vector for this branch happens to exercise not_after, so this pins
	// resolveTimeBound's !ok path being reached from the not_before arm
	// of checkMU11 as well.
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotBefore(t, decl, "now-5y")
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("2026-08-05T00:00:00Z"),
		// HasEvaluatedAt deliberately left false.
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}

func TestParseTimestampValue_EpochOverflow(t *testing.T) {
	// An epoch value with more digits than any int64 can hold: DecimalPlaces
	// is still 0 (it is an integer), so this reaches strconv.ParseInt,
	// which is the branch this test exists to exercise.
	huge := mustParse(t, "99999999999999999999999999999999")
	_, ok := parseTimestampValue(field.NewNumberValue(huge), field.EncodingEpochSeconds)
	if ok {
		t.Fatal("parseTimestampValue with an int64-overflowing epoch value succeeded, want false")
	}
}

func TestParseRelativeDuration_DigitOverflow(t *testing.T) {
	// A digit run isASCIIDigits accepts but strconv.ParseInt cannot hold
	// in an int64.
	_, ok := parseRelativeDuration("now+99999999999999999999d")
	if ok {
		t.Fatal("parseRelativeDuration with an int64-overflowing digit run succeeded, want false")
	}
}

func TestParseRelativeDuration(t *testing.T) {
	cases := []struct {
		in     string
		wantOK bool
		want   time.Duration
	}{
		{"now", true, 0},
		{"now+90d", true, 90 * 24 * time.Hour},
		{"now-5y", true, -5 * 365 * 24 * time.Hour},
		{"now+1s", true, time.Second},
		{"now+1m", true, time.Minute},
		{"now+1h", true, time.Hour},
		{"now+1w", true, 7 * 24 * time.Hour},
		{"now?1d", false, 0},   // not a valid sign
		{"now+d", false, 0},    // no digits
		{"now+1x", false, 0},   // unrecognised unit
		{"now+1", false, 0},    // too short: no unit
		{"now+1.5d", false, 0}, // not an unsigned integer
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseRelativeDuration(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("parseRelativeDuration(%q) ok = %v, want %v", tc.in, ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("parseRelativeDuration(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestResolveTimeBound_AbsoluteMalformed(t *testing.T) {
	_, ok := resolveTimeBound("not a timestamp", time.Time{}, false)
	if ok {
		t.Fatal("resolveTimeBound with a malformed absolute bound succeeded, want false")
	}
}

// TestParseRelativeDuration_RejectsOverflow pins the boundary between a
// relative bound whose nanosecond count fits in an int64 and one that does
// not. Below the boundary the duration is exact; at or above it the
// multiplication would wrap, and a wrapped duration is not detectable after
// the fact -- "now+300y" silently becomes a bound in 1741, against which
// MU-11 then FAILs timestamps that should pass. An unresolvable bound owes
// the caller INDETERMINATE (vector MU-V100), never a confident wrong
// verdict, so parseRelativeDuration must refuse these outright.
func TestParseRelativeDuration_RejectsOverflow(t *testing.T) {
	cases := []struct {
		in     string
		wantOK bool
	}{
		{"now+292y", true},                  // 292 years of nanoseconds still fits
		{"now+293y", false},                 // one more wraps int64
		{"now+300y", false},                 // the value that produced a bound in 1741
		{"now-300y", false},                 // sign is applied after the product, so both
		{"now+106751d", true},               // day boundary, just inside
		{"now+106752d", false},              // day boundary, just outside
		{"now+15250w", true},                // week boundary, just inside
		{"now+15251w", false},               // week boundary, just outside
		{"now+9223372036854775807s", false}, // MaxInt64 seconds
		{"now-9223372036854775807s", false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseRelativeDuration(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("parseRelativeDuration(%q) ok = %v, want %v (got %v)", tc.in, ok, tc.wantOK, got)
			}
			if ok && got == 0 && tc.in != "now" {
				t.Errorf("parseRelativeDuration(%q) accepted but returned a zero duration", tc.in)
			}
			if ok && (tc.in[3] == '+') != (got > 0) {
				t.Errorf("parseRelativeDuration(%q) = %v, sign disagrees with the bound", tc.in, got)
			}
		})
	}
}

// TestCheckMU11_OverflowingBoundIsIndeterminate is the end-to-end case the
// unit test above exists to prevent: a far-future ceiling an author could
// plausibly write, against a value five months out that must pass.
func TestCheckMU11_OverflowingBoundIsIndeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	decl = mustNotAfter(t, decl, "now+300y")
	in := Input{
		Field:          "arguments.amount",
		Registry:       mustRegistryT(decl),
		HasRawValue:    true,
		RawValue:       field.NewStringValue("2027-01-01T00:00:00Z"),
		EvaluatedAt:    referenceEvaluatedAt(t),
		HasEvaluatedAt: true,
	}
	wantMU11(t, in, verdict.OutcomeIndeterminate)
}
