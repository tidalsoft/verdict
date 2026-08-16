package mu

import (
	"time"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// epochMagnitudeThreshold is the 10^11 boundary MU-10 compares an epoch
// value's magnitude against (SPEC-MU §5): "1754404927" (about 1.75e9) is a
// plausible epoch_s value and an implausible epoch_ms one, and the
// boundary is symmetric between the two encodings -- see checkMU10
// EpochSeconds and checkMU10EpochMillis, and MU-10's own "Boundary note"
// for why the same numeric threshold lands on two very different dates
// depending on which unit it is read in.
func epochMagnitudeThreshold() decimal.Decimal { return mustParseDecimal("100000000000") }

// checkMU10 implements the timestamp_encoding check (MU-10, SPEC-MU §5).
//
// MU-10 catches epoch-seconds versus epoch-milliseconds confusion and
// ISO/epoch substitution.
//
// Applicability (SPEC-MU §2.5.1: applies to timestamp, no further gate):
//   - no declaration for the field, or a declaration whose kind is not
//     timestamp → not applicable.
//
// Branch matrix, once applicable:
//   - encoding is not declared → INDETERMINATE (§2.5.2: a timestamp has an
//     encoding whether or not the ruleset says which -- a required input,
//     not a gate; vector 54).
//   - encoding: iso8601 → checkMU10ISO8601.
//   - encoding: epoch_s → checkMU10EpochSeconds.
//   - encoding: epoch_ms → checkMU10EpochMillis.
//
// MU-10 is not value-dependent (SPEC-MU §2.6.3's table: no value-dependent
// check is ever applicable to a `timestamp` field at all, so coercion
// never runs on one and there is nothing for this check to be suppressed
// by). It reads Input.RawValue directly -- the field's own value in its
// original JSON shape -- never Input.Value/Provenance/ValueCoercionFailed.
func checkMU10(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	tsDecl, ok := decl.(field.TimestampDeclaration)
	if !ok {
		return notApplicable()
	}

	encoding, hasEncoding := tsDecl.Encoding()
	if !hasEncoding {
		return indeterminateResult("MU-10")
	}

	if encoding == field.EncodingISO8601 {
		return checkMU10ISO8601(in.RawValue)
	}
	if encoding == field.EncodingEpochSeconds {
		return checkMU10EpochSeconds(in.RawValue)
	}
	// encoding == field.EncodingEpochMillis: field.TimestampDeclaration.
	// WithEncoding accepts only these three Encoding values, and
	// hasEncoding == true confirms one was actually set, so this is the
	// only remaining possibility once the two branches above have
	// returned.
	return checkMU10EpochMillis(in.RawValue)
}

// checkMU10ISO8601 is MU-10's `encoding: iso8601` branch.
//
//   - the value is a JSON number → FAIL (vector 20): an iso8601 field
//     requires a string.
//   - the value is a JSON string that fails RFC 3339 parsing outright →
//     FAIL.
//   - the value is a JSON string that parses as a timestamp but carries no
//     timezone offset → FAIL at severity warn (vector 21): "a naive
//     timestamp is an ambiguity, not an error."
//   - the value is a JSON string that parses with an explicit offset →
//     PASS.
//   - the value is neither a JSON number nor a JSON string (including
//     Input.HasRawValue false, an absent value) → INDETERMINATE (vector
//     55): every condition above reads one shape or the other, so
//     anything else matches none of them, and §2.1 forbids reaching PASS
//     by exhausting conditions against a value this check could not
//     interpret at all.
func checkMU10ISO8601(v field.Value) (verdict.Result, bool, error) {
	switch v.Kind() {
	case field.ValueKindNumber:
		return failResult("MU-10")
	case field.ValueKindString:
		s, _ := v.StringValue()
		_, hasOffset, ok := parseISO8601(s)
		if !ok {
			return failResult("MU-10")
		}
		if !hasOffset {
			return warnResult("MU-10", verdict.OutcomeFail)
		}
		return passResult("MU-10")
	default:
		return indeterminateResult("MU-10")
	}
}

// checkMU10EpochSeconds is MU-10's `encoding: epoch_s` branch.
//
//   - the value is a JSON string → FAIL (vector 53's epoch_ms mirror
//     applies identically here: "the two epoch encodings are symmetric on
//     this point").
//   - the value is a JSON number greater than 10^11 → FAIL (vector 18): "a
//     seconds value that large is year 5138; it is almost certainly
//     milliseconds."
//   - the value is a JSON number no greater than 10^11 → PASS (vector 17).
//   - neither a string nor a number → INDETERMINATE (vector 55's epoch_s
//     analogue; not separately vectored, but the same closed-evaluation
//     rule applies).
func checkMU10EpochSeconds(v field.Value) (verdict.Result, bool, error) {
	switch v.Kind() {
	case field.ValueKindString:
		return failResult("MU-10")
	case field.ValueKindNumber:
		n, _ := v.NumberValue()
		if n.Compare(epochMagnitudeThreshold()) > 0 {
			return failResult("MU-10")
		}
		return passResult("MU-10")
	default:
		return indeterminateResult("MU-10")
	}
}

// checkMU10EpochMillis is MU-10's `encoding: epoch_ms` branch.
//
//   - the value is a JSON string → FAIL (vector 53).
//   - the value is a JSON number strictly between 0 and 10^11 → FAIL
//     (vector 19): "a milliseconds value that small is before 1973;
//     almost certainly seconds."
//   - the value is a JSON number that is zero, negative, or at least
//     10^11 → PASS (vector 52). Zero and negative values are left to
//     MU-11's own bound checking, not this check's magnitude heuristic,
//     which SPEC-MU states only as "value < 10^11 and value > 0."
//   - neither a string nor a number → INDETERMINATE (mirrors checkMU10
//     EpochSeconds' own final branch).
func checkMU10EpochMillis(v field.Value) (verdict.Result, bool, error) {
	switch v.Kind() {
	case field.ValueKindString:
		return failResult("MU-10")
	case field.ValueKindNumber:
		n, _ := v.NumberValue()
		if n.Sign() > 0 && n.Compare(epochMagnitudeThreshold()) < 0 {
			return failResult("MU-10")
		}
		return passResult("MU-10")
	default:
		return indeterminateResult("MU-10")
	}
}

// isoNaiveLayout is the Go reference-time layout for an ISO 8601
// date-time carrying no timezone offset at all -- neither "Z" nor a
// "+HH:MM"/"-HH:MM" suffix. Go's time.Parse recognises an optional
// fractional-seconds field immediately after the seconds field even when
// the layout itself does not mention one (documented on time.Parse: "the
// input may contain a fractional second field immediately after the
// seconds field, even if the layout does not signify its presence"), so
// this one layout alone is sufficient to accept both "2026-08-05T14:22:07"
// and "2026-08-05T14:22:07.123456" without a second, fraction-specific
// layout.
const isoNaiveLayout = "2006-01-02T15:04:05"

// parseISO8601 parses s as a point in time under SPEC-MU's iso8601
// encoding, reporting whether it parsed at all (ok) and, if so, whether it
// carried an explicit timezone offset (hasOffset). SPEC-MU's own encoding
// is RFC 3339, which always carries an offset, but MU-10 must also
// recognise a *naive* timestamp -- one with no offset -- in order to flag
// it as an ambiguity at warn severity rather than reject it outright
// (SPEC-MU §5: "parses, but carries no timezone offset → FAIL at severity
// warn... A naive timestamp is an ambiguity, not an error"). MU-11
// (date_range_bound, daterange.go) uses this same parse to read a
// timestamp field's value as an instant regardless of which of the two
// shapes it took -- a naive string parses in UTC, since time.Parse
// defaults to UTC "in the absence of a time zone indicator," which is the
// only reading available when SPEC-MU itself does not name one.
//
// Trying the strict, offset-bearing layout first is deliberate: it can
// never also match a string with no offset (RFC3339Nano requires one), so
// there is no ordering hazard in checking it ahead of the naive layout.
func parseISO8601(s string) (t time.Time, hasOffset bool, ok bool) {
	if parsed, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return parsed, true, true
	}
	if parsed, err := time.Parse(isoNaiveLayout, s); err == nil {
		return parsed, false, true
	}
	return time.Time{}, false, false
}
