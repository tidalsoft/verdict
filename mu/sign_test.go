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

// mustWhen builds a single-entry when-list on "arguments.type" -- the
// sibling every money-declaration sign_when test in this file conditions
// on, unless a test needs a multi-entry clause (built directly with
// field.NewWhenEntry instead).
func mustWhen(t *testing.T, whenValue string) []field.WhenEntry {
	t.Helper()
	entry, err := field.NewWhenEntry("arguments.type", field.NewStringValue(whenValue))
	if err != nil {
		t.Fatalf("NewWhenEntry unexpected error: %v", err)
	}
	return []field.WhenEntry{entry}
}

func mustConditionalSign(t *testing.T, whenValue string, s field.Sign) field.ConditionalSign {
	t.Helper()
	c, err := field.NewConditionalSign(mustWhen(t, whenValue), s)
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

// wantMU06 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU06: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-06's only severity), and Outcome.
func wantMU06(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU06 applicable = false, want true")
	}
	if res.CheckID() != "MU-06" {
		t.Errorf("CheckID() = %q, want MU-06", res.CheckID())
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

func TestCheckMU06_Vector_30(t *testing.T) {
	// Vector 30: sign_when refund → negative | type=refund, amount=500 | FAIL | MU-06
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	charge := mustConditionalSign(t, "charge", field.SignPositive)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund, charge})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "refund"}),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_31(t *testing.T) {
	// Vector 31: no sign declaration | amount=-500 | INDETERMINATE | MU-06
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, field.NewMoneyDeclaration()),
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_Vector_79(t *testing.T) {
	// Vector 79: sign: positive and sign_when refund → negative |
	// type=charge, amount=500 | PASS | MU-06 (unconditional sign governs)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "charge"}),
	}
	wantMU06(t, in, verdict.OutcomePass)
}

func TestCheckMU06_Vector_80(t *testing.T) {
	// Vector 80: sign: positive and sign_when refund → negative |
	// type=charge, amount=-500 | FAIL | MU-06 (unconditional sign governs)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "charge"}),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_81(t *testing.T) {
	// Vector 81: sign_when refund → negative, arguments.type absent |
	// amount=500 | INDETERMINATE | MU-06 (when path unresolved)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{},
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_Vector_82(t *testing.T) {
	// Vector 82: sign: positive, nonzero: true | 0 | FAIL | MU-06
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignPositive).WithNonzero()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0"),
		Registry: mustRegistry(t, decl),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_91(t *testing.T) {
	// Vector 91: sign: positive and sign_when { type: refund, category:
	// fees } → negative | type=charge, category absent, amount=-500 |
	// FAIL | MU-06 (one contradicted entry rules the clause out)
	typeEntry, err := field.NewWhenEntry("arguments.type", field.NewStringValue("refund"))
	if err != nil {
		t.Fatalf("NewWhenEntry unexpected error: %v", err)
	}
	categoryEntry, err := field.NewWhenEntry("arguments.category", field.NewStringValue("fees"))
	if err != nil {
		t.Fatalf("NewWhenEntry unexpected error: %v", err)
	}
	clause, err := field.NewConditionalSign([]field.WhenEntry{typeEntry, categoryEntry}, field.SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign unexpected error: %v", err)
	}
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{clause})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, decl),
		// arguments.category deliberately absent: this entry doesn't
		// resolve, but arguments.type=charge contradicts the clause's
		// stated "refund" outright, and that contradiction rules the
		// clause out regardless.
		Vals: stringVals(map[string]string{"arguments.type": "charge"}),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_92(t *testing.T) {
	// Vector 92: sign: positive and sign_when refund → negative |
	// type=null, amount=-500 | FAIL | MU-06 (an explicit null resolves)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.type": field.NewNullValue()},
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_104(t *testing.T) {
	// Vector 104: sign_when refund → negative, no unconditional sign |
	// type=charge, amount=500 | INDETERMINATE | MU-06 (no clause matches,
	// no fallback)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "charge"}),
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_Vector_113(t *testing.T) {
	// Vector 113: sign: positive and sign_when refund → negative,
	// arguments.type absent | amount=500 | INDETERMINATE | MU-06 (rule 2
	// precedes the unconditional sign fallback)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{},
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_Vector_116(t *testing.T) {
	// Vector 116: sign: any | -500 | PASS | MU-06 (any establishes a sign
	// nothing violates)
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignAny)
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, decl),
	}
	wantMU06(t, in, verdict.OutcomePass)
}

func TestCheckMU06_Vector_117(t *testing.T) {
	// Vector 117: sign: any, nonzero: true | 0 | FAIL | MU-06 (any with
	// nonzero still rejects zero)
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignAny).WithNonzero()
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "0"),
		Registry: mustRegistry(t, decl),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_Vector_118(t *testing.T) {
	// Vector 118: sign: positive and sign_when { arguments.metadata:
	// "premium" } → negative | arguments.metadata: {tier: "gold"},
	// amount=500 | INDETERMINATE | MU-06 (when path resolves to a JSON
	// object, not a comparable shape)
	entry, err := field.NewWhenEntry("arguments.metadata", field.NewStringValue("premium"))
	if err != nil {
		t.Fatalf("NewWhenEntry unexpected error: %v", err)
	}
	clause, err := field.NewConditionalSign([]field.WhenEntry{entry}, field.SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign unexpected error: %v", err)
	}
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{clause})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.metadata": field.NewNonComparableValue()},
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_Vector_122(t *testing.T) {
	// Vector 122: sign: positive and sign_when { arguments.type: ["refund"]
	// } → negative | type="refund", amount=500 | INDETERMINATE | MU-06
	// (stated value is a JSON array, not a comparable shape)
	entry, err := field.NewWhenEntry("arguments.type", field.NewNonComparableValue())
	if err != nil {
		t.Fatalf("NewWhenEntry unexpected error: %v", err)
	}
	clause, err := field.NewConditionalSign([]field.WhenEntry{entry}, field.SignNegative)
	if err != nil {
		t.Fatalf("NewConditionalSign unexpected error: %v", err)
	}
	decl := mustSignWhen(t, mustSign(t, field.NewMoneyDeclaration(), field.SignPositive), []field.ConditionalSign{clause})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     map[string]field.Value{"arguments.type": field.NewStringValue("refund")},
	}
	wantMU06(t, in, verdict.OutcomeIndeterminate)
}

func TestCheckMU06_SignWhen_ChargeMatches_Pass(t *testing.T) {
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	charge := mustConditionalSign(t, "charge", field.SignPositive)
	decl := mustSignWhen(t, field.NewMoneyDeclaration(), []field.ConditionalSign{refund, charge})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "charge"}),
	}
	wantMU06(t, in, verdict.OutcomePass)
}

func TestCheckMU06_SignWhen_MatchingClauseGovernsOverFallback(t *testing.T) {
	// Both an unconditional Sign and a genuinely matching SignWhen clause
	// are present. Rule 1 (a matching clause governs) has priority over
	// rule 3 (the unconditional fallback), even though the fallback would
	// have said something different.
	decl := mustSign(t, field.NewMoneyDeclaration(), field.SignPositive)
	refund := mustConditionalSign(t, "refund", field.SignNegative)
	decl = mustSignWhen(t, decl, []field.ConditionalSign{refund})
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "500"),
		Registry: mustRegistry(t, decl),
		Vals:     stringVals(map[string]string{"arguments.type": "refund"}),
	}
	res, applicable, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU06 applicable = false, want true")
	}
	// Positive 500 violates the matching clause's sign (negative), even
	// though it would satisfy the unconditional Sign (positive).
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
			res, applicable, err := checkMU06(in)
			if err != nil {
				t.Fatalf("checkMU06 unexpected error: %v", err)
			}
			if !applicable {
				t.Fatal("checkMU06 applicable = false, want true")
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
			res, applicable, err := checkMU06(in)
			if err != nil {
				t.Fatalf("checkMU06 unexpected error: %v", err)
			}
			if !applicable {
				t.Fatal("checkMU06 applicable = false, want true")
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
		res, applicable, err := checkMU06(in)
		if err != nil {
			t.Fatalf("checkMU06 unexpected error: %v", err)
		}
		if !applicable {
			t.Fatal("checkMU06 applicable = false, want true")
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
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: field.Registry{},
	}
	_, applicable, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU06 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU06_WrongKind_NotApplicable(t *testing.T) {
	// percentage carries no Sign/SignWhen/Nonzero at all, so it does not
	// satisfy signDeclaration -- unlike decimal, which now does (SPEC-MU
	// §2.5.1's trigger matrix applies MU-06 to both money and decimal).
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, field.NewPercentageDeclaration()),
	}
	_, applicable, err := checkMU06(in)
	if err != nil {
		t.Fatalf("checkMU06 unexpected error: %v", err)
	}
	if applicable {
		t.Error("checkMU06 applicable = true, want false (wrong kind)")
	}
}

// TestCheckMU06_DecimalKind_Applies confirms MU-06 now evaluates a
// `kind: decimal` field exactly as it does money, per SPEC-MU §2.5.1's
// trigger matrix.
func TestCheckMU06_DecimalKind_Applies(t *testing.T) {
	decl, err := field.NewDecimalDeclaration().WithSign(field.SignPositive)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}
	in := Input{
		Field:    "arguments.amount",
		Value:    mustParse(t, "-500"),
		Registry: mustRegistry(t, decl),
	}
	wantMU06(t, in, verdict.OutcomeFail)
}

func TestCheckMU06_ValueNotCoercible_Indeterminate(t *testing.T) {
	// SPEC-MU §2.6.3: MU-06 is value-dependent, so a value the coercion
	// gate could not read is INDETERMINATE without ever consulting
	// signViolated -- Value must not be read once ValueCoercionFailed is
	// true (see Input's own doc comment).
	decl, err := field.NewMoneyDeclaration().WithSign(field.SignPositive)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}
	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
	}
	res, applicable, gotErr := checkMU06(in)
	if gotErr != nil {
		t.Fatalf("checkMU06 unexpected error: %v", gotErr)
	}
	if !applicable {
		t.Fatal("checkMU06 applicable = false, want true")
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
}

func TestClauseState_EmptyEntries_Matches(t *testing.T) {
	// A clause with an empty when list matches every request -- SPEC-MU
	// §3: "a clause with an empty when map matches every request, which is
	// a pointless way to write an unconditional sign." field.
	// NewConditionalSign itself refuses to construct one (see
	// money_test.go), so this drives clauseState directly with a nil/empty
	// entries slice to exercise the branch clauseState alone can still
	// reach.
	if got := clauseState(nil, map[string]field.Value{}); got != clauseMatches {
		t.Errorf("clauseState(nil, {}) = %v, want clauseMatches", got)
	}
}
