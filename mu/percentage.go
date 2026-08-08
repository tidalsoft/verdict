package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/field"
)

// checkMU13 implements the percentage_domain check (MU-13, SPEC-MU §3).
//
// MU-13 catches confusion between fractional (0-1) and percentage (0-100)
// representations.
//
// Branch matrix:
//   - no declaration for the field, or a declaration whose kind is not
//     percentage → INDETERMINATE (the "Requires kind: percentage" clause).
//   - percentage declaration with no domain declared → INDETERMINATE.
//   - domain: unit_interval and value > 1 → FAIL at the class default
//     (block).
//   - domain: unit_interval and value ≤ 1 → PASS.
//   - domain: hundred and value ≤ 1 and value ≠ 0 → FAIL at severity warn
//     only -- asymmetric by design. SPEC-MU §3: "A legitimate 0.5% exists;
//     this is genuinely ambiguous in that direction." This is the one
//     branch in this package that does not use the check's class-default
//     severity; see warnResult (mu.go).
//   - domain: hundred and (value > 1 or value == 0) → PASS.
func checkMU13(in Input) (verdict.Result, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return indeterminateResult("MU-13")
	}
	pctDecl, ok := decl.(field.PercentageDeclaration)
	if !ok {
		return indeterminateResult("MU-13")
	}
	domain, hasDomain := pctDecl.Domain()
	if !hasDomain {
		return indeterminateResult("MU-13")
	}

	unitBoundary := one()
	if domain == field.DomainUnitInterval {
		if in.Value.Compare(unitBoundary) > 0 {
			return failResult("MU-13")
		}
		return passResult("MU-13")
	}

	// domain is field.DomainHundred: WithDomain rejects every Domain other
	// than DomainUnitInterval and DomainHundred, and hasDomain == true
	// confirms one of those two was actually set, so the unit_interval
	// branch above and this one are exhaustive -- there is no third arm.
	if in.Value.Compare(unitBoundary) <= 0 && !in.Value.IsZero() {
		return warnResult("MU-13", verdict.OutcomeFail)
	}
	return passResult("MU-13")
}
