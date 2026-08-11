package mu

import (
	"github.com/tidalsoft/verdict/decimal"
	"github.com/tidalsoft/verdict/field"
	"github.com/tidalsoft/verdict/tables"
)

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

// convertBetweenUnits converts value, expressed in from, to the equivalent
// value expressed in to -- both already known to be the same dimension
// (every call site checks that itself, since what counts as an
// unresolvable dimension mismatch differs by check: MU-07 and MU-15 both
// treat it as INDETERMINATE, but only after resolving this far). This is
// the one place either check turns a value's own unit and the field's
// declared canonical_unit into a single comparable number -- MU-07's bound
// comparison and MU-15's round trip both go through it, so "how two units
// convert" is one rule rather than reimplemented per check with a chance
// to disagree at the edges (mirroring resolveQuantityUnit's own reason for
// existing as a shared file).
//
// When from and to are literally the same unit, this returns value
// unchanged rather than routing through the registry's fixed
// per-dimension canonical (kg, m, L, K, ...). That shortcut is not an
// optimisation; it is required for correctness. Unit.ToCanonical and
// Unit.FromCanonical are independently stored, not-necessarily-reciprocal
// decimal approximations (see Unit's own doc comment, and MU-15's
// Fahrenheit vector 97, which depends on exactly that asymmetry to
// produce a genuine FAIL at tolerance "0"). Routing a unit to itself
// through that detour -- lb -> kg -> lb, using lb's own imperfectly
// reciprocal factors -- manufactures a truncation error out of a
// conversion that never needed to happen at all: a bound or a round trip
// stated in the value's own already-declared unit (canonical_unit: lb
// against a value arriving in lb) must be exact, because no unit
// conversion is actually occurring. TestCheckMU07Quantity_
// CanonicalUnitNotRegistryCanonical_BoundEnforced and
// TestCheckMU15_CanonicalUnitNotRegistryCanonical_IdentityRoundTrip pin
// this directly.
//
// Where from and to differ, the registry canonical is the only conversion
// path a Unit exposes, so this goes through it: from.ToCanonical, then
// to.FromCanonical. Its one failure mode is either step's own
// (Unit.ToCanonical/FromCanonical's shared contract: an exponent-range
// overflow), collapsed into one error since every caller treats both
// identically -- INDETERMINATE, never an aborted evaluation (SPEC-MU §2.6
// does not let one check's arithmetic failure discard its siblings'
// results).
func convertBetweenUnits(from, to tables.Unit, value decimal.Decimal) (decimal.Decimal, error) {
	if from.Symbol() == to.Symbol() {
		return value, nil
	}
	canonical, err := from.ToCanonical(value)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return to.FromCanonical(canonical)
}
