package mu

import (
	"embed"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

//go:embed *.go
var muSource embed.FS

func mustParse(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.Parse(s)
	if err != nil {
		t.Fatalf("decimal.Parse(%q) unexpected error: %v", s, err)
	}
	return d
}

func mustRegistry(t *testing.T, decl field.Declaration) field.Registry {
	t.Helper()
	r, err := field.NewRegistry(map[string]field.Declaration{"arguments.amount": decl})
	if err != nil {
		t.Fatalf("field.NewRegistry unexpected error: %v", err)
	}
	return r
}

// stringVals converts a plain path -> string map into Input.Vals' real
// type, map[string]field.Value, wrapping every entry as a
// field.NewStringValue. Most of this package's tests only ever need a
// sibling to resolve to a string (a currency code, a unit, a sign_when
// comparand), so this is the common case's shorthand; tests exercising
// MU-06's number/bool/null/non-comparable comparable-shape rules build
// their Vals map directly instead.
func stringVals(m map[string]string) map[string]field.Value {
	out := make(map[string]field.Value, len(m))
	for k, v := range m {
		out[k] = field.NewStringValue(v)
	}
	return out
}

// fixedResult builds an OnFunc that always returns the given outcome at the
// given checkID, applicable == true, for exercising evaluateChecks'
// aggregation with injected outcomes.
func fixedResult(checkID string, outcome verdict.Outcome) OnFunc {
	return func(_ Input) (verdict.Result, bool, error) {
		res, err := newResult(checkID, outcome)
		return res, true, err
	}
}

// funcPointer returns f's entry address, for comparing two OnFunc values by
// identity (Go func values are not otherwise comparable). Used only by
// TestChecksFor_DispatchTable below.
func funcPointer(f OnFunc) uintptr {
	return reflect.ValueOf(f).Pointer()
}

func TestChecksFor_DispatchTable(t *testing.T) {
	cases := []struct {
		name string
		kind field.Kind
		want []OnFunc
		ok   bool
	}{
		{"money", field.KindMoney, []OnFunc{checkMU01, checkMU02, checkMU03, checkMU06, checkMU07, checkMU14}, true},
		{"decimal", field.KindDecimal, []OnFunc{checkMU02, checkMU06, checkMU07}, true},
		{"percentage", field.KindPercentage, []OnFunc{checkMU07, checkMU13}, true},
		{"quantity", field.KindQuantity, []OnFunc{checkMU04, checkMU05, checkMU07, checkMU15}, true},
		{"unspecified", field.KindUnspecified, nil, false},
		{"timestamp", field.KindTimestamp, nil, false},
		{"identifier", field.KindIdentifier, nil, false},
	}
	// This table already runs every field.Kind this package defines, so
	// it doubles as the regression guard for checksFor's ok == len(checks)
	// > 0 invariant: ok is derived structurally in checksFor now (see its
	// doc comment), but a future kind added here with the wrong want list
	// would only be caught if every case actually checks ok against
	// len(checks), not just against a hand-written expectation -- so this
	// loop asserts the derivation itself, not just the table's own values.
	//
	// This test compares checksFor's returned []OnFunc against the exact
	// function values expected, by entry-point pointer (funcPointer),
	// rather than invoking each with a zero Input and reading its result:
	// invoking is no longer a safe way to identify a check now that
	// "not applicable" (a zero, unread Result) is a real outcome a check
	// can produce for a zero Input -- see OnFunc's own doc comment.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checks, ok := checksFor(tc.kind)
			if ok != tc.ok {
				t.Fatalf("checksFor(%v) ok = %v, want %v", tc.kind, ok, tc.ok)
			}
			if ok != (len(checks) > 0) {
				t.Fatalf("checksFor(%v): ok = %v but len(checks) = %d -- ok must equal len(checks) > 0", tc.kind, ok, len(checks))
			}
			if len(checks) != len(tc.want) {
				t.Fatalf("checksFor(%v) = %d checks, want %d", tc.kind, len(checks), len(tc.want))
			}
			for i, check := range checks {
				if funcPointer(check) != funcPointer(tc.want[i]) {
					t.Errorf("checksFor(%v)[%d] is not the expected check function", tc.kind, i)
				}
			}
		})
	}
}

func TestEvaluate_DispatchPerKind(t *testing.T) {
	// Every case below declares its kind but nothing else. Every check
	// gated on a declared attribute (SPEC-MU §2.5.1/§2.5.2: MU-03's
	// target_currency_field, MU-05's unit_required, MU-07's min/max, MU-15's
	// canonical_unit) is not applicable to a bare declaration and so
	// contributes no entry at all. Every check with no gate, or whose gate
	// is itself undecidable without a declared attribute (MU-14's
	// scale: major_units), is applicable and returns INDETERMINATE for
	// want of a required input -- except MU-02 (precision.go), whose only
	// requirement is the field's kind (money or decimal) being one it
	// applies to at all: with Value's zero value (exact 0) and Provenance's
	// zero value (FromString), MU-02 evaluates for real and PASSes, since
	// decimal.PrecisionLoss never fails a FromString value. This is also
	// the regression case the applicability fix exists for: an ordinary,
	// minimally-declared money field produces no MU-03 or MU-07 entry at
	// all, rather than a spurious INDETERMINATE that ModeStrict would deny
	// on (see TestOrdinaryMoneyRuleset_NotDenyUnderStrict below). Evaluate
	// (§2.6: no short-circuiting) must return every applicable check's
	// result, in ascending check-ID order, not just the first.
	cases := []struct {
		name         string
		decl         field.Declaration
		wantIDs      []string
		wantOutcomes []verdict.Outcome
	}{
		{
			"money",
			field.NewMoneyDeclaration(),
			[]string{"MU-01", "MU-02", "MU-06", "MU-14"},
			[]verdict.Outcome{
				verdict.OutcomeIndeterminate, verdict.OutcomePass,
				verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate,
			},
		},
		{
			"decimal",
			field.NewDecimalDeclaration(),
			[]string{"MU-02", "MU-06"},
			[]verdict.Outcome{verdict.OutcomePass, verdict.OutcomeIndeterminate},
		},
		{
			"percentage",
			field.NewPercentageDeclaration(),
			[]string{"MU-13"},
			[]verdict.Outcome{verdict.OutcomeIndeterminate},
		},
		{
			"quantity",
			field.NewQuantityDeclaration(),
			[]string{"MU-04"},
			[]verdict.Outcome{verdict.OutcomeIndeterminate},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:    "arguments.amount",
				Registry: mustRegistry(t, tc.decl),
			}
			results, err := Evaluate(in)
			if err != nil {
				t.Fatalf("Evaluate unexpected error: %v", err)
			}
			if len(results) != len(tc.wantIDs) {
				t.Fatalf("Evaluate returned %d results, want %d", len(results), len(tc.wantIDs))
			}
			for i, res := range results {
				if res.CheckID() != tc.wantIDs[i] {
					t.Errorf("result %d CheckID() = %q, want %q", i, res.CheckID(), tc.wantIDs[i])
				}
				if res.Outcome() != tc.wantOutcomes[i] {
					t.Errorf("result %d Outcome() = %v, want %v", i, res.Outcome(), tc.wantOutcomes[i])
				}
			}
		})
	}
}

// TestOrdinaryRuleset_NotDenyUnderStrict is the regression this task exists
// for, covering all four kinds this package carries gated checks for. Each
// case declares a field with everything a real check actually requires --
// scale/currency/sign for money, sign for decimal, domain for percentage,
// dimension and a resolvable unit for quantity -- but deliberately nothing
// beyond that: no target_currency_field or bounds on money, no bounds on
// decimal or percentage, no unit_required or canonical_unit or bounds on
// quantity. None of that must produce a spurious block-severity
// INDETERMINATE that ModeStrict then denies on. Before the applicability
// fix, every one of the listed absentIDs reported INDETERMINATE for a gate
// the ruleset never asked to have evaluated at all, and
// verdict.ComputeAggregate turns any block-severity INDETERMINATE into
// VerdictDeny under ModeStrict -- invariant 1 (INDETERMINATE never
// collapses to PASS) must not be read backwards into "manufacture a DENY
// out of a check that was never applicable". Money and decimal declare
// sign: any and percentage declares a domain so that MU-06/MU-13 -- genuine
// required-input checks, not gates -- resolve to a real PASS rather than
// their own honest INDETERMINATE, which would deny under ModeStrict for a
// completely different, correct reason these cases are not about.
func TestOrdinaryRuleset_NotDenyUnderStrict(t *testing.T) {
	moneyDecl := field.NewMoneyDeclaration()
	moneyDecl, err := moneyDecl.WithScale(field.ScaleMajorUnits)
	if err != nil {
		t.Fatalf("WithScale unexpected error: %v", err)
	}
	moneyDecl, err = moneyDecl.WithCurrencyField("arguments.currency")
	if err != nil {
		t.Fatalf("WithCurrencyField unexpected error: %v", err)
	}
	moneyDecl, err = moneyDecl.WithSign(field.SignAny)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}

	decimalDecl, err := field.NewDecimalDeclaration().WithSign(field.SignAny)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}

	percentageDecl, err := field.NewPercentageDeclaration().WithDomain(field.DomainUnitInterval)
	if err != nil {
		t.Fatalf("WithDomain unexpected error: %v", err)
	}

	quantityDecl := mustDimension(t, field.NewQuantityDeclaration(), "mass")

	cases := []struct {
		name      string
		in        Input
		absentIDs []string // gated checks that must not appear at all
	}{
		{
			name: "money",
			in: Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, "49.99"),
				Registry: mustRegistry(t, moneyDecl),
				Vals:     stringVals(map[string]string{"arguments.currency": "USD"}),
				Tables:   Tables{ISO4217: tables.NewISO4217Table()},
			},
			// MU-03: no target_currency_field. MU-07: no min/max.
			absentIDs: []string{"MU-03", "MU-07"},
		},
		{
			name: "decimal",
			in: Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, "5"),
				Registry: mustRegistry(t, decimalDecl),
			},
			// MU-07: no min/max.
			absentIDs: []string{"MU-07"},
		},
		{
			name: "percentage",
			in: Input{
				Field:    "arguments.amount",
				Value:    mustParse(t, "0.5"),
				Registry: mustRegistry(t, percentageDecl),
			},
			// MU-07: no min/max.
			absentIDs: []string{"MU-07"},
		},
		{
			name: "quantity",
			in: Input{
				Field:           "arguments.amount",
				Value:           mustParse(t, "12"),
				EmbeddedUnit:    "kg",
				HasEmbeddedUnit: true,
				Registry:        mustRegistry(t, quantityDecl),
				Tables:          unitTables(),
			},
			// MU-05: unit_required not declared. MU-07: no min/max.
			// MU-15: no canonical_unit.
			absentIDs: []string{"MU-05", "MU-07", "MU-15"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := Evaluate(tc.in)
			if err != nil {
				t.Fatalf("Evaluate unexpected error: %v", err)
			}
			absent := make(map[string]bool, len(tc.absentIDs))
			for _, id := range tc.absentIDs {
				absent[id] = true
			}
			for _, res := range results {
				if absent[res.CheckID()] {
					t.Errorf("Evaluate returned a %s result for a field that never declared its gate: %v", res.CheckID(), res.Outcome())
				}
			}
			agg, err := verdict.ComputeAggregate(results, verdict.ModeStrict)
			if err != nil {
				t.Fatalf("ComputeAggregate unexpected error: %v", err)
			}
			if agg.Verdict == verdict.VerdictDeny {
				t.Errorf("ComputeAggregate under ModeStrict = VerdictDeny (reason %v), want anything else, for results %v", agg.Reason, results)
			}
		})
	}
}

// TestEvaluate_Vector_16 wires SPEC-MU §8.3 vector 16: a money field whose
// value did not coerce ("1,234", a string resolution refuses) must report
// INDETERMINATE from every applicable value-dependent check (§2.6.3) --
// MU-01, MU-02, MU-06, MU-07, MU-14 for this declaration -- while MU-03,
// which is not value-dependent (§2.6.3's table) and never reads the
// field's own value, evaluates normally and is unaffected by the coercion
// failure. That contrast is the point of the vector: the coercion gate is
// scoped to value-dependent checks, not to the field as a whole.
func TestEvaluate_Vector_16(t *testing.T) {
	decl := field.NewMoneyDeclaration()
	decl, err := decl.WithScale(field.ScaleMajorUnits)
	if err != nil {
		t.Fatalf("WithScale unexpected error: %v", err)
	}
	decl, err = decl.WithCurrencyField("arguments.currency")
	if err != nil {
		t.Fatalf("WithCurrencyField unexpected error: %v", err)
	}
	decl, err = decl.WithTargetCurrencyField("arguments.target_currency")
	if err != nil {
		t.Fatalf("WithTargetCurrencyField unexpected error: %v", err)
	}
	decl, err = decl.WithSign(field.SignPositive)
	if err != nil {
		t.Fatalf("WithSign unexpected error: %v", err)
	}
	decl = decl.WithMin(mustParse(t, "0"))

	in := Input{
		Field:               "arguments.amount",
		ValueCoercionFailed: true,
		Registry:            mustRegistry(t, decl),
		Vals: stringVals(map[string]string{
			"arguments.currency":        "USD",
			"arguments.target_currency": "USD",
		}),
		Tables: Tables{ISO4217: tables.NewISO4217Table()},
	}
	results, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate unexpected error: %v", err)
	}

	wantOutcome := map[string]verdict.Outcome{
		"MU-01": verdict.OutcomeIndeterminate,
		"MU-02": verdict.OutcomeIndeterminate,
		"MU-03": verdict.OutcomePass, // not value-dependent: unaffected
		"MU-06": verdict.OutcomeIndeterminate,
		"MU-07": verdict.OutcomeIndeterminate,
		"MU-14": verdict.OutcomeIndeterminate,
	}
	if len(results) != len(wantOutcome) {
		t.Fatalf("Evaluate returned %d results, want %d (%v)", len(results), len(wantOutcome), results)
	}
	seen := make(map[string]bool, len(results))
	for _, res := range results {
		seen[res.CheckID()] = true
		want, ok := wantOutcome[res.CheckID()]
		if !ok {
			t.Errorf("Evaluate returned unexpected check %s", res.CheckID())
			continue
		}
		if res.Outcome() != want {
			t.Errorf("%s Outcome() = %v, want %v", res.CheckID(), res.Outcome(), want)
		}
	}
	for id := range wantOutcome {
		if !seen[id] {
			t.Errorf("Evaluate missing expected check %s", id)
		}
	}
}

func TestEvaluate_UnknownKind(t *testing.T) {
	in := Input{
		Field:    "arguments.amount",
		Registry: mustRegistry(t, field.NewTimestampDeclaration()),
	}
	_, err := Evaluate(in)
	if err == nil {
		t.Fatal("Evaluate with timestamp kind succeeded, want error")
	}
	if !strings.Contains(err.Error(), "no checks") {
		t.Errorf("error %q missing context", err.Error())
	}
}

func TestEvaluate_ZeroValueRegistry(t *testing.T) {
	in := Input{Field: "arguments.amount"}
	results, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate with zero-value Registry unexpected error: %v", err)
	}
	// "No declaration" is itself the one thing worth reporting when no
	// check ran at all -- a single-element slice, not an empty one.
	if len(results) != 1 {
		t.Fatalf("Evaluate with zero-value Registry returned %d results, want 1", len(results))
	}
	if results[0].Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Evaluate Outcome() = %v, want INDETERMINATE", results[0].Outcome())
	}
	if results[0].CheckID() != checkIDNoDeclaration {
		t.Errorf("Evaluate CheckID() = %q, want %q", results[0].CheckID(), checkIDNoDeclaration)
	}
}

func TestEvaluateChecks_ReturnsEveryResultInOrder(t *testing.T) {
	// SPEC-MU §2.4: "Evaluation does not short-circuit: all applicable
	// checks run and all failures are reported." evaluateChecks must
	// return every check's result, in the order the checks were given --
	// never collapse to "the first FAIL" or "the first non-PASS".
	cases := []struct {
		name         string
		checks       []OnFunc
		wantIDs      []string
		wantOutcomes []verdict.Outcome
	}{
		{
			name:         "FAIL does not suppress a later result",
			checks:       []OnFunc{fixedResult("MU-01", verdict.OutcomeIndeterminate), fixedResult("MU-14", verdict.OutcomeFail)},
			wantIDs:      []string{"MU-01", "MU-14"},
			wantOutcomes: []verdict.Outcome{verdict.OutcomeIndeterminate, verdict.OutcomeFail},
		},
		{
			name:         "two independent FAILs are both reported",
			checks:       []OnFunc{fixedResult("MU-01", verdict.OutcomeFail), fixedResult("MU-02", verdict.OutcomeFail)},
			wantIDs:      []string{"MU-01", "MU-02"},
			wantOutcomes: []verdict.Outcome{verdict.OutcomeFail, verdict.OutcomeFail},
		},
		{
			name:         "PASS, FAIL, INDETERMINATE all preserved in order",
			checks:       []OnFunc{fixedResult("MU-01", verdict.OutcomePass), fixedResult("MU-14", verdict.OutcomeFail), fixedResult("MU-02", verdict.OutcomeIndeterminate)},
			wantIDs:      []string{"MU-01", "MU-14", "MU-02"},
			wantOutcomes: []verdict.Outcome{verdict.OutcomePass, verdict.OutcomeFail, verdict.OutcomeIndeterminate},
		},
		{
			name:         "all PASS returns every PASS",
			checks:       []OnFunc{fixedResult("MU-01", verdict.OutcomePass), fixedResult("MU-14", verdict.OutcomePass)},
			wantIDs:      []string{"MU-01", "MU-14"},
			wantOutcomes: []verdict.Outcome{verdict.OutcomePass, verdict.OutcomePass},
		},
		{
			// SPEC-MU §2.5: a not-applicable check contributes no entry at
			// all -- neither an omitted-but-counted slot nor a zero Result
			// -- so evaluateChecks must skip it entirely rather than
			// reporting it as some fourth outcome.
			name: "not applicable check contributes no entry",
			checks: []OnFunc{
				fixedResult("MU-01", verdict.OutcomePass),
				func(Input) (verdict.Result, bool, error) { return notApplicable() },
				fixedResult("MU-14", verdict.OutcomeFail),
			},
			wantIDs:      []string{"MU-01", "MU-14"},
			wantOutcomes: []verdict.Outcome{verdict.OutcomePass, verdict.OutcomeFail},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := evaluateChecks(Input{}, tc.checks)
			if err != nil {
				t.Fatalf("evaluateChecks unexpected error: %v", err)
			}
			if len(results) != len(tc.wantIDs) {
				t.Fatalf("evaluateChecks returned %d results, want %d", len(results), len(tc.wantIDs))
			}
			for i, res := range results {
				if res.CheckID() != tc.wantIDs[i] {
					t.Errorf("result %d CheckID() = %q, want %q", i, res.CheckID(), tc.wantIDs[i])
				}
				if res.Outcome() != tc.wantOutcomes[i] {
					t.Errorf("result %d Outcome() = %v, want %v", i, res.Outcome(), tc.wantOutcomes[i])
				}
			}
		})
	}
}

func TestEvaluateChecks_ErrorPropagation(t *testing.T) {
	boom := func(_ Input) (verdict.Result, bool, error) {
		return verdict.Result{}, true, errors.New("boom")
	}
	_, err := evaluateChecks(Input{}, []OnFunc{boom})
	if err == nil {
		t.Fatal("evaluateChecks with failing check succeeded, want error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q missing cause", err.Error())
	}
}

func TestEvaluateChecks_Empty(t *testing.T) {
	// Unlike the single-Result design this replaced -- which had no
	// honest empty value and so treated an empty check list as an error
	// -- a []verdict.Result has one: "no checks ran" is simply the empty
	// slice, not an error condition.
	results, err := evaluateChecks(Input{}, nil)
	if err != nil {
		t.Fatalf("evaluateChecks with no checks unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("evaluateChecks with no checks returned %d results, want 0", len(results))
	}
}

func TestResolveCurrency(t *testing.T) {
	tbl := tables.NewISO4217Table()
	cases := []struct {
		name     string
		code     string
		wantCode string
		ok       bool
	}{
		{"upper", "USD", "USD", true},
		{"lower", "usd", "USD", true},
		{"mixed", "Usd", "USD", true},
		{"lower other", "jpy", "JPY", true},
		{"not a code", "ZZZ", "", false},
		{"empty", "", "", false},
		// Regression: Turkish dotless ı (U+0131) upper-cases to plain
		// ASCII "I" under Unicode's full case-mapping tables (what
		// strings.ToUpper would do), so "ıqd" would resolve to IQD -- a
		// real currency -- even though "ıqd" is not ISO 4217 text at
		// all (ı is not a letter ISO 4217 codes use). resolveCurrency's
		// ASCII-only fold must leave ı untouched, so this must miss.
		{"dotless i manufactures IQD", "ıqd", "", false},
		{"leading space", " USD", "", false},
		{"trailing space", "USD ", "", false},
		// Kelvin sign (U+212A) is canonically equivalent to 'K' under
		// Unicode normalization/case-folding, but must not be treated
		// as ASCII 'K' by a fold that only touches 'a'-'z'.
		{"kelvin sign prefix", "KWD", "", false},
		{"full-width letters", "ＵＳＤ", "", false}, // "ＵＳＤ"
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := (Tables{ISO4217: tbl}).resolveCurrency(tc.code)
			if ok != tc.ok {
				t.Fatalf("resolveCurrency(%q) ok = %v, want %v", tc.code, ok, tc.ok)
			}
			if ok && got.Code() != tc.wantCode {
				t.Errorf("resolveCurrency(%q).Code() = %q, want %q", tc.code, got.Code(), tc.wantCode)
			}
		})
	}
}

func TestAsciiUpper(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"already upper", "USD", "USD"},
		{"lower", "usd", "USD"},
		{"mixed", "UsD", "USD"},
		{"empty", "", ""},
		{"digits and punctuation untouched", "u1-d", "U1-D"},
		// The whole point: bytes outside 'a'-'z' -- including every byte
		// of a multi-byte UTF-8 rune -- pass through unchanged, so this
		// can never manufacture an ASCII letter that wasn't already one.
		{"dotless i untouched", "ıqd", "ıQD"},
		{"kelvin sign untouched", "Kwd", "KWD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := asciiUpper(tc.in); got != tc.want {
				t.Errorf("asciiUpper(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestBounded exercises bounded's own contract directly: value, min, and
// max are already in the same units by the time bounded sees them (that
// normalization is checkMU07's job -- see range_test.go for the
// unit-conversion cases), so every case here passes min/max already
// expressed in whatever unit "value" is in, exactly as checkMU07 would
// hand them over post-normalization.
func TestBounded(t *testing.T) {
	cases := []struct {
		name   string
		decl   field.MoneyDeclaration
		value  string
		min    string
		hasMin bool
		max    string
		hasMax bool
		want   verdict.Outcome
	}{
		{
			name:  "below min -> FAIL",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			min:   "50", hasMin: true,
			want: verdict.OutcomeFail,
		},
		{
			name:  "above max -> FAIL",
			decl:  field.NewMoneyDeclaration(),
			value: "50.01",
			max:   "50", hasMax: true,
			want: verdict.OutcomeFail,
		},
		{
			name:  "within bounds -> PASS",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			min:   "0", hasMin: true,
			max: "100", hasMax: true,
			want: verdict.OutcomePass,
		},
		{
			name:  "inclusive min equality -> PASS",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			min:   "49.99", hasMin: true,
			want: verdict.OutcomePass,
		},
		{
			name:  "inclusive max equality -> PASS",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			max:   "49.99", hasMax: true,
			want: verdict.OutcomePass,
		},
		{
			name:  "exclusive min equality -> FAIL",
			decl:  field.NewMoneyDeclaration().WithExclusiveMin(),
			value: "49.99",
			min:   "49.99", hasMin: true,
			want: verdict.OutcomeFail,
		},
		{
			name:  "exclusive max equality -> FAIL",
			decl:  field.NewMoneyDeclaration().WithExclusiveMax(),
			value: "49.99",
			max:   "49.99", hasMax: true,
			want: verdict.OutcomeFail,
		},
		{
			name:  "only min bound, value above -> PASS",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			min:   "0", hasMin: true,
			want: verdict.OutcomePass,
		},
		{
			name:  "only max bound, value below -> PASS",
			decl:  field.NewMoneyDeclaration(),
			value: "49.99",
			max:   "100", hasMax: true,
			want: verdict.OutcomePass,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, applicable, err := bounded(tc.decl, mustParse(t, tc.value), mustParse(t, orZero(tc.min)), tc.hasMin, mustParse(t, orZero(tc.max)), tc.hasMax)
			if err != nil {
				t.Fatalf("bounded unexpected error: %v", err)
			}
			if !applicable {
				t.Fatal("bounded applicable = false, want true")
			}
			if res.CheckID() != "MU-07" {
				t.Errorf("bounded CheckID() = %q, want MU-07", res.CheckID())
			}
			if res.Outcome() != tc.want {
				t.Errorf("bounded Outcome() = %v, want %v", res.Outcome(), tc.want)
			}
		})
	}
}

// orZero returns "0" for an empty string, so a table-driven case that
// leaves min or max unset can still pass a parseable literal through
// mustParse -- bounded ignores the value whenever the matching hasMin/
// hasMax is false, so what it parses to is immaterial.
func orZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func TestNewResult_ErrorPaths(t *testing.T) {
	if _, err := newResult("", verdict.OutcomeIndeterminate); err == nil {
		t.Fatal("newResult with empty checkID succeeded, want error")
	}
	if _, err := newResult("MU-01", verdict.Outcome(99)); err == nil {
		t.Fatal("newResult with invalid outcome succeeded, want error")
	}
}

func TestMustResult_Valid(t *testing.T) {
	res := mustResult("MU-01", verdict.OutcomePass)
	if res.CheckID() != "MU-01" {
		t.Errorf("CheckID() = %q, want MU-01", res.CheckID())
	}
	if res.Outcome() != verdict.OutcomePass {
		t.Errorf("Outcome() = %v, want PASS", res.Outcome())
	}
}

func TestMustResult_PanicsOnInvalidInput(t *testing.T) {
	// mustResult's panic branch is never taken by any legitimate call
	// site in this package -- every real call passes a fixed, valid
	// literal -- but the branch must still be real and tested, not
	// unreachable code the coverage bar should reject. This drives it
	// directly with the same two invalid inputs TestNewResult_ErrorPaths
	// uses against newResult.
	cases := []struct {
		name    string
		checkID string
		outcome verdict.Outcome
	}{
		{"empty checkID", "", verdict.OutcomeIndeterminate},
		{"invalid outcome", "MU-01", verdict.Outcome(99)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("mustResult(%q, %v) did not panic", tc.checkID, tc.outcome)
				}
			}()
			mustResult(tc.checkID, tc.outcome)
		})
	}
}

func TestOne(t *testing.T) {
	got := one()
	if got.Compare(mustParse(t, "1")) != 0 {
		t.Errorf("one() = %v, want a value comparing equal to Parse(%q)", got, "1")
	}
}

func TestMustParseDecimal_Valid(t *testing.T) {
	got := mustParseDecimal("1")
	if got.Compare(mustParse(t, "1")) != 0 {
		t.Errorf("mustParseDecimal(%q) = %v, want a value comparing equal to Parse(%q)", "1", got, "1")
	}
}

func TestMustParseDecimal_PanicsOnInvalidInput(t *testing.T) {
	// mustParseDecimal's panic branch is never taken by its one real
	// caller, one() (whose literal, "1", can never make decimal.Parse
	// fail) -- but the branch must still be real and tested, not
	// unreachable code the coverage bar should reject. This drives it
	// directly with text decimal.Parse itself rejects.
	defer func() {
		if recover() == nil {
			t.Fatal("mustParseDecimal(invalid) did not panic")
		}
	}()
	mustParseDecimal("not-a-number")
}

func TestPurity_NoForbiddenImports(t *testing.T) {
	entries, err := muSource.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir unexpected error: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := muSource.ReadFile(entry.Name())
		if err != nil {
			t.Fatalf("ReadFile(%s) unexpected error: %v", entry.Name(), err)
		}
		for _, line := range strings.Split(string(content), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, `"net"`) || strings.HasPrefix(trimmed, `"net/`) ||
				strings.HasPrefix(trimmed, `"os"`) || strings.HasPrefix(trimmed, `"os/`) ||
				strings.HasPrefix(trimmed, `"io"`) || strings.HasPrefix(trimmed, `"io/`) {
				t.Errorf("%s: forbidden import %s", entry.Name(), trimmed)
			}
		}
	}
}
