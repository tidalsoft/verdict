package mu

import (
	"testing"
	"time"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU10 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU10: CheckID, Class (ClassD), the given Severity,
// and Outcome. Severity is a parameter, unlike this package's simpler
// want* helpers, because MU-10's "no timezone offset" branch is warn while
// every other branch is block -- see checkMU10ISO8601's own doc comment.
func wantMU10(t *testing.T, in Input, want verdict.Outcome, wantSeverity verdict.Severity) {
	t.Helper()
	res, applicable, err := checkMU10(in)
	if err != nil {
		t.Fatalf("checkMU10 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU10 applicable = false, want true")
	}
	if res.CheckID() != "MU-10" {
		t.Errorf("CheckID() = %q, want MU-10", res.CheckID())
	}
	if res.Class() != verdict.ClassD {
		t.Errorf("Class() = %v, want ClassD", res.Class())
	}
	if res.Severity() != wantSeverity {
		t.Errorf("Severity() = %v, want %v", res.Severity(), wantSeverity)
	}
	if res.Outcome() != want {
		t.Errorf("Outcome() = %v, want %v", res.Outcome(), want)
	}
}

func mustEncoding(t *testing.T, d field.TimestampDeclaration, e field.Encoding) field.TimestampDeclaration {
	t.Helper()
	out, err := d.WithEncoding(e)
	if err != nil {
		t.Fatalf("WithEncoding(%v) unexpected error: %v", e, err)
	}
	return out
}

func timestampInput(decl field.TimestampDeclaration, raw field.Value) Input {
	return Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    raw,
	}
}

func TestCheckMU10_MU_V17(t *testing.T) {
	// MU-V17: timestamp, epoch_s | 1754404927 | PASS | MU-10
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927")))
	wantMU10(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V18(t *testing.T) {
	// MU-V18: timestamp, epoch_s | 1754404927000 | FAIL | MU-10 (> 10^11)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927000")))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V19(t *testing.T) {
	// MU-V19: timestamp, epoch_ms | 1754404927 | FAIL | MU-10 (< 10^11)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927")))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V20(t *testing.T) {
	// MU-V20: timestamp, iso8601 | 1754404927 | FAIL | MU-10 (numeric)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927")))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V21(t *testing.T) {
	// MU-V21: timestamp, iso8601 | "2026-08-05T14:22:07" | FAIL @ warn | MU-10 (no offset)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	in := timestampInput(decl, field.NewStringValue("2026-08-05T14:22:07"))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityWarn)
}

func TestCheckMU10_MU_V52(t *testing.T) {
	// MU-V52: timestamp, epoch_ms | 1754404927000 | PASS | MU-10
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927000")))
	wantMU10(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V53(t *testing.T) {
	// MU-V53: timestamp, epoch_ms | "1754404927000" | FAIL | MU-10 (string under an epoch encoding)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewStringValue("1754404927000"))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V54(t *testing.T) {
	// MU-V54: timestamp (no encoding) | 1754404927 | INDETERMINATE | MU-10
	decl := field.NewTimestampDeclaration()
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "1754404927")))
	wantMU10(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}

func TestCheckMU10_MU_V55(t *testing.T) {
	// MU-V55: timestamp, iso8601 | true | INDETERMINATE | MU-10 (neither number nor string)
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	in := timestampInput(decl, field.NewBoolValue(true))
	wantMU10(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}

func TestCheckMU10_ISO8601_WithOffset_Pass(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	in := timestampInput(decl, field.NewStringValue("2026-08-05T14:22:07Z"))
	wantMU10(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU10_ISO8601_DoesNotParse_Fail(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingISO8601)
	in := timestampInput(decl, field.NewStringValue("not a date"))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_EpochSeconds_String_Fail(t *testing.T) {
	// The epoch_s mirror of vector 53 (epoch_ms): "the two epoch encodings
	// are symmetric on this point," per checkMU10EpochSeconds' own doc
	// comment, but no vector exercises the epoch_s side directly.
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	in := timestampInput(decl, field.NewStringValue("1754404927"))
	wantMU10(t, in, verdict.OutcomeFail, verdict.SeverityBlock)
}

func TestCheckMU10_EpochSeconds_Boolean_Indeterminate(t *testing.T) {
	// Neither a string nor a number under epoch_s: exercises checkMU10
	// EpochSeconds' final INDETERMINATE branch directly (not vector-
	// tested, but the same closed-evaluation rule vector 55 pins for
	// iso8601 applies symmetrically here).
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochSeconds)
	in := timestampInput(decl, field.NewBoolValue(false))
	wantMU10(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}

func TestCheckMU10_EpochMillis_Boolean_Indeterminate(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewBoolValue(false))
	wantMU10(t, in, verdict.OutcomeIndeterminate, verdict.SeverityBlock)
}

func TestCheckMU10_EpochMillis_Zero_Pass(t *testing.T) {
	// SPEC-MU's epoch_ms FAIL condition is "value < 10^11 and value > 0" --
	// zero is neither positive nor does it fail this branch, per
	// checkMU10EpochMillis' own doc comment.
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "0")))
	wantMU10(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU10_EpochMillis_Negative_Pass(t *testing.T) {
	decl := mustEncoding(t, field.NewTimestampDeclaration(), field.EncodingEpochMillis)
	in := timestampInput(decl, field.NewNumberValue(mustParse(t, "-100")))
	wantMU10(t, in, verdict.OutcomePass, verdict.SeverityBlock)
}

func TestCheckMU10_NotApplicable(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{"no declaration", Input{Field: "arguments.amount"}},
		{
			"declaration kind MU-10 does not apply to",
			Input{
				Field:    "arguments.amount",
				Registry: mustRegistryT(field.NewMoneyDeclaration()),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, applicable, err := checkMU10(tc.in)
			if err != nil {
				t.Fatalf("checkMU10 unexpected error: %v", err)
			}
			if applicable {
				t.Fatal("checkMU10 applicable = true, want false")
			}
		})
	}
}

func TestParseISO8601(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		wantOK      bool
		wantOffset  bool
		wantInstant time.Time
	}{
		{"offset Z", "2026-08-05T14:22:07Z", true, true, time.Date(2026, 8, 5, 14, 22, 7, 0, time.UTC)},
		{"offset numeric", "2026-08-05T14:22:07+02:00", true, true, time.Date(2026, 8, 5, 12, 22, 7, 0, time.UTC)},
		{"naive", "2026-08-05T14:22:07", true, false, time.Date(2026, 8, 5, 14, 22, 7, 0, time.UTC)},
		{"naive with fraction", "2026-08-05T14:22:07.5", true, false, time.Date(2026, 8, 5, 14, 22, 7, 500000000, time.UTC)},
		{"garbage", "not a date", false, false, time.Time{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotT, gotOffset, gotOK := parseISO8601(tc.in)
			if gotOK != tc.wantOK {
				t.Fatalf("parseISO8601(%q) ok = %v, want %v", tc.in, gotOK, tc.wantOK)
			}
			if !gotOK {
				return
			}
			if gotOffset != tc.wantOffset {
				t.Errorf("parseISO8601(%q) hasOffset = %v, want %v", tc.in, gotOffset, tc.wantOffset)
			}
			if !gotT.UTC().Equal(tc.wantInstant) {
				t.Errorf("parseISO8601(%q) = %v, want %v", tc.in, gotT.UTC(), tc.wantInstant)
			}
		})
	}
}
