package mu

import (
	"testing"

	"github.com/tidalsoft/verdict/field"
)

func TestResolveQuantityUnit(t *testing.T) {
	cases := []struct {
		name         string
		decl         field.QuantityDeclaration
		embedded     string
		hasEmbedded  bool
		vals         map[string]field.Value
		wantSymbol   string
		wantOK       bool
		wantConflict bool
	}{
		{
			name:        "embedded only",
			decl:        field.NewQuantityDeclaration(),
			embedded:    "lb",
			hasEmbedded: true,
			wantSymbol:  "lb",
			wantOK:      true,
		},
		{
			name:       "unit_field only",
			decl:       mustUnitField(t, field.NewQuantityDeclaration()),
			vals:       map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
			wantSymbol: "kg",
			wantOK:     true,
		},
		{
			name:        "both agree",
			decl:        mustUnitField(t, field.NewQuantityDeclaration()),
			embedded:    "kg",
			hasEmbedded: true,
			vals:        map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
			wantSymbol:  "kg",
			wantOK:      true,
		},
		{
			name:         "both disagree -> conflict",
			decl:         mustUnitField(t, field.NewQuantityDeclaration()),
			embedded:     "lb",
			hasEmbedded:  true,
			vals:         map[string]field.Value{"arguments.unit": field.NewStringValue("kg")},
			wantConflict: true,
		},
		{
			name: "neither source present",
			decl: field.NewQuantityDeclaration(),
		},
		{
			name: "unit_field declared but sibling absent from Vals",
			decl: mustUnitField(t, field.NewQuantityDeclaration()),
			vals: map[string]field.Value{},
		},
		{
			name: "unit_field resolves to a non-string Value",
			decl: mustUnitField(t, field.NewQuantityDeclaration()),
			vals: map[string]field.Value{"arguments.unit": field.NewNonComparableValue()},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := Input{
				EmbeddedUnit:    tc.embedded,
				HasEmbeddedUnit: tc.hasEmbedded,
				Vals:            tc.vals,
			}
			got := resolveQuantityUnit(in, tc.decl)
			if got.ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", got.ok, tc.wantOK)
			}
			if got.conflict != tc.wantConflict {
				t.Errorf("conflict = %v, want %v", got.conflict, tc.wantConflict)
			}
			if tc.wantOK && got.symbol != tc.wantSymbol {
				t.Errorf("symbol = %q, want %q", got.symbol, tc.wantSymbol)
			}
		})
	}
}
