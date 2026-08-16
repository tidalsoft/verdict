package mu

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU11 implements the date_range_bound check (MU-11, SPEC-MU §5).
//
// MU-11 rejects dates outside a declared plausible window.
//
// Applicability (SPEC-MU §2.5.1: applies to timestamp, gated on
// `not_before` or `not_after` being declared):
//   - no declaration for the field, or a declaration whose kind is not
//     timestamp → not applicable.
//   - neither not_before nor not_after declared → not applicable (§2.5.2:
//     a field declaring neither bound is unbounded, a coherent, complete
//     declaration).
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS:
//   - encoding is not declared → INDETERMINATE (vector 99): this check
//     reads the value as a point in time, and a bare number is a
//     different instant under epoch_s than under epoch_ms, so no window
//     comparison is possible without knowing which.
//   - the value does not parse under the declared encoding →
//     INDETERMINATE (vector 73): "a window comparison against something
//     that is not a time is not a comparison." Whether the value was
//     *supposed* to be a time in that encoding is MU-10's report, answered
//     independently.
//   - a declared bound is relative ("now"-prefixed, SPEC-MU §2.4.2's
//     grammar) and the request carries no evaluated_at to resolve it
//     against → INDETERMINATE (vector 100): "there is no clock to fall
//     back on."
//   - the resolved value falls outside either declared, resolved bound →
//     FAIL (vector 71). Bounds are inclusive.
//   - the resolved value falls within every declared, resolved bound →
//     PASS (vector 72).
//
// MU-11 is not value-dependent (SPEC-MU §2.6.3's table: no value-dependent
// check is ever applicable to a `timestamp` field). It reads Input.RawValue
// directly, exactly as MU-10 does (see checkMU10's own doc comment), and
// additionally Input.EvaluatedAt/Input.HasEvaluatedAt (see Input's own doc
// comment) wherever a declared bound is relative.
func checkMU11(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	tsDecl, ok := decl.(field.TimestampDeclaration)
	if !ok {
		return notApplicable()
	}

	notBefore, hasNotBefore := tsDecl.NotBefore()
	notAfter, hasNotAfter := tsDecl.NotAfter()
	if !hasNotBefore && !hasNotAfter {
		return notApplicable()
	}

	encoding, hasEncoding := tsDecl.Encoding()
	if !hasEncoding {
		return indeterminateResult("MU-11")
	}

	instant, ok := parseTimestampValue(in.RawValue, encoding)
	if !ok {
		return indeterminateResult("MU-11")
	}

	if hasNotBefore {
		bound, ok := resolveTimeBound(notBefore, in.EvaluatedAt, in.HasEvaluatedAt)
		if !ok {
			return indeterminateResult("MU-11")
		}
		if instant.Before(bound) {
			return failResult("MU-11")
		}
	}
	if hasNotAfter {
		bound, ok := resolveTimeBound(notAfter, in.EvaluatedAt, in.HasEvaluatedAt)
		if !ok {
			return indeterminateResult("MU-11")
		}
		if instant.After(bound) {
			return failResult("MU-11")
		}
	}
	return passResult("MU-11")
}

// parseTimestampValue reads v as a point in time under encoding, the
// declared field.Encoding, returning ok == false wherever v's own JSON
// shape or content does not match what that encoding requires -- a string
// under an epoch encoding, a non-finite-integer number under either epoch
// encoding, or a string that fails ISO 8601 parsing under
// field.EncodingISO8601 (via parseISO8601, timestamp.go). A naive ISO 8601
// string -- one with no timezone offset -- is a value MU-11 can still
// place in a window (it is read in UTC, per parseISO8601's own doc
// comment); MU-10's own, separate opinion about the missing offset does
// not change what MU-11 does with it.
func parseTimestampValue(v field.Value, encoding field.Encoding) (time.Time, bool) {
	if encoding == field.EncodingISO8601 {
		s, ok := v.StringValue()
		if !ok {
			return time.Time{}, false
		}
		t, _, ok := parseISO8601(s)
		return t, ok
	}

	// A fractional epoch value is treated as not parsing under an epoch
	// encoding, which is a deliberate choice on a point SPEC-MU does not
	// settle rather than something the document states. MU-11's
	// *Indeterminate when* turns on "the value does not parse under the
	// declared encoding", and whether 1754404927.5 parses as epoch seconds
	// has two defensible answers: it denotes a real instant half a second
	// after an integer one, or an epoch encoding denotes whole units and a
	// fractional value is malformed. MU-10 enumerates only the string and
	// magnitude cases and so would evaluate the same value without
	// objection, which makes the two checks disagree about one input.
	// INDETERMINATE is the safe side of that disagreement: it reports that
	// the bound could not be evaluated instead of risking a PASS on a
	// window comparison whose meaning is unsettled. Revisit once SPEC-MU
	// states whether an epoch encoding admits a fractional value.
	n, ok := v.NumberValue()
	if !ok || n.DecimalPlaces() > 0 {
		return time.Time{}, false
	}
	i, err := strconv.ParseInt(n.String(), 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	if encoding == field.EncodingEpochSeconds {
		return time.Unix(i, 0).UTC(), true
	}
	// encoding == field.EncodingEpochMillis: checkMU11's only caller
	// already required Encoding() to have resolved to one of the three
	// defined values, and the iso8601 branch above has already returned,
	// so this is the only remaining possibility.
	return time.UnixMilli(i).UTC(), true
}

// resolveTimeBound resolves a declared `not_before`/`not_after` bound --
// either an absolute RFC 3339 timestamp, or a value relative to
// evaluatedAt under SPEC-MU §2.4.2's grammar -- into the time.Time MU-11
// compares the field's own instant against. ok is false wherever the bound
// cannot be resolved: a relative bound with hasEvaluatedAt false (vector
// 100, "there is no clock to fall back on"), a malformed relative bound
// (SPEC-MU §2.4.2's grammar rejects it), or an absolute bound that does not
// parse as RFC 3339 -- the last of these is not vector-tested (every
// conformance vector's declared bound is well-formed), but is the same
// closed-evaluation rule applied symmetrically to the bound's own text as
// to the field's value.
func resolveTimeBound(bound string, evaluatedAt time.Time, hasEvaluatedAt bool) (time.Time, bool) {
	if strings.HasPrefix(bound, "now") {
		if !hasEvaluatedAt {
			return time.Time{}, false
		}
		dur, ok := parseRelativeDuration(bound)
		if !ok {
			return time.Time{}, false
		}
		return evaluatedAt.Add(dur), true
	}
	t, err := time.Parse(time.RFC3339Nano, bound)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// The addition above needs no overflow guard of its own, and an earlier
// draft's monotonicity check on the result was removed rather than left
// uncovered. parseRelativeDuration rejects any bound whose nanosecond
// count does not fit in an int64, so dur is bounded by roughly 292 years
// either way, while time.Time spans on the order of 1e11 years. Go's
// time.Time.Add also saturates rather than wrapping. Between the two,
// there is no evaluatedAt this package can receive for which the sum is
// not a real instant -- so a post-hoc check would be a branch no test
// could reach, which the coverage bar forbids and which would in any case
// be guarding against the wrong thing: the defect here was never the
// addition, it was the multiplication that fed it.

// parseRelativeDuration parses bound against SPEC-MU §2.4.2's relative
// time bound grammar: "`now`, optionally followed by `+` or `-`, an
// unsigned integer, and one of `s` (seconds), `m` (minutes), `h` (hours),
// `d` (days), `w` (7 days), or `y` (365 days)." This grammar is complete
// and normative in §2.4.2 -- MU-11's own section states only two
// illustrative examples ("now-5y", "now+90d") without repeating it, which
// is not the same as the grammar being absent from the specification.
//
// "now" alone (bound == "now", vector 71) is the zero duration. There is
// no calendar arithmetic and no month unit, per §2.4.2's own text, so
// every unit here is a fixed multiple of time.Duration with no leap
// adjustment of any kind.
func parseRelativeDuration(bound string) (time.Duration, bool) {
	rest := strings.TrimPrefix(bound, "now")
	if rest == "" {
		return 0, true
	}
	if len(rest) < 3 { // sign + at least one digit + a unit letter
		return 0, false
	}

	sign := rest[0]
	if sign != '+' && sign != '-' {
		return 0, false
	}
	unit := rest[len(rest)-1]
	digits := rest[1 : len(rest)-1]
	if !isASCIIDigits(digits) {
		return 0, false
	}
	n, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, false
	}

	unitDuration, ok := relativeTimeUnit(unit)
	if !ok {
		return 0, false
	}

	// n is non-negative (digits carry no sign), so the product overflows
	// exactly when n exceeds MaxInt64 divided by the unit's nanosecond
	// count. Checking before multiplying rather than inspecting the result
	// afterwards is the only reliable order: a wrapped time.Duration is a
	// perfectly ordinary negative duration, indistinguishable after the
	// fact from one the author asked for. Checking the resolved instant
	// afterwards cannot catch it either, because a wrapped "+" bound moves
	// the instant consistently backwards and so looks entirely ordinary.
	// Left unchecked, "now+300y"
	// resolves to a date in 1741 and MU-11 then FAILs timestamps that
	// should pass -- a confidently wrong verdict rather than the
	// INDETERMINATE an unresolvable bound owes the caller (vector MU-V100).
	if n > int64(math.MaxInt64)/int64(unitDuration) {
		return 0, false
	}

	dur := time.Duration(n) * unitDuration
	if sign == '-' {
		dur = -dur
	}
	return dur, true
}

// relativeTimeUnit maps SPEC-MU §2.4.2's five relative-time-bound unit
// letters to a fixed time.Duration -- "w" is exactly 7 days and "y" is
// exactly 365 days, per that section's own definitions, never a calendar
// week or year.
func relativeTimeUnit(unit byte) (time.Duration, bool) {
	switch unit {
	case 's':
		return time.Second, true
	case 'm':
		return time.Minute, true
	case 'h':
		return time.Hour, true
	case 'd':
		return 24 * time.Hour, true
	case 'w':
		return 7 * 24 * time.Hour, true
	case 'y':
		return 365 * 24 * time.Hour, true
	default:
		return 0, false
	}
}
