package mu

import (
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// wantMU08 asserts every field SPEC-MU §8.3 constrains for a conformance
// vector against checkMU08: CheckID, Class (ClassD), Severity
// (SeverityBlock -- MU-08's only severity), and Outcome.
func wantMU08(t *testing.T, in Input, want verdict.Outcome) {
	t.Helper()
	res, applicable, err := checkMU08(in)
	if err != nil {
		t.Fatalf("checkMU08 unexpected error: %v", err)
	}
	if !applicable {
		t.Fatal("checkMU08 applicable = false, want true")
	}
	if res.CheckID() != "MU-08" {
		t.Errorf("CheckID() = %q, want MU-08", res.CheckID())
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

func mustNullSemanticsDistinct(t *testing.T, d field.MoneyDeclaration) field.MoneyDeclaration {
	t.Helper()
	out, err := d.WithNullSemantics(field.NullSemanticsDistinct)
	if err != nil {
		t.Fatalf("WithNullSemantics unexpected error: %v", err)
	}
	return out
}

func TestCheckMU08_MU_V69(t *testing.T) {
	// MU-V69: money, null_semantics: distinct | null | FAIL | MU-08
	decl := mustNullSemanticsDistinct(t, field.NewMoneyDeclaration())
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistry(t, decl),
		HasRawValue: true,
		RawValue:    field.NewNullValue(),
	}
	wantMU08(t, in, verdict.OutcomeFail)
}

func TestCheckMU08_MU_V70(t *testing.T) {
	// MU-V70: money, null_semantics: distinct | path absent | PASS | MU-08
	decl := mustNullSemanticsDistinct(t, field.NewMoneyDeclaration())
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, decl),
		// HasRawValue left false: the path is absent from the arguments.
	}
	wantMU08(t, in, verdict.OutcomePass)
}

func TestCheckMU08_PresentNonNull_Pass(t *testing.T) {
	// A field present with any non-null value is exactly the case
	// null_semantics: distinct does not object to -- neither of the two
	// meanings it distinguishes.
	decl := mustNullSemanticsDistinct(t, field.NewMoneyDeclaration())
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistry(t, decl),
		HasRawValue: true,
		RawValue:    field.NewNumberValue(mustParse(t, "49.99")),
	}
	wantMU08(t, in, verdict.OutcomePass)
}

func TestCheckMU08_NoDeclaration_NotApplicable(t *testing.T) {
	in := Input{Field: "arguments.amount"}
	_, applicable, err := checkMU08(in)
	if err != nil {
		t.Fatalf("checkMU08 unexpected error: %v", err)
	}
	if applicable {
		t.Fatal("checkMU08 applicable = true, want false (no declaration)")
	}
}

func TestCheckMU08_NullSemanticsNotDeclared_NotApplicable(t *testing.T) {
	// §2.5.2: null_semantics absent is a gate, not a gap -- a coherent,
	// complete declaration that null and omission mean the same thing.
	decl := field.NewMoneyDeclaration()
	in := Input{
		Field:       "arguments.amount",
		Registry:    mustRegistry(t, decl),
		HasRawValue: true,
		RawValue:    field.NewNullValue(),
	}
	_, applicable, err := checkMU08(in)
	if err != nil {
		t.Fatalf("checkMU08 unexpected error: %v", err)
	}
	if applicable {
		t.Fatal("checkMU08 applicable = true, want false (null_semantics not declared)")
	}
}

func TestCheckMU08_EveryKind(t *testing.T) {
	// MU-08 applies to every kind (SPEC-MU §2.5.1) -- exercised here
	// against quantity, timestamp, and identifier declarations, none of
	// which this package's other checks that read NullSemantics touch.
	cases := []struct {
		name string
		decl field.Declaration
	}{
		{"quantity", func() field.Declaration {
			d, err := field.NewQuantityDeclaration().WithNullSemantics(field.NullSemanticsDistinct)
			if err != nil {
				t.Fatalf("WithNullSemantics unexpected error: %v", err)
			}
			return d
		}()},
		{"timestamp", func() field.Declaration {
			d, err := field.NewTimestampDeclaration().WithNullSemantics(field.NullSemanticsDistinct)
			if err != nil {
				t.Fatalf("WithNullSemantics unexpected error: %v", err)
			}
			return d
		}()},
		{"identifier", func() field.Declaration {
			d, err := field.NewIdentifierDeclaration().WithNullSemantics(field.NullSemanticsDistinct)
			if err != nil {
				t.Fatalf("WithNullSemantics unexpected error: %v", err)
			}
			return d
		}()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:       "arguments.amount",
				Registry:    mustRegistry(t, tc.decl),
				HasRawValue: true,
				RawValue:    field.NewNullValue(),
			}
			wantMU08(t, in, verdict.OutcomeFail)
		})
	}
}
