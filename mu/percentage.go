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
// # This check reads magnitude, and ignores sign
//
// SPEC-MU §3 is explicit and normative on this point: "This check tests
// the magnitude of the value against the declared domain, and reads
// abs(value) in every condition below. Sign is not its subject." An
// earlier reading tested the raw signed value in each domain, which made
// the two domains disagree about sign for no principled reason -- a
// unit_interval field's `value > 1` test let any negative value pass
// regardless of magnitude, while a hundred field's `value ≤ 1 and value ≠
// 0` test caught negatives incidentally, at warn, with a message about
// sub-1% ambiguity that had nothing to do with what was actually wrong.
// Reading abs(value) uniformly, on both domains, settles the disagreement
// in one move: a signed percentage rate (a -5% change) is exactly what a
// percentage field routinely carries, and the declaration such a value
// violates, if any, is one about sign -- MU-06's territory, not this
// check's (vectors 75, 76, 94).
//
// Applicability (SPEC-MU §2.5.1: applies to percentage, no further gate):
//   - no declaration for the field, or a declaration whose kind is not
//     percentage → not applicable.
//
// Branch matrix, once applicable:
//   - the value is not coercible (§2.6.3; MU-13 is value-dependent) →
//     INDETERMINATE, reason value_not_coercible.
//   - percentage declaration with no domain declared → INDETERMINATE
//     (§2.5.2: a percentage has a domain whether or not the ruleset says
//     which -- a required input, not a gate).
//   - domain: unit_interval and abs(value) > 1 → FAIL at the class
//     default (block) (vector 75).
//   - domain: unit_interval and abs(value) ≤ 1 → PASS (vector 76).
//   - domain: hundred and 0 < abs(value) ≤ 1 → FAIL at severity warn
//     only -- asymmetric by design. SPEC-MU §3: "A legitimate 0.5% exists;
//     this is genuinely ambiguous in that direction." This is the one
//     branch in this package that does not use the check's class-default
//     severity; see warnResult (mu.go). Vector 94 pins this with a
//     negative input: -0.5's magnitude of 0.5 is in the ambiguous band
//     exactly as 0.5 itself is.
//   - value = 0 → PASS under both domains -- "0" and "0%" are the same
//     claim, and abs(0) = 0 makes this fall out of the conditions above
//     without a separate branch.
//   - domain: hundred and abs(value) > 1 → PASS, with no upper bound of
//     any kind (vector 77).
func checkMU13(in Input) (verdict.Result, bool, error) {
	decl, ok := in.Registry.Lookup(in.Field)
	if !ok {
		return notApplicable()
	}
	pctDecl, ok := decl.(field.PercentageDeclaration)
	if !ok {
		return notApplicable()
	}
	if in.ValueCoercionFailed {
		return indeterminateResult("MU-13")
	}
	domain, hasDomain := pctDecl.Domain()
	if !hasDomain {
		return indeterminateResult("MU-13")
	}

	magnitude := in.Value.Abs()
	unitBoundary := one()
	if domain == field.DomainUnitInterval {
		if magnitude.Compare(unitBoundary) > 0 {
			return failResult("MU-13")
		}
		return passResult("MU-13")
	}

	// domain is field.DomainHundred: WithDomain rejects every Domain other
	// than DomainUnitInterval and DomainHundred, and hasDomain == true
	// confirms one of those two was actually set, so the unit_interval
	// branch above and this one are exhaustive -- there is no third arm.
	if magnitude.Compare(unitBoundary) <= 0 && !in.Value.IsZero() {
		return warnResult("MU-13", verdict.OutcomeFail)
	}
	return passResult("MU-13")
}
