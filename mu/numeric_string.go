package mu

import (
	"strings"
	"unicode"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// checkMU09 implements the numeric_string_coercion check (MU-09, SPEC-MU
// §5).
//
// MU-09 rejects numeric strings whose value is ambiguous under locale
// variation.
//
// Applicability (SPEC-MU §2.5.1: applies to money, decimal, and
// percentage, gated on the value having arrived as a string):
//   - no declaration for the field, or a declaration whose kind is none of
//     money, decimal, or percentage → not applicable.
//   - the field's value did not arrive as a string (Input.RawValue is not
//     ValueKindString -- including when Input.HasRawValue is false, an
//     absent field) → not applicable. SPEC-MU §2.5.1: "MU-09's condition is
//     a property of the value that arrived and of nothing else" -- a
//     property of the request, decided by reading Input.RawValue directly,
//     never by Input.Value/Provenance/ValueCoercionFailed (§2.6.3's
//     coercion gate does not reach this check at all: MU-09 "is the report
//     of that coercion," §2.6.2, and cannot itself be suppressed by it).
//
// Branch matrix, once applicable -- classifyNumericString implements
// SPEC-MU §5's Evaluation clause; see its own doc comment for the full
// branch-by-branch mapping, including the one documented, disclosed
// judgment call this package makes where the specification itself leaves a
// term ("unambiguous") undefined.
func checkMU09(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	switch decl.(type) {
	case field.MoneyDeclaration, field.DecimalDeclaration, field.PercentageDeclaration:
		// applicable kinds; fall through
	default:
		return notApplicable()
	}

	s, isString := in.RawValue.StringValue()
	if !isString {
		return notApplicable()
	}

	switch classifyNumericString(s) {
	case verdict.OutcomePass:
		return passResult("MU-09")
	case verdict.OutcomeFail:
		return failResult("MU-09")
	default:
		return indeterminateResult("MU-09")
	}
}

// classifyNumericString implements SPEC-MU §5 MU-09's Evaluation clause
// against the field's raw transported text s, returning the Outcome MU-09
// reports (never a Result directly: checkMU09 is this function's one
// caller, and every other outcome this package could report -- not
// applicable, an error -- cannot arise from text classification alone).
//
// The clauses below are read in an order equivalent to, but not identical
// to, SPEC-MU's own listed order -- see the "parses cleanly" branch's own
// comment for why checking it first changes no vector's outcome.
//
//  1. s parses cleanly as decimal text (SPEC-MU §2.6.1's grammar, reused
//     here via decimal.Parse rather than a second hand-written grammar
//     that could drift from it) → PASS. This subsumes SPEC-MU's fourth
//     bullet ("String parses cleanly as a decimal") and can never
//     misfire ahead of the comma/currency branches below, because none of
//     the text those branches match (a comma, a currency symbol, a space)
//     is legal in decimal text at all -- a string containing one can never
//     reach this branch.
//  2. s contains at least one '.' and at least one ',' → classifyMixed
//     Separators (branch 1 of SPEC-MU's own list: "parse by position...
//     PASS only if unambiguous; otherwise FAIL").
//  3. s contains exactly one ',' and no '.' → classifySingleComma
//     (branches 2 and 3 of SPEC-MU's own list, the "exactly three
//     following digits" ambiguity and its converse).
//  4. s contains a currency symbol or any Unicode space (including a
//     non-breaking space) anywhere → FAIL (SPEC-MU's fifth bullet).
//  5. none of the above → INDETERMINATE ("abc" on a money field, vector
//     51: "not ambiguous under locale variation, which is the only thing
//     this check enumerates," so no enumerated condition matches, and
//     §2.1 forbids reaching PASS by exhausting conditions against a value
//     this check could not interpret at all).
func classifyNumericString(s string) verdict.Outcome {
	if _, err := decimal.Parse(s); err == nil {
		return verdict.OutcomePass
	}

	dotCount := strings.Count(s, ".")
	commaCount := strings.Count(s, ",")

	switch {
	case dotCount > 0 && commaCount > 0:
		return classifyMixedSeparators(s, dotCount, commaCount)
	case commaCount == 1 && dotCount == 0:
		if outcome, matched := classifySingleComma(s); matched {
			return outcome
		}
	}

	if containsCurrencyNoise(s) {
		return verdict.OutcomeFail
	}
	return verdict.OutcomeIndeterminate
}

// classifyMixedSeparators implements SPEC-MU §5 MU-09's first Evaluation
// bullet: "String contains both '.' and ',' → parse by position (last
// separator is the decimal) and PASS only if unambiguous; otherwise FAIL."
//
// # A disclosed reading of an undefined term
//
// SPEC-MU never defines "unambiguous" for this branch (unlike the
// following two bullets, which state an exact digit-count rule). The only
// conformance vector exercising this branch (MU-V14, "1.234,56" → PASS →
// 1234.56) is fully determined by "the last separator is the decimal"
// alone and does not exercise the FAIL side of this bullet at all, so it
// settles nothing about what "unambiguous" excludes.
//
// This implementation's reading, adopted because the specification gives
// no grammar to check against and this package's contract forbids
// inventing one silently: **unambiguous means exactly one '.' and exactly
// one ',' in the whole string.** Given exactly one of each, the last one
// (by byte position) is the decimal separator and the other is a grouping
// separator, which is removed before the candidate is re-parsed as decimal
// text -- this is not itself an extra requirement invented here, since a
// string with only one of each separator, correctly assigned, is decimal
// text once the grouping separator is stripped, by construction. Any
// repetition of either separator (two commas, two dots, or more) is
// treated as ambiguous and FAILs outright, without attempting a candidate
// parse: this package takes no position on what grouping convention such a
// string might follow, because SPEC-MU states none.
//
// A reviewer or the specification's maintainer may reasonably read
// "unambiguous" more broadly (e.g. permitting repeated grouping
// separators, such as "1.234.567,89"). That reading is not implemented
// here because it is not written anywhere in the specification either --
// implementing it would be exactly the invented reading this package's
// contract prohibits. This is reported as an open question in this task's
// own report, not resolved by this comment.
func classifyMixedSeparators(s string, dotCount, commaCount int) verdict.Outcome {
	if dotCount != 1 || commaCount != 1 {
		return verdict.OutcomeFail
	}

	dotIdx := strings.IndexByte(s, '.')
	commaIdx := strings.IndexByte(s, ',')

	var decimalSep, groupSep string
	if dotIdx > commaIdx {
		decimalSep, groupSep = ".", ","
	} else {
		decimalSep, groupSep = ",", "."
	}

	candidate := strings.ReplaceAll(s, groupSep, "")
	candidate = strings.ReplaceAll(candidate, decimalSep, ".")

	if _, err := decimal.Parse(candidate); err == nil {
		return verdict.OutcomePass
	}
	return verdict.OutcomeFail
}

// classifySingleComma implements SPEC-MU §5 MU-09's second and third
// Evaluation bullets, for a string containing exactly one ',' and no '.':
//
//   - "exactly one ',' with exactly three following digits ('1,234')" →
//     FAIL (matched == true, verdict.OutcomeFail): 1234 under English
//     convention, 1.234 under German, and no declared locale to resolve it
//     (vector 12).
//   - "one ',' with other than three following digits ('1,23')" → PASS
//     (matched == true, verdict.OutcomeFail is not returned; the comma is
//     read as the decimal separator) (vector 13).
//
// matched is false when s does not have the shape either bullet
// describes -- the text before the comma is empty or not an optionally-
// signed run of digits, or the text after it is empty or not a run of
// digits at all -- so the caller falls through to the currency/whitespace
// check and then, failing that, INDETERMINATE. SPEC-MU states both bullets
// only for a string that already looks like a number broken by one comma;
// this package does not extend "exactly three following digits" to cover
// text that is not shaped like a number in the first place, since nothing
// in the specification says to.
func classifySingleComma(s string) (verdict.Outcome, bool) {
	idx := strings.IndexByte(s, ',')
	before, after := s[:idx], s[idx+1:]

	if before == "" || !isSignedDigits(before) || after == "" || !isASCIIDigits(after) {
		return verdict.OutcomeIndeterminate, false
	}
	if len(after) == 3 {
		return verdict.OutcomeFail, true
	}
	return verdict.OutcomePass, true
}

// isASCIIDigits reports whether s is non-empty and consists entirely of
// ASCII digits.
func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// isSignedDigits reports whether s is an optional leading '+' or '-'
// followed by one or more ASCII digits.
func isSignedDigits(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '+' || s[0] == '-' {
		s = s[1:]
	}
	return isASCIIDigits(s)
}

// containsCurrencyNoise reports whether s contains a Unicode currency
// symbol (category Sc: "$", "€", "£", "¥", ...) or any Unicode space
// character, including U+00A0 NO-BREAK SPACE -- SPEC-MU §5's fifth
// Evaluation bullet, "String contains currency symbols, spaces, or
// non-breaking spaces → FAIL" (vector 15: "$49.99"). Using unicode.Sc and
// unicode.IsSpace, rather than a hand-picked symbol list, means this test
// tracks the Unicode Consortium's own currency-symbol assignments (and
// every whitespace code point Go's standard library already recognises,
// which includes U+00A0) without this package maintaining a duplicate,
// driftable enumeration of either set.
func containsCurrencyNoise(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) || unicode.Is(unicode.Sc, r) {
			return true
		}
	}
	return false
}
