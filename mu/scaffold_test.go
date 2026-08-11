package mu

import (
	"embed"
	"errors"
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
// given checkID, for exercising evaluateChecks' aggregation with injected
// outcomes.
func fixedResult(checkID string, outcome verdict.Outcome) OnFunc {
	return func(_ Input) (verdict.Result, error) {
		return newResult(checkID, outcome)
	}
}

func TestChecksFor_DispatchTable(t *testing.T) {
	cases := []struct {
		name string
		kind field.Kind
		want []string
		ok   bool
	}{
		{"money", field.KindMoney, []string{"MU-01", "MU-02", "MU-03", "MU-06", "MU-07", "MU-14"}, true},
		{"decimal", field.KindDecimal, []string{"MU-02", "MU-06", "MU-07"}, true},
		{"percentage", field.KindPercentage, []string{"MU-07", "MU-13"}, true},
		{"quantity", field.KindQuantity, []string{"MU-04", "MU-05", "MU-07", "MU-15"}, true},
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checks, ok := checksFor(tc.kind)
			if ok != tc.ok {
				t.Fatalf("checksFor(%v) ok = %v, want %v", tc.kind, ok, tc.ok)
			}
			if ok != (len(checks) > 0) {
				t.Fatalf("checksFor(%v): ok = %v but len(checks) = %d -- ok must equal len(checks) > 0", tc.kind, ok, len(checks))
			}
			if !ok {
				return
			}
			if len(checks) != len(tc.want) {
				t.Fatalf("checksFor(%v) = %d checks, want %d", tc.kind, len(checks), len(tc.want))
			}
			for i, check := range checks {
				res, err := check(Input{})
				if err != nil {
					t.Fatalf("check %d unexpected error: %v", i, err)
				}
				if res.CheckID() != tc.want[i] {
					t.Errorf("check %d CheckID() = %q, want %q", i, res.CheckID(), tc.want[i])
				}
				if res.Outcome() != verdict.OutcomeIndeterminate {
					t.Errorf("check %d Outcome() = %v, want INDETERMINATE", i, res.Outcome())
				}
			}
		})
	}
}

func TestEvaluate_DispatchPerKind(t *testing.T) {
	// Every case below declares its kind but nothing else, so every check
	// the kind carries is unsatisfied and returns INDETERMINATE -- except
	// MU-02 (precision.go), whose only requirement is the field's kind
	// (money or decimal) being one it applies to at all: with Value's zero
	// value (exact 0) and Provenance's zero value (FromString), MU-02
	// evaluates for real and PASSes, since decimal.PrecisionLoss never
	// fails a FromString value. Evaluate (§2.6: no short-circuiting) must
	// return every applicable check's result, in ascending check-ID order,
	// not just the first.
	cases := []struct {
		name         string
		decl         field.Declaration
		wantIDs      []string
		wantOutcomes []verdict.Outcome
	}{
		{
			"money",
			field.NewMoneyDeclaration(),
			[]string{"MU-01", "MU-02", "MU-03", "MU-06", "MU-07", "MU-14"},
			[]verdict.Outcome{
				verdict.OutcomeIndeterminate, verdict.OutcomePass, verdict.OutcomeIndeterminate,
				verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate,
			},
		},
		{
			"decimal",
			field.NewDecimalDeclaration(),
			[]string{"MU-02", "MU-06", "MU-07"},
			[]verdict.Outcome{verdict.OutcomePass, verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate},
		},
		{
			"percentage",
			field.NewPercentageDeclaration(),
			[]string{"MU-07", "MU-13"},
			[]verdict.Outcome{verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate},
		},
		{
			"quantity",
			field.NewQuantityDeclaration(),
			[]string{"MU-04", "MU-05", "MU-07", "MU-15"},
			[]verdict.Outcome{verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate, verdict.OutcomeIndeterminate},
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
	boom := func(_ Input) (verdict.Result, error) {
		return verdict.Result{}, errors.New("boom")
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
			res, err := bounded(tc.decl, mustParse(t, tc.value), mustParse(t, orZero(tc.min)), tc.hasMin, mustParse(t, orZero(tc.max)), tc.hasMax)
			if err != nil {
				t.Fatalf("bounded unexpected error: %v", err)
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
