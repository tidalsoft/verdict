package mu

import (
	"strconv"
	"strings"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU16 implements the identifier_checksum check (MU-16, SPEC-MU §5).
//
// MU-16 validates check digits on structured identifiers appearing in
// numeric or string fields, or membership in a controlled list.
//
// Applicability (SPEC-MU §2.5.1: applies to identifier, no further gate):
//   - no declaration for the field, or a declaration whose kind is not
//     identifier → not applicable.
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS or FAIL:
//   - scheme is not declared → INDETERMINATE (vector 103): "nothing was
//     asked."
//   - the declared scheme is not one this package recognises →
//     INDETERMINATE, never PASS (vector 37): "nothing was validated, and
//     saying so is the whole point."
//   - the field's value is not a JSON string or a JSON number (including
//     Input.HasRawValue false, an absent value) → INDETERMINATE. SPEC-MU
//     names no outcome for a value it cannot even read as a candidate
//     identifier, and §2.1 forbids reaching PASS by exhausting conditions
//     against an uninterpretable value; this closed-evaluation rule
//     applies here symmetrically with every other check in this package.
//   - the well-formed-and-valid test for the declared scheme succeeds →
//     PASS (vectors 35, 93).
//   - the value is not well-formed for the declared scheme (wrong length,
//     characters the scheme does not admit) or fails its checksum → FAIL,
//     never INDETERMINATE (vector 36): SPEC-MU §5 is explicit that this is
//     "a violation and not an absence of evidence."
//
// MU-16 is not value-dependent (SPEC-MU §2.6.3's table): it reads
// Input.RawValue directly, as text, never Input.Value/Provenance/
// ValueCoercionFailed -- an identifier is not a magnitude, and SPEC-MU's
// coercion pipeline (§2.6.2) exists to turn text into an exact decimal,
// which is not the question this check asks.
func checkMU16(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	idDecl, ok := decl.(field.IdentifierDeclaration)
	if !ok {
		return notApplicable()
	}

	scheme, hasScheme := idDecl.Scheme()
	if !hasScheme {
		return indeterminateResult("MU-16")
	}
	if !schemeRecognized(scheme) {
		return indeterminateResult("MU-16")
	}

	text, ok := identifierText(in.RawValue)
	if !ok {
		return indeterminateResult("MU-16")
	}

	if schemeSatisfied(scheme, text, in.Tables) {
		return passResult("MU-16")
	}
	return failResult("MU-16")
}

// identifierText extracts the candidate identifier text from v: v's own
// string, when v is a JSON string, or v's exact decimal text (never a
// float64 detour -- decimal.Decimal.String() renders the same plain-digit
// text SPEC-MU §2.6.1 requires evaluation to use throughout), when v is a
// JSON number (vectors 35, 36, 103 all transport a card number this way).
// ok is false for every other JSON shape, and for the zero field.Value a
// caller reports via Input.HasRawValue == false -- there is no text to
// extract from a boolean, an explicit null, a sequence, another JSON
// array, or a JSON object, and none of them is a shape any scheme in this
// document defines characters for.
func identifierText(v field.Value) (string, bool) {
	switch v.Kind() {
	case field.ValueKindString:
		return v.StringValue()
	case field.ValueKindNumber:
		n, _ := v.NumberValue()
		return n.String(), true
	default:
		return "", false
	}
}

// schemeRecognized reports whether scheme is one of SPEC-MU §5's
// "Supported schemes and algorithms." It is checked, and must return true,
// before schemeSatisfied is ever called with the same scheme -- see that
// function's own doc comment for why its own default case is otherwise
// unreachable from checkMU16.
func schemeRecognized(scheme field.Scheme) bool {
	switch scheme {
	case field.SchemeIBAN, field.SchemeLuhn, field.SchemeISBN13, field.SchemeISBN10,
		field.SchemeGTIN8, field.SchemeGTIN12, field.SchemeGTIN13, field.SchemeGTIN14,
		field.SchemeLEI, field.SchemeBIC, field.SchemeISO4217, field.SchemeISO3166Alpha2:
		return true
	default:
		return false
	}
}

// schemeSatisfied reports whether text is well-formed for, and passes,
// scheme's own checksum or membership test. checkMU16 calls this only
// after schemeRecognized has already confirmed scheme is one of the
// twelve defined schemes, so this function's own default arm is never
// reached from checkMU16 -- it exists, and is tested directly
// (TestSchemeSatisfied_UnrecognizedScheme), so that this switch's shape
// mirrors schemeRecognized's exactly rather than silently drifting if a
// scheme is ever added to one and not the other; a future omission from
// schemeRecognized would still route here to a defined, deliberate
// answer (false) rather than an unhandled case.
func schemeSatisfied(scheme field.Scheme, text string, tbl Tables) bool {
	switch scheme {
	case field.SchemeIBAN:
		return ibanValid(text)
	case field.SchemeLuhn:
		return luhnValid(text)
	case field.SchemeISBN13:
		return isbn13Valid(text)
	case field.SchemeISBN10:
		return isbn10Valid(text)
	case field.SchemeGTIN8:
		return gtinValid(text, 8)
	case field.SchemeGTIN12:
		return gtinValid(text, 12)
	case field.SchemeGTIN13:
		return gtinValid(text, 13)
	case field.SchemeGTIN14:
		return gtinValid(text, 14)
	case field.SchemeLEI:
		return leiValid(text)
	case field.SchemeBIC:
		return bicValid(text)
	case field.SchemeISO4217:
		_, found := tbl.resolveCurrency(text)
		return found
	case field.SchemeISO3166Alpha2:
		_, found := tbl.resolveCountry(text)
		return found
	default:
		return false
	}
}

// isAllASCIIDigits reports whether s is non-empty and consists entirely of
// ASCII digits '0'-'9'. Every checksum scheme below needs exactly this
// shape test on its own payload before computing a checksum at all -- a
// non-digit character is "not well-formed," SPEC-MU §5's own phrase for
// the FAIL (not INDETERMINATE) outcome such a value gets.
func isAllASCIIDigits(s string) bool {
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

// luhnValid implements the Luhn checksum (ISO/IEC 7812), MU-16's `luhn`
// scheme: double every second digit counting from the rightmost, summing
// the digits of any result over 9, and require the total sum to be a
// multiple of 10. text must be non-empty and consist entirely of ASCII
// digits; anything else is not well-formed and fails outright, without
// attempting the checksum arithmetic (vectors 35, 36 both transport a
// 16-digit card number this way).
func luhnValid(text string) bool {
	if !isAllASCIIDigits(text) {
		return false
	}
	sum := 0
	parity := len(text) % 2
	for i := 0; i < len(text); i++ {
		d := int(text[i] - '0')
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

// isbn13Valid implements ISBN-13's weighted modulus-10 checksum: text must
// be exactly 13 ASCII digits, and the sum of each digit times an
// alternating weight of 1 and 3 (starting at 1 for the leftmost digit)
// must be a multiple of 10.
func isbn13Valid(text string) bool {
	if len(text) != 13 || !isAllASCIIDigits(text) {
		return false
	}
	sum := 0
	for i := 0; i < 13; i++ {
		weight := 1
		if i%2 == 1 {
			weight = 3
		}
		sum += int(text[i]-'0') * weight
	}
	return sum%10 == 0
}

// isbn10Valid implements ISBN-10's weighted modulus-11 checksum: text must
// be exactly 10 characters, the first nine ASCII digits and the tenth
// either an ASCII digit or the uppercase letter 'X' (standing for the
// value 10). Each position's value is weighted by 10 down to 1 from left
// to right, and the sum must be a multiple of 11.
func isbn10Valid(text string) bool {
	if len(text) != 10 {
		return false
	}
	sum := 0
	for i := 0; i < 10; i++ {
		c := text[i]
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c == 'X' && i == 9:
			d = 10
		default:
			return false
		}
		sum += d * (10 - i)
	}
	return sum%11 == 0
}

// gtinValid implements the GS1 modulus-10 check-digit algorithm shared by
// MU-16's gtin8/gtin12/gtin13/gtin14 schemes: text must be exactly length
// ASCII digits, and the rightmost digit is the check digit. Every digit to
// its left is weighted 3, 1, 3, 1, ... starting from the digit
// immediately to the check digit's own left (the standard GS1 "weight 3
// from the right, excluding the check digit itself" rule); the check
// digit must equal (10 - sum mod 10) mod 10.
func gtinValid(text string, length int) bool {
	if len(text) != length || !isAllASCIIDigits(text) {
		return false
	}
	payload := text[:length-1]
	checkDigit := int(text[length-1] - '0')

	sum := 0
	weight := 3
	for i := len(payload) - 1; i >= 0; i-- {
		sum += int(payload[i]-'0') * weight
		if weight == 3 {
			weight = 1
		} else {
			weight = 3
		}
	}
	computed := (10 - sum%10) % 10
	return computed == checkDigit
}

// mod97 computes digits mod 97, one digit at a time -- digits may be far
// larger than fits in any fixed-width integer type (an IBAN's rearranged
// numeric form routinely runs to 30+ digits), so this never materialises
// the full number, only ever the running remainder, which never exceeds
// 96. digits must be non-empty and consist entirely of ASCII digits;
// ibanValid and leiValid, this function's only two callers, both
// guarantee that before calling it.
func mod97(digits string) int {
	remainder := 0
	for i := 0; i < len(digits); i++ {
		remainder = (remainder*10 + int(digits[i]-'0')) % 97
	}
	return remainder
}

// alphanumericToDigits renders s -- already known to consist entirely of
// uppercase ASCII letters and digits -- as the numeric string ISO 7064's
// mod-97-10 check consumes: each digit renders as itself, and each letter
// renders as its position in the alphabet plus 10 (A=10, B=11, ..., Z=35),
// exactly as ISO 13616 (IBAN) and ISO 17442 (LEI) both define. ibanValid
// and leiValid are this function's only two callers, and both validate
// their input's alphabet before calling it.
func alphanumericToDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			b.WriteByte(c)
			continue
		}
		b.WriteString(strconv.Itoa(int(c-'A') + 10))
	}
	return b.String()
}

// isUpperAlphanumeric reports whether s consists entirely of uppercase
// ASCII letters and digits. SPEC-MU §5 states that the checksum schemes'
// "alphabets are defined by the standards that own them," unlike iso4217
// and iso3166_alpha2, which this package explicitly case-folds (see
// Tables.resolveCurrency's and Tables.resolveCountry's own doc comments)
// -- IBAN and LEI's own standards define an uppercase-only alphabet, so a
// lowercase letter here is not well-formed and fails, rather than being
// folded to match.
func isUpperAlphanumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// ibanValid implements ISO 13616's mod-97-10 check for MU-16's `iban`
// scheme: text must be 5 to 34 characters of uppercase ASCII letters and
// digits; the check moves the first four characters (the country code and
// two check digits) to the end, converts every letter to two digits
// (alphanumericToDigits), and requires the resulting number, mod 97, to
// equal 1.
func ibanValid(text string) bool {
	if len(text) < 5 || len(text) > 34 || !isUpperAlphanumeric(text) {
		return false
	}
	rearranged := text[4:] + text[:4]
	return mod97(alphanumericToDigits(rearranged)) == 1
}

// leiValid implements ISO 17442's mod-97-10 check for MU-16's `lei`
// scheme: text must be exactly 20 characters of uppercase ASCII letters
// and digits (18 identifier characters plus 2 numeric check digits, per
// ISO 17442); unlike IBAN, no rearrangement is performed -- the two check
// digits are already at the end -- and the resulting number, mod 97, must
// equal 1.
func leiValid(text string) bool {
	if len(text) != 20 || !isUpperAlphanumeric(text) {
		return false
	}
	return mod97(alphanumericToDigits(text)) == 1
}

// bicValid implements ISO 9362's structural validation for MU-16's `bic`
// scheme -- SPEC-MU §5 states explicitly that this scheme carries "no
// check digit," so this function tests shape alone: exactly 8 or 11
// characters; the first 4 (the bank code) and the following 2 (the country
// code) uppercase ASCII letters; the next 2 (the location code), and the
// trailing 3 when present (the branch code), uppercase ASCII letters or
// digits.
func bicValid(text string) bool {
	if len(text) != 8 && len(text) != 11 {
		return false
	}
	if !isUpperAlpha(text[0:6]) {
		return false
	}
	if !isUpperAlphanumeric(text[6:8]) {
		return false
	}
	if len(text) == 11 {
		return isUpperAlphanumeric(text[8:11])
	}
	return true
}

// isUpperAlpha reports whether s consists entirely of uppercase ASCII
// letters. bicValid's one caller uses this for BIC's bank-code and
// country-code segments, which ISO 9362 defines as letters only, unlike
// the location and branch code segments, which admit digits too
// (isUpperAlphanumeric).
func isUpperAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
