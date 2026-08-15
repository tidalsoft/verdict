package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

// wantMU02 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU02: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-02's only severity), and Outcome.
func wantMU02(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU02 applicable = false, want true")
	}
	if res.CheckID() != "MU-02" {
		t.Errorf("CheckID() = %q, want MU-02", res.CheckID())
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

func TestCheckMU02_MU_V9(t *testing.T) {
	// MU-V9: money, decimal string | "49.99" | PASS | MU-02
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "49.99"),
		Provenance: decimal.FromString,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU02(t, in, verdict.OutcomePass)
}

func TestCheckMU02_MU_V10(t *testing.T) {
	// MU-V10: money, JSON number | 0.1 | FAIL | MU-02 (not exactly
	// representable in binary64)
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU02(t, in, verdict.OutcomeFail)
}

func TestCheckMU02_MU_V11(t *testing.T) {
	// MU-V11: money, JSON number | 9007199254740993 | FAIL | MU-02
	// (> 2^53-1)
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "9007199254740993"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU02(t, in, verdict.OutcomeFail)
}

func TestCheckMU02_JSONNumber_SafeIntegerBoundary_Pass(t *testing.T) {
	// 2^53-1 itself is exactly representable and does not exceed the safe
	// integer ceiling -- PASS, not FAIL, distinguishing this boundary from
	// MU-V11's one-more-than-the-ceiling case.
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "9007199254740991"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU02 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU02_JSONNumber_ExactInteger_Pass(t *testing.T) {
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "4999"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU02 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU02_DecimalKind_JSONNumber(t *testing.T) {
	// kind: decimal is precision-sensitive too, and behaves identically to
	// kind: money -- MU-02 does not distinguish between them.
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewDecimalDeclaration()),
	}
	res, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU02 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU02_DecimalKind_StringAlwaysPasses(t *testing.T) {
	// A value that would fail as a JSON number passes when it arrived as a
	// decimal string -- provenance, not magnitude, is what MU-02 gates on.
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromString,
		Registry:   mustRegistry(t, field.NewDecimalDeclaration()),
	}
	res, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU02 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU02_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.1"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU02 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU02_WrongKind_NotApplicable(t *testing.T) {
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewPercentageDeclaration()),
	}
	_, applicable, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU02 applicable = true, want false (wrong kind)")
	}
}

func TestCheckMU02_MU_V44(t *testing.T) {
	// MU-V44: money | "1,234" (resolution refused, not coercible) |
	// INDETERMINATE | MU-02 (value_not_coercible). This is the case
	// checkMU02's own doc comment warns about: without the coercion
	// interception, a naive "arrived as a string -> PASS" reading would
	// report no precision lost about a value that was never read at all.
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU02(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU02_MU_V45(t *testing.T) {
	// MU-V45: money | null (neither string nor number, not coercible) |
	// INDETERMINATE | MU-02 (value_not_coercible).
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU02(t, in, verdict.OutcomeIndeterminate)
}
