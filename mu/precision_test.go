package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
)

func TestCheckMU02_Vector_09(t *testing.T) {
	// Vector 9: money, decimal string | "49.99" | PASS | MU-02
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "49.99"),
		Provenance: decimal.FromString,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-02" {
		t.Errorf("CheckID() = %q, want MU-02", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU02_Vector_10(t *testing.T) {
	// Vector 10: money, JSON number | 0.1 | FAIL | MU-02 (not exactly
	// representable in binary64)
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU02_Vector_11(t *testing.T) {
	// Vector 11: money, JSON number | 9007199254740993 | FAIL | MU-02
	// (> 2^53-1)
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "9007199254740993"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU02_JSONNumber_SafeIntegerBoundary_Pass(t *testing.T) {
	// 2^53-1 itself is exactly representable and does not exceed the safe
	// integer ceiling -- PASS, not FAIL, distinguishing this boundary from
	// vector 11's one-more-than-the-ceiling case.
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "9007199254740991"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
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
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
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
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
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
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU02_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0.1"),
		Registry: field.Registry{},
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU02_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:      "arguments.amount",
		Value:      mustParse(t, "0.1"),
		Provenance: decimal.FromJSONNumber,
		Registry:   mustRegistry(t, field.NewPercentageDeclaration()),
	}
	res, err := checkMU02(in)
	if err != nil {
		t.Fatalf("checkMU02 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
