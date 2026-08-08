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
		{"money", field.KindMoney, []string{"MU-01", "MU-14", "MU-02", "MU-03", "MU-06", "MU-07"}, true},
		{"decimal", field.KindDecimal, []string{"MU-02"}, true},
		{"percentage", field.KindPercentage, []string{"MU-13"}, true},
		{"quantity", field.KindQuantity, []string{"MU-07"}, true},
		{"unspecified", field.KindUnspecified, nil, false},
		{"timestamp", field.KindTimestamp, nil, false},
		{"identifier", field.KindIdentifier, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checks, ok := checksFor(tc.kind)
			if ok != tc.ok {
				t.Fatalf("checksFor(%v) ok = %v, want %v", tc.kind, ok, tc.ok)
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
	cases := []struct {
		name string
		decl field.Declaration
		want string
	}{
		{"money", field.NewMoneyDeclaration(), "MU-01"},
		{"decimal", field.NewDecimalDeclaration(), "MU-02"},
		{"percentage", field.NewPercentageDeclaration(), "MU-13"},
		{"quantity", field.NewQuantityDeclaration(), "MU-07"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				Field:    "arguments.amount",
				Registry: mustRegistry(t, tc.decl),
			}
			res, err := Evaluate(in)
			if err != nil {
				t.Fatalf("Evaluate unexpected error: %v", err)
			}
			if res.CheckID() != tc.want {
				t.Errorf("Evaluate CheckID() = %q, want %q", res.CheckID(), tc.want)
			}
			if res.Outcome() != verdict.OutcomeIndeterminate {
				t.Errorf("Evaluate Outcome() = %v, want INDETERMINATE", res.Outcome())
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
	res, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate with zero-value Registry unexpected error: %v", err)
	}
	if res.Outcome() != verdict.OutcomeIndeterminate {
		t.Errorf("Evaluate Outcome() = %v, want INDETERMINATE", res.Outcome())
	}
	if res.CheckID() != checkIDNoDeclaration {
		t.Errorf("Evaluate CheckID() = %q, want %q", res.CheckID(), checkIDNoDeclaration)
	}
}

func TestEvaluateChecks_Aggregation(t *testing.T) {
	cases := []struct {
		name   string
		checks []OnFunc
		wantID string
		want   verdict.Outcome
	}{
		{
			name:   "first FAIL wins over earlier INDETERMINATE",
			checks: []OnFunc{fixedResult("MU-01", verdict.OutcomeIndeterminate), fixedResult("MU-14", verdict.OutcomeFail)},
			wantID: "MU-14",
			want:   verdict.OutcomeFail,
		},
		{
			name:   "first FAIL wins over earlier PASS",
			checks: []OnFunc{fixedResult("MU-01", verdict.OutcomePass), fixedResult("MU-02", verdict.OutcomeFail)},
			wantID: "MU-02",
			want:   verdict.OutcomeFail,
		},
		{
			name:   "first INDETERMINATE wins when no FAIL",
			checks: []OnFunc{fixedResult("MU-01", verdict.OutcomePass), fixedResult("MU-14", verdict.OutcomeIndeterminate), fixedResult("MU-02", verdict.OutcomePass)},
			wantID: "MU-14",
			want:   verdict.OutcomeIndeterminate,
		},
		{
			name:   "all PASS returns first PASS",
			checks: []OnFunc{fixedResult("MU-01", verdict.OutcomePass), fixedResult("MU-14", verdict.OutcomePass)},
			wantID: "MU-01",
			want:   verdict.OutcomePass,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := evaluateChecks(Input{}, tc.checks)
			if err != nil {
				t.Fatalf("evaluateChecks unexpected error: %v", err)
			}
			if res.CheckID() != tc.wantID {
				t.Errorf("CheckID() = %q, want %q", res.CheckID(), tc.wantID)
			}
			if res.Outcome() != tc.want {
				t.Errorf("Outcome() = %v, want %v", res.Outcome(), tc.want)
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
	_, err := evaluateChecks(Input{}, nil)
	if err == nil {
		t.Fatal("evaluateChecks with no checks succeeded, want error")
	}
}

func TestNormalizeCurrency(t *testing.T) {
	tbl := tables.NewISO4217Table()
	cases := []struct {
		code string
		want string
		ok   bool
	}{
		{"USD", "USD", true},
		{"usd", "USD", true},
		{"Usd", "USD", true},
		{"jpy", "JPY", true},
		{"ZZZ", "ZZZ", false},
		{"", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got, ok := (Tables{ISO4217: tbl}).normalizeCurrency(tc.code)
			if got != tc.want || ok != tc.ok {
				t.Errorf("normalizeCurrency(%q) = (%q, %v), want (%q, %v)", tc.code, got, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestBounded(t *testing.T) {
	tbl := tables.NewISO4217Table()
	usd, _ := tbl.Lookup("USD")
	jpy, _ := tbl.Lookup("JPY")
	xau, _ := tbl.Lookup("XAU")

	cases := []struct {
		name     string
		decl     field.MoneyDeclaration
		value    string
		currency tables.Currency
		want     verdict.Outcome
	}{
		{
			name:     "no bounds declared -> INDETERMINATE",
			decl:     field.NewMoneyDeclaration(),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomeIndeterminate,
		},
		{
			name:     "currency without minor-unit exponent -> INDETERMINATE",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "0")),
			value:    "49.99",
			currency: xau,
			want:     verdict.OutcomeIndeterminate,
		},
		{
			name:     "below min -> FAIL",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "50")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomeFail,
		},
		{
			name:     "above max -> FAIL",
			decl:     field.NewMoneyDeclaration().WithMax(mustParse(t, "50")),
			value:    "50.01",
			currency: usd,
			want:     verdict.OutcomeFail,
		},
		{
			name:     "within bounds -> PASS",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "0")).WithMax(mustParse(t, "100")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomePass,
		},
		{
			name:     "inclusive min equality -> PASS",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "49.99")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomePass,
		},
		{
			name:     "inclusive max equality -> PASS",
			decl:     field.NewMoneyDeclaration().WithMax(mustParse(t, "49.99")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomePass,
		},
		{
			name:     "exclusive min equality -> FAIL",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "49.99")).WithExclusiveMin(),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomeFail,
		},
		{
			name:     "exclusive max equality -> FAIL",
			decl:     field.NewMoneyDeclaration().WithMax(mustParse(t, "49.99")).WithExclusiveMax(),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomeFail,
		},
		{
			name:     "only min bound, value above -> PASS",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "0")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomePass,
		},
		{
			name:     "only max bound, value below -> PASS",
			decl:     field.NewMoneyDeclaration().WithMax(mustParse(t, "100")),
			value:    "49.99",
			currency: usd,
			want:     verdict.OutcomePass,
		},
		{
			name:     "JPY exponent 0 normalization",
			decl:     field.NewMoneyDeclaration().WithMin(mustParse(t, "500")),
			value:    "499",
			currency: jpy,
			want:     verdict.OutcomeFail,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := bounded(tc.decl, mustParse(t, tc.value), tc.currency)
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

func TestBounded_ScaleByExponentError(t *testing.T) {
	usd, _ := tables.NewISO4217Table().Lookup("USD")
	decl := field.NewMoneyDeclaration().WithMin(mustParse(t, "0"))
	huge := mustParse(t, "999"+strings.Repeat("0", 99997))
	_, err := bounded(decl, huge, usd)
	if err == nil {
		t.Fatal("bounded with value overflowing ScaleByExponent succeeded, want error")
	}
}

func TestBounded_BoundScaleError(t *testing.T) {
	usd, _ := tables.NewISO4217Table().Lookup("USD")
	huge := mustParse(t, "999"+strings.Repeat("0", 99997))

	minDecl := field.NewMoneyDeclaration().WithMin(huge)
	if _, err := bounded(minDecl, mustParse(t, "1"), usd); err == nil {
		t.Fatal("bounded with min overflowing ScaleByExponent succeeded, want error")
	}

	maxDecl := field.NewMoneyDeclaration().WithMax(huge)
	if _, err := bounded(maxDecl, mustParse(t, "1"), usd); err == nil {
		t.Fatal("bounded with max overflowing ScaleByExponent succeeded, want error")
	}
}

func TestNewResult_ErrorPaths(t *testing.T) {
	if _, err := newResult("", verdict.OutcomeIndeterminate); err == nil {
		t.Fatal("newResult with empty checkID succeeded, want error")
	}
	if _, err := newResult("MU-01", verdict.Outcome(99)); err == nil {
		t.Fatal("newResult with invalid outcome succeeded, want error")
	}
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
