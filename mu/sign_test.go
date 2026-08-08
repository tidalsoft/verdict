package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

func mustSign(t *testing.T, d field.MoneyDeclaration, s field.Sign) field.MoneyDeclaration {
	t.Helper()
	out, err := d.WithSign(s)
	if err != nil {
		t.Fatalf("WithSign(%v) unexpected error: %v", s, err)
	}
	return out
}

// mustConditionalSign builds a ConditionalSign keyed on "arguments.type",
// the only sibling field this file's tests ever condition on -- a
// parameter no call site varies is dead flexibility (golangci-lint's
// unparam agrees; see mustCurrencyField in scale_test.go for the same
// reasoning).
func mustConditionalSign(t *testing.T, whenValue string, s field.Sign) field.ConditionalSign {
	t.Helper()
	c, err := field.NewConditionalSign("arguments.type", whenValue, s)
	if err != nil {
		t.Fatalf("NewConditionalSign unexpected error: %v", err)
	}
	return c
}

func mustSignWhen(t *testing.T, d field.MoneyDeclaration, conds []field.ConditionalSign) field.MoneyDeclaration {
	t.Helper()
	out, err := d.WithSignWhen(conds)
	if err != nil {
		t.Fatalf("WithSignWhen unexpected error: %v", err)
	}
	return out
}

func TestCheckMU06_Vector_30(t *testing.T) {
	// Vector 30: sign_when refund → negative | type=refund, amount=500 | FAIL | MU-06
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	charge := mustConditionalSign(t, "charge", field.SignPositive)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund, charge})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.type": "refund"},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.CheckID() != "MU-06" {
		t.Errorf("CheckID() = %q, want MU-06", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU06_Vector_31(t *testing.T) {
	// Vector 31: no sign declaration | amount=-500 | INDETERMINATE | MU-06
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU06_SignWhen_ChargeMatches_Pass(t *testing.T) {
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	charge := mustConditionalSign(t, "charge", field.SignPositive)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund, charge})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.type": "charge"},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestCheckMU06_SignWhen_NoConditionMatches_Indeterminate(t *testing.T) {
	// SignWhen declared but the sibling value matches none of its
	// conditions -- INDETERMINATE, not a fallback to any unconditional
	// Sign (there isn't one declared here either).
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.type": "adjustment"},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU06_SignWhen_SiblingAbsent_Indeterminate(t *testing.T) {
	// The sibling field itself is absent from Vals entirely (comma-ok
	// miss), not merely holding a non-matching value.
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU06_SignWhen_TakesPrecedenceOverSign(t *testing.T) {
	// Both an unconditional Sign and SignWhen are declared. SignWhen is
	// authoritative once declared at all (see checkMU06's doc comment): a
	// matching condition wins even though it contradicts the unconditional
	// Sign.
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignPositive)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl = mustSignWhen(t, decl, []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]string{"arguments.type": "refund"},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	// Positive 500 violates the sign_when-resolved requirement (negative),
	// even though it would satisfy the unconditional Sign (positive).
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU06_UnconditionalSign_Positive(t *testing.T) {
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignPositive)
	cases := []struct {
		name  string
		value string
		want  verdict.Outcome
	}{
		{"positive passes", "500", verdict.OutcomePass},
		{"negative fails", "-500", verdict.OutcomeFail},
		{"zero passes (permitted under positive)", "0", verdict.OutcomePass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: mustRegistry(t, decl),
			}
			res, err := checkMU06(in)
			if err != nil {
				t.Fatalf("checkMU06 unexpected error: %v", err)
			}
			if res.Outcome() != tc.want {
				t.Errorf("Outcome() = %v, want %v", res.Outcome(), tc.want)
			}
		})
	}
}

func TestCheckMU06_UnconditionalSign_Negative(t *testing.T) {
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignNegative)
	cases := []struct {
		name  string
		value string
		want  verdict.Outcome
	}{
		{"negative passes", "-500", verdict.OutcomePass},
		{"positive fails", "500", verdict.OutcomeFail},
		{"zero passes (permitted under negative)", "0", verdict.OutcomePass},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, tc.value),
				Registry: mustRegistry(t, decl),
			}
			res, err := checkMU06(in)
			if err != nil {
				t.Fatalf("checkMU06 unexpected error: %v", err)
			}
			if res.Outcome() != tc.want {
				t.Errorf("Outcome() = %v, want %v", res.Outcome(), tc.want)
			}
		})
	}
}

func TestCheckMU06_UnconditionalSign_Any(t *testing.T) {
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignAny)
	for _, value := range []string{"500", "-500", "0"} {
		in := Input{
			Field:    "arguments.amount",
			Value:    mustParse(t, value),
			Registry: mustRegistry(t, decl),
		}
		res, err := checkMU06(in)
		if err != nil {
			t.Fatalf("checkMU06 unexpected error: %v", err)
		}
		if res.Outcome() != verdict.OutcomePass {
			t.Errorf("value %s: Outcome() = %v, want PASS", value, res.Outcome())
		}
	}
}

func TestCheckMU06_Nonzero_RejectsZero(t *testing.T) {
	decl, err := field.NewMoneyDeclaration().WithSign(field.SignPositive)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}
	decl = decl.WithNonzero()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0"),
		Registry: mustRegistry(t, decl),
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeFail {
		t.Errorf("Outcome() = %v, want FAIL", res.Outcome())
	}
}

func TestCheckMU06_NoDeclaration_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: field.Registry{},
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestCheckMU06_WrongKind_Indeterminate(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, field.NewDecimalDeclaration()),
	}
	res, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}
