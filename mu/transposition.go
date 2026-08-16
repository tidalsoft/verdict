package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// mu22MagnitudeFactor is SPEC-MU §6 MU-22's own threshold: "the magnitude
// difference exceeds 5x".
func mu22MagnitudeFactor() decimal.Decimal { return mustParseDecimal("5") }

// checkMU22 implements the digit_transposition_suspect check (MU-22,
// SPEC-MU §6): flag a value that looks like a typographic slip against a
// prior value recorded for the same field and the same entity.
//
// Applicability (SPEC-MU §2.5.1: applies to `money`, `decimal`, with no
// further gate):
//   - no declaration for the field, or a declaration of a kind this check
//     does not apply to -> not applicable.
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS or FAIL:
//   - the value is not coercible (§2.6.3; MU-22 is value-dependent) ->
//     INDETERMINATE, reason value_not_coercible.
//   - the caller identifies no entity for the observation (Input.HasEntity
//     false) -> INDETERMINATE: "this check has no comparand" (vector
//     MU-V90).
//   - no entry in Input.Observations carries the same entity ->
//     INDETERMINATE: "returning PASS would report 'no likely
//     transposition' on the strength of having looked at nothing" (vector
//     MU-V89).
//   - the value matches one of SPEC-MU §6's three named patterns against
//     any same-entity prior value, and the magnitude difference exceeds
//     5x -> FAIL at warn (vector MU-V88: an extra trailing zero, exactly
//     10x). This is checked across *every* same-entity prior, not just
//     the first one that decides anything: a FAIL wins regardless of
//     where in Input.Observations that prior sits, and regardless of
//     whether an earlier prior's own comparison overflowed. A set of
//     priors is not an ordered input (SPEC-SYS §14.1: a verdict is a
//     function of the request and the reference tables, and Observations
//     is state the caller hands over, not a sequence whose order carries
//     meaning) -- returning on the first error, as an earlier version of
//     this function did, made the verdict depend on caller-supplied
//     ordering: the same two priors, in one order, produced INDETERMINATE
//     (the erroring prior examined first) and, reordered, FAIL (the
//     matching prior examined first). The fix below scans every prior
//     unconditionally: a FAIL from any of them wins outright, and only a
//     prior that neither matched nor errored is silent. Order no longer
//     has anything to say about the outcome.
//   - no prior produced a FAIL, and at least one prior's comparison
//     overflowed the exact decimal range -> INDETERMINATE, for the same
//     reason every other arithmetic failure on caller-supplied data in
//     this package is (see checkMU20's and checkMU21's own doc comments):
//     an inconclusive comparison is not evidence of anything, and must
//     not silently drop out of the FAIL-or-not decision.
//   - otherwise, every same-entity prior compared cleanly and none
//     matched -> PASS.
//
// # Implementation note: the pattern algorithm
//
// SPEC-MU §6 names three patterns -- "a single-digit transposition, a
// repeated digit, or an extra trailing zero" -- but does not formalize
// any of them; this is not one of the three spec gaps this task's brief
// names (undefined "observation," even-count median/quartiles, and
// window join timing), because there SPEC-MU is silent on a term it
// never defines at all, while here it names the patterns and leaves their
// precise algorithm as an implementation detail of a Class S heuristic
// check whose "Known false positives" section already says "High ...
// included for evaluation rather than production." digitPatternMatch
// below implements each of the three against the value's and the prior's
// digit sequences (sign and decimal point stripped): an extra trailing
// zero is one digit sequence equal to the other with "0" appended;
// a repeated digit is one sequence equal to the other with exactly one
// digit removed, where the removed digit duplicates a neighbour it now
// sits next to; a single-digit transposition is two equal-length
// sequences differing at exactly two adjacent positions whose characters
// are each other's. Only vector MU-V88 (extra trailing zero) is
// vector-tested; the other two patterns are exercised by tests in
// transposition_test.go this package wrote itself, since SPEC-MU
// publishes no vector for them.
func checkMU22(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	switch decl.(type) {
	case field.MoneyDeclaration, field.DecimalDeclaration:
	default:
		return notApplicable()
	}

	if in.ValueCoercionFailed {
		return statIndeterminateResult("MU-22")
	}
	if !in.HasEntity {
		return statIndeterminateResult("MU-22")
	}

	priors := matchingEntityObservations(in.Observations, in.Entity)
	if len(priors) == 0 {
		return statIndeterminateResult("MU-22")
	}

	// in.Value's digit string is computed once, outside the loop, and
	// passed to every comparison rather than recomputed per prior.
	// decimal.Decimal.String() renders plain-decimal form (SPEC-MU
	// §2.6.1), so its cost grows with digit count -- measured at low
	// single-digit milliseconds for a 100,000-digit value -- and
	// Input.Value is attacker-controlled with no digit-count bound the
	// specification imposes. Recomputing it once per prior against a
	// window of thousands turns one cheap render into thousands of them,
	// all producing the identical string, before a single prior is even
	// compared. This is the same cost problem the linear rewrite of
	// isRepeatedDigitInsertion removed, one level up: the fix there was
	// doing the expensive work once instead of on every candidate
	// position; this is doing it once instead of on every prior.
	valueDigits := digitsOnly(in.Value.String())

	// Every prior is scanned, never stopping at the first one that errors:
	// a later prior may still produce a definite FAIL, and skipping the
	// rest of the scan on the first overflow would make the verdict
	// depend on which order the caller happened to list Observations in
	// -- see this function's own doc comment for the exact scenario. A
	// FAIL from any prior wins outright; INDETERMINATE is the fallback
	// only when nothing FAILed and at least one comparison could not be
	// completed.
	sawError := false
	for _, prior := range priors {
		matched, err := suspectTransposition(in.Value, valueDigits, prior)
		if err != nil {
			sawError = true
			continue
		}
		if matched {
			return statFailResult("MU-22")
		}
	}
	if sawError {
		return statIndeterminateResult("MU-22")
	}
	return statPassResult("MU-22")
}

// matchingEntityObservations returns the Value of every entry in
// observations whose HasEntity is true and whose Entity equals entity --
// SPEC-MU §6's own restriction of MU-20's shared window "to those
// carrying the same entity." An entry with HasEntity false never matches
// any entity, including the empty string.
func matchingEntityObservations(observations []Observation, entity string) []decimal.Decimal {
	var matches []decimal.Decimal
	for _, obs := range observations {
		if obs.HasEntity && obs.Entity == entity {
			matches = append(matches, obs.Value)
		}
	}
	return matches
}

// suspectTransposition reports whether value looks like a typographic
// slip against prior: one of digitPatternMatch's three shapes, combined
// with SPEC-MU §6's own magnitude requirement, "the magnitude difference
// exceeds 5x" (vector MU-V88 is exactly 10x, well past this bound). Both
// conditions must hold, so a pattern match alone never FAILs.
//
// One consequence is a known specification defect, not intended behaviour:
// an adjacent-digit swap can change a value by at most 91/19, about 4.79,
// which never clears the 5x floor. So the single-digit transposition
// SPEC-MU §6 names can never FAIL on its own, at any digit length. See
// TestCheckMU22_AdjacentTranspositionPattern_MagnitudeTooSmall_Pass in
// transposition_test.go for the proof. The gap is recorded against
// SPEC-MU and is not fixed here.
//
// valueDigits is value's own digitsOnly(value.String()), computed once by
// checkMU22 outside its loop over priors and threaded through here rather
// than recomputed per call -- see checkMU22's own doc comment for why
// that hoist matters.
func suspectTransposition(value decimal.Decimal, valueDigits string, prior decimal.Decimal) (bool, error) {
	if !digitPatternMatch(value, valueDigits, prior) {
		return false, nil
	}
	return exceedsMagnitudeRatio(value, prior, mu22MagnitudeFactor())
}

// exceedsMagnitudeRatio reports whether the larger of a and b (by
// absolute value) exceeds factor times the smaller -- restated as a
// multiplication to avoid division, mirroring the same restatement in
// outlier.go and scaleshift.go.
func exceedsMagnitudeRatio(a, b, factor decimal.Decimal) (bool, error) {
	larger, smaller := a.Abs(), b.Abs()
	if smaller.Compare(larger) > 0 {
		larger, smaller = smaller, larger
	}
	bound, err := factor.Mul(smaller)
	if err != nil {
		return false, err
	}
	return larger.Compare(bound) > 0, nil
}

// digitPatternMatch reports whether valueDigits (value's own
// digitsOnly(value.String()), see suspectTransposition's doc comment for
// why it arrives precomputed) and prior's decimal-digit sequence fit any
// of SPEC-MU §6's three named patterns. Values of opposite sign never
// match: a sign flip is a different kind of error than a typographic
// slip within one number's digits, and excluding it here keeps a refund
// (-49.99) from being compared against a charge (49.99) as though one
// were a transposition of the other.
func digitPatternMatch(value decimal.Decimal, valueDigits string, prior decimal.Decimal) bool {
	if value.Sign() != prior.Sign() {
		return false
	}
	b := digitsOnly(prior.String())
	return isExtraTrailingZero(valueDigits, b) || isExtraTrailingZero(b, valueDigits) ||
		isRepeatedDigitInsertion(valueDigits, b) || isRepeatedDigitInsertion(b, valueDigits) ||
		isAdjacentTransposition(valueDigits, b)
}

// digitsOnly strips every byte of s that is not an ASCII digit --
// dropping a leading sign and the decimal point from decimal.Decimal's
// own String() output -- leaving the bare digit sequence the three
// pattern functions below compare.
func digitsOnly(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
	}
	return string(out)
}

// isExtraTrailingZero reports whether long is short with exactly one "0"
// appended at the end -- SPEC-MU §6's "an extra trailing zero" pattern,
// e.g. "123400" (1234.00) against "1234000" (12340.00), vector MU-V88.
func isExtraTrailingZero(long, short string) bool {
	return len(long) == len(short)+1 && long == short+"0"
}

// isRepeatedDigitInsertion reports whether long is short with exactly one
// digit inserted somewhere, where the inserted digit duplicates a
// neighbour it now sits next to in long -- SPEC-MU §6's "a repeated
// digit" pattern, e.g. "12234" against "1234" (a "2" inserted next to the
// "2" already there).
//
// This is a single linear pass, not the quadratic form it replaced (which
// built a full-length string, long[:i]+long[i+1:], at every candidate
// position i): a value at an extreme exponent renders to a ~100,000-digit
// String(), and the quadratic form measured over a second on a single
// value/prior pair at that length -- a real cost, not a theoretical one,
// since decimal.Decimal.String() (SPEC-MU §2.6.1's plain-decimal
// requirement) is exactly what feeds this function's inputs.
//
// The scan walks long and short together while their characters agree,
// which finds i, the first index at which they differ (or len(short), if
// they agree all the way through short and long's one extra character is
// the last one). If long is short with exactly one character inserted at
// all, skipping that one character must make the remainder line up
// exactly: long[i+1:] == short[i:]. That check is the whole verification
// -- it is what the old loop's long[:i]+long[i+1:] != short comparison
// established one candidate i at a time; here there is exactly one
// candidate, found without rebuilding a string per position.
//
// Only long[i]'s *left* neighbour is checked, never its right. This is
// not the asymmetric convenience the loop it replaces started from -- it
// is required by how i itself is found. Suppose long[i] also equalled
// long[i+1]. Because long[i+1:] == short[i:], their first characters
// agree wherever both are non-empty, so long[i+1] == short[i]. Combined
// with the supposition, that gives long[i] == short[i] -- contradicting
// i being the *first* index where long and short disagree, which is
// exactly how this scan chose i. So long[i] == long[i+1] can never hold
// at this i; only a left-neighbour check (long[i-1] == long[i]) can ever
// fire, and it is sufficient on its own: wherever a duplicate run in long
// is what caused the single-character difference, this scan's
// character-by-character agreement check consumes the run up to its
// *last* member before diverging, so i always lands there, with the
// character before it (part of the same run) equal to it.
func isRepeatedDigitInsertion(long, short string) bool {
	if len(long) != len(short)+1 {
		return false
	}
	i := 0
	for i < len(short) && long[i] == short[i] {
		i++
	}
	if long[i+1:] != short[i:] {
		return false
	}
	return i > 0 && long[i-1] == long[i]
}

// isAdjacentTransposition reports whether a and b are the same length and
// differ at exactly two positions, which are adjacent and hold each
// other's digit -- SPEC-MU §6's "a single-digit transposition" pattern,
// e.g. "1324" against "1234" (the "3" and "2" swapped).
func isAdjacentTransposition(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	first, second := -1, -1
	for i := 0; i < len(a); i++ {
		if a[i] == b[i] {
			continue
		}
		switch {
		case first == -1:
			first = i
		case second == -1:
			second = i
		default:
			return false // more than two differing positions
		}
	}
	if first == -1 || second == -1 || second != first+1 {
		return false
	}
	return a[first] == b[second] && a[second] == b[first]
}
