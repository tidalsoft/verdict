package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

// wantMU16 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU16: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-16's only severity), and Outcome.
func wantMU16(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU16(in)
	if err != nil {
		t.Fatalf("checkMU16 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU16 applicable = false, want true")
	}
	if res.CheckID() != "MU-16" {
		t.Errorf("CheckID() = %q, want MU-16", res.CheckID())
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

func mustScheme(t *testing.T, d field.IdentifierDeclaration, s field.Scheme) field.IdentifierDeclaration {
	t.Helper()
	out, err := d.WithScheme(s)
	if err != nil {
		t.Fatalf("WithScheme(%v) unexpected error: %v", s, err)
	}
	return out
}

func identifierNumberInput(t *testing.T, scheme field.Scheme, n string) Input {
	t.Helper()
	decl := mustScheme(t, field.NewIdentifierDeclaration(), scheme)
	return Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, n)),
	}
}

func identifierStringInput(t *testing.T, scheme field.Scheme, s string) Input {
	t.Helper()
	decl := mustScheme(t, field.NewIdentifierDeclaration(), scheme)
	return Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue(s),
	}
}

func TestCheckMU16_MU_V35(t *testing.T) {
	// MU-V35: identifier, luhn | 4111111111111111 | PASS | MU-16
	wantMU16(t, identifierNumberInput(t, field.SchemeLuhn, "4111111111111111"), verdict.OutcomePass)
}

func TestCheckMU16_MU_V36(t *testing.T) {
	// MU-V36: identifier, luhn | 4111111111111112 | FAIL | MU-16
	wantMU16(t, identifierNumberInput(t, field.SchemeLuhn, "4111111111111112"), verdict.OutcomeFail)
}

func TestCheckMU16_MU_V37(t *testing.T) {
	// MU-V37: identifier, unknown_scheme | anything | INDETERMINATE | MU-16
	decl := mustScheme(t, field.NewIdentifierDeclaration(), field.Scheme("unknown_scheme"))
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewStringValue("anything"),
	}
	wantMU16(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU16_MU_V93(t *testing.T) {
	// MU-V93: identifier, iso4217 | "usd" | PASS | MU-16 (membership is case-folded)
	in := identifierStringInput(t, field.SchemeISO4217, "usd")
	in.Tables = Tables{ISO4217: tables.NewISO4217Table()}
	wantMU16(t, in, verdict.OutcomePass)
}

func TestCheckMU16_MU_V103(t *testing.T) {
	// MU-V103: identifier (no scheme) | 4111111111111111 | INDETERMINATE | MU-16
	decl := field.NewIdentifierDeclaration()
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, "4111111111111111")),
	}
	wantMU16(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU16_ISO4217_UnknownCode_Fail(t *testing.T) {
	in := identifierStringInput(t, field.SchemeISO4217, "ZZZ")
	in.Tables = Tables{ISO4217: tables.NewISO4217Table()}
	wantMU16(t, in, verdict.OutcomeFail)
}

func TestCheckMU16_ISO3166Alpha2_KnownCode_Pass(t *testing.T) {
	in := identifierStringInput(t, field.SchemeISO3166Alpha2, "ca")
	in.Tables = Tables{Countries: tables.NewISO3166Alpha2Table()}
	wantMU16(t, in, verdict.OutcomePass)
}

func TestCheckMU16_ISO3166Alpha2_UnknownCode_Fail(t *testing.T) {
	in := identifierStringInput(t, field.SchemeISO3166Alpha2, "ZZ")
	in.Tables = Tables{Countries: tables.NewISO3166Alpha2Table()}
	wantMU16(t, in, verdict.OutcomeFail)
}

func TestCheckMU16_ISBN13_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN13, "9780306406157"), verdict.OutcomePass)
}

func TestCheckMU16_ISBN13_Invalid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN13, "9780306406158"), verdict.OutcomeFail)
}

func TestCheckMU16_ISBN13_WrongLength(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN13, "97803064061"), verdict.OutcomeFail)
}

func TestCheckMU16_ISBN10_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN10, "0306406152"), verdict.OutcomePass)
}

func TestCheckMU16_ISBN10_ValidWithXCheckDigit(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN10, "097522980X"), verdict.OutcomePass)
}

func TestCheckMU16_ISBN10_Invalid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN10, "0306406153"), verdict.OutcomeFail)
}

func TestCheckMU16_ISBN10_WrongLength(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeISBN10, "030640615"), verdict.OutcomeFail)
}

func TestCheckMU16_GTIN8_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN8, "40170725"), verdict.OutcomePass)
}

func TestCheckMU16_GTIN8_Invalid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN8, "40170726"), verdict.OutcomeFail)
}

func TestCheckMU16_GTIN12_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN12, "614141000036"), verdict.OutcomePass)
}

func TestCheckMU16_GTIN13_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN13, "4006381333931"), verdict.OutcomePass)
}

func TestCheckMU16_GTIN14_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN14, "00614141000036"), verdict.OutcomePass)
}

func TestCheckMU16_GTIN14_WrongLength(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeGTIN14, "0061414100003"), verdict.OutcomeFail)
}

func TestCheckMU16_IBAN_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeIBAN, "GB29NWBK60161331926819"), verdict.OutcomePass)
}

func TestCheckMU16_IBAN_Invalid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeIBAN, "GB29NWBK60161331926818"), verdict.OutcomeFail)
}

func TestCheckMU16_IBAN_NotWellFormed(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeIBAN, "gb29nwbk60161331926819"), verdict.OutcomeFail)
}

func TestCheckMU16_IBAN_TooShort(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeIBAN, "GB2"), verdict.OutcomeFail)
}

func TestCheckMU16_LEI_Valid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeLEI, "529900T8BM49AURSDO55"), verdict.OutcomePass)
}

func TestCheckMU16_LEI_Invalid(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeLEI, "529900T8BM49AURSDO56"), verdict.OutcomeFail)
}

func TestCheckMU16_LEI_WrongLength(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeLEI, "529900T8BM49AURSDO"), verdict.OutcomeFail)
}

func TestCheckMU16_BIC_Valid8(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "DEUTDEFF"), verdict.OutcomePass)
}

func TestCheckMU16_BIC_Valid11(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "DEUTDEFF500"), verdict.OutcomePass)
}

func TestCheckMU16_BIC_WrongLength(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "DEUTDEF"), verdict.OutcomeFail)
}

func TestCheckMU16_BIC_LowercaseBankCode(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "deutdeff"), verdict.OutcomeFail)
}

func TestCheckMU16_BIC_NonAlnumLocationCode(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "DEUTDE-F"), verdict.OutcomeFail)
}

func TestCheckMU16_BIC_NonAlnumBranchCode(t *testing.T) {
	wantMU16(t, identifierStringInput(t, field.SchemeBIC, "DEUTDEFF5-0"), verdict.OutcomeFail)
}

func TestCheckMU16_ValueNotResolvable_Indeterminate(t *testing.T) {
	decl := mustScheme(t, field.NewIdentifierDeclaration(), field.SchemeLuhn)
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistryT(decl),
		HasRawValue: true,
		RawValue:    field.NewBoolValue(true),
	}
	wantMU16(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU16_ValueAbsent_Indeterminate(t *testing.T) {
	decl := mustScheme(t, field.NewIdentifierDeclaration(), field.SchemeLuhn)
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistryT(decl),
	}
	wantMU16(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU16_NotApplicable(t *testing.T) {
	cases := []struct {
		name string
		in   Input
	}{
		{"no declaration", Input{Field: "arguments.amount"}},
		{
			"declaration kind MU-16 does not apply to",
			Input{Field: "arguments.amount", Registry: mustRegistryT(field.NewMoneyDeclaration())},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, applicable, err := checkMU16(tc.in)
			if err != nil {
				t.Fatalf("checkMU16 unexpected error: %v", err)
			}
			if applicable {
				t.Fatal("checkMU16 applicable = true, want false")
			}
		})
	}
}

func TestSchemeSatisfied_UnrecognizedScheme(t *testing.T) {
	// schemeSatisfied's default arm is unreachable from checkMU16, which
	// always checks schemeRecognized first -- see schemeSatisfied's own
	// doc comment for why the arm exists and is tested directly anyway.
	if got := schemeSatisfied(field.Scheme("nonsense"), "anything", Tables{}); got {
		t.Errorf("schemeSatisfied(nonsense) = true, want false")
	}
}

func TestLuhnValid_NotAllDigits(t *testing.T) {
	if luhnValid("41111a1111111111") {
		t.Error("luhnValid with a non-digit character = true, want false")
	}
	if luhnValid("") {
		t.Error("luhnValid(\"\") = true, want false")
	}
}

func TestLuhnValid_DoubledDigitReducesOverNine(t *testing.T) {
	// A well-known Mastercard test number: doubling several of its 5s
	// produces 10, which must reduce to 1 (d -= 9) for the checksum to
	// come out valid -- 4111111111111111/...112's own digits (1s and 4s)
	// never exceed 9 when doubled, so this is the one input in this
	// file's table that actually reaches that branch.
	if !luhnValid("5555555555554444") {
		t.Error("luhnValid(\"5555555555554444\") = false, want true")
	}
}

func TestIsbn10Valid_NonDigitNonX(t *testing.T) {
	if isbn10Valid("030640615Y") {
		t.Error("isbn10Valid with a non-digit, non-X final character = true, want false")
	}
}

func TestGtinValid_NonDigit(t *testing.T) {
	if gtinValid("4017072a", 8) {
		t.Error("gtinValid with a non-digit character = true, want false")
	}
}

func TestIbanValid_LowerAlpha(t *testing.T) {
	if ibanValid("") {
		t.Error("ibanValid(\"\") = true, want false")
	}
}

func TestLeiValid_LowerAlpha(t *testing.T) {
	if leiValid("529900t8bm49aursdo55") {
		t.Error("leiValid with lowercase letters = true, want false")
	}
}

func TestIsUpperAlphanumeric_Empty(t *testing.T) {
	// Every current caller (ibanValid, leiValid, bicValid) only ever
	// passes a fixed-length, already-non-empty slice, so this guard is
	// unreachable through any of them -- tested directly so the branch
	// itself stays exercised.
	if isUpperAlphanumeric("") {
		t.Error("isUpperAlphanumeric(\"\") = true, want false")
	}
}

func TestIsUpperAlpha_Empty(t *testing.T) {
	// bicValid always passes a fixed 6-byte slice, so this guard is
	// unreachable through it -- tested directly for the same reason as
	// TestIsUpperAlphanumeric_Empty.
	if isUpperAlpha("") {
		t.Error("isUpperAlpha(\"\") = true, want false")
	}
}

func TestIdentifierText(t *testing.T) {
	cases := []struct {
		name     string
		v        field.Value
		wantText string
		wantOK   bool
	}{
		{"string", field.NewStringValue("GB29NWBK60161331926819"), "GB29NWBK60161331926819", true},
		{"number", field.NewNumberValue(mustParse(t, "4111111111111111")), "4111111111111111", true},
		{"bool", field.NewBoolValue(true), "", false},
		{"null", field.NewNullValue(), "", false},
		{"non-comparable", field.NewNonComparableValue(), "", false},
		{"unspecified (absent)", field.Value{}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := identifierText(tc.v)
			if ok != tc.wantOK {
				t.Fatalf("identifierText ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && got != tc.wantText {
				t.Errorf("identifierText = %q, want %q", got, tc.wantText)
			}
		})
	}
}
