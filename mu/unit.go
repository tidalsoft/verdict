package mu

import "github.com/tidalsoft/verdict/field"

// quantityUnit is the outcome of resolving a `kind: quantity` field's
// unit from its two possible sources (SPEC-MU §2.6.1): the value's own
// embedded unit (Input.EmbeddedUnit, decomposed upstream from text like
// "12 lb") and unit_field, a sibling path resolved through Input.Vals.
// Exactly one of ok or conflict is true whenever the unit did not resolve
// to a single agreed symbol; both are false only when neither source
// supplied anything at all.
type quantityUnit struct {
	symbol   string
	ok       bool
	conflict bool
}

// resolveQuantityUnit resolves decl's unit against in -- the one
// resolution helper this package's every quantity check (MU-04, MU-05,
// MU-07's quantity branch, MU-15) shares, so that "two sources naming
// different units" is one rule with one behaviour everywhere it applies,
// never reimplemented per check with a chance to disagree at the edges
// (this file exists because of exactly that hazard -- see its callers'
// own doc comments for the consumer trace this package's task record
// requires).
//
// SPEC-MU §2.6.1: "Where a value decomposes to a unit part and unit_field
// also resolves to a unit, and the two are not the same unit, the field's
// unit does not resolve... Neither source wins... Where only one source
// is present, or where both are present and name the same unit, the unit
// resolves normally." unit_field's own resolution is shape, not
// membership -- it resolves to a unit whenever the sibling path yields a
// JSON string, via Input.Vals's field.Value contract (see Input's own doc
// comment) -- exactly mirroring how a value's embedded unit resolves,
// regardless of whether either string is one the unit registry
// recognises (SPEC-MU §2.6.1's closing paragraph, vector 121: "the
// conflicting unit_field string need not itself be registry-recognised").
// Registry membership is each individual check's own next question, asked
// only after this function has resolved (or failed to resolve) a single
// agreed symbol.
func resolveQuantityUnit(in Input, decl field.QuantityDeclaration) quantityUnit {
	embedded, hasEmbedded := in.EmbeddedUnit, in.HasEmbeddedUnit

	var fromField string
	var hasFromField bool
	if path, has := decl.UnitField(); has {
		if v, ok := in.Vals[path]; ok {
			fromField, hasFromField = v.StringValue()
		}
	}

	switch {
	case hasEmbedded && hasFromField:
		if embedded != fromField {
			return quantityUnit{conflict: true}
		}
		return quantityUnit{symbol: embedded, ok: true}
	case hasEmbedded:
		return quantityUnit{symbol: embedded, ok: true}
	case hasFromField:
		return quantityUnit{symbol: fromField, ok: true}
	default:
		return quantityUnit{}
	}
}
