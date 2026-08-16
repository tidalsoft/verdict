package mu

import (
	"github.com/tidalsoft/verdict"
	"github.com/tidalsoft/verdict/decimal"
)

// checkMU12 implements the total_reconciliation check (MU-12, SPEC-MU §5).
//
// MU-12 verifies that a declared total equals the sum of its components
// (plus any declared adjustments), within a declared absolute tolerance.
//
// Applicability (SPEC-MU §2.5.1: applies ruleset-level, gated on a
// `reconcile` entry naming this field as its `total`):
//   - Input.HasReconcile is false → not applicable. No reconcile entry
//     names this field, so there is nothing for this check to verify --
//     the ordinary case for the overwhelming majority of fields, which
//     never participate in a reconciliation at all.
//
// Branch matrix, once applicable -- every unmet requirement is
// INDETERMINATE, never PASS:
//   - the total path did not resolve to a single value at all -- absent
//     (vector 109), or resolved to a sequence rather than one value
//     (vector 115, "a total naming more than one location is not a
//     total") → INDETERMINATE.
//   - the total path resolved to a single value that did not coerce →
//     INDETERMINATE, reason value_not_coercible.
//   - the components path matched no element of the request at all
//     (Input.Components is nil) → INDETERMINATE (vector 110): "a total
//     reconciled against nothing has been verified by nothing," even
//     where the total itself is zero and an empty sum would agree with it
//     by coincidence.
//   - any component is a miss (an array element the wildcard matched, but
//     without the leaf the path names) → INDETERMINATE (vector 96): an
//     unread line item is a reconciliation that did not cover it.
//   - any component's leaf value did not coerce → INDETERMINATE, reason
//     value_not_coercible, reported against the component's own path
//     (vector 74).
//   - any adjustment sequence contains a miss → INDETERMINATE (vector
//     114). An adjustment path resolving to *no* value at all is a
//     different case entirely (see below) -- only a miss inside a
//     sequence that did match something is treated this way.
//   - any adjustment's leaf value did not coerce → INDETERMINATE, reason
//     value_not_coercible.
//   - sum(components) + sum(adjustments), computed entirely in exact
//     decimal arithmetic, differs from the total by more than the
//     declared (or default zero) tolerance → FAIL (vector 34).
//   - the difference is within tolerance → PASS (vectors 32, 33 -- the
//     latter, `[0.1, 0.2]` reconciling exactly against `0.3`, is "the
//     single most important test in this document," SPEC-MU §8.3, and
//     passes here only because this comparison never touches float64
//     anywhere in Input's construction or in decimal.Sum/
//     decimal.Reconciles).
//
// An adjustment path that resolved to no value at all -- Input.Adjustments'
// entry at that index is nil, meaning the path was simply absent, or (for
// a wildcarded adjustment path) matched no element of the request --
// contributes zero to the sum without making this check INDETERMINATE
// (vector 95): "the adjustments list is a set of optional addends." This
// is the one place Components and an individual Adjustments entry are
// treated asymmetrically for the identical "matched nothing" situation --
// see Input's own doc comment for why, and SPEC-MU §5's own text for the
// same distinction stated normatively.
//
// MU-12 is value-dependent in the sense that it reads several fields'
// values as numbers, but §2.6.3's coercion gate is stated per-field for a
// single Value/ValueCoercionFailed pair; MU-12 instead reads several
// already-coerced-or-not SequenceElement values directly from Input,
// exactly as documented on Input.TotalResolved/TotalValue/
// TotalCoercionFailed/Components/Adjustments.
func checkMU12(in Input) (verdict.Result, bool, error) {
	if !in.HasReconcile {
		return notApplicable()
	}

	if !in.TotalResolved {
		return indeterminateResult("MU-12")
	}
	if in.TotalCoercionFailed {
		return indeterminateResult("MU-12")
	}
	if in.Components == nil {
		return indeterminateResult("MU-12")
	}

	sum := decimal.Decimal{} // the exact decimal 0

	componentSum, ok := sumSequence(in.Components, sum)
	if !ok {
		return indeterminateResult("MU-12")
	}
	sum = componentSum

	for _, adjustment := range in.Adjustments {
		adjustmentSum, ok := sumSequence(adjustment, sum)
		if !ok {
			return indeterminateResult("MU-12")
		}
		sum = adjustmentSum
	}

	reconciles, err := decimal.Reconciles(sum, in.TotalValue, in.Reconcile.Tolerance())
	if err != nil {
		// decimal.Add/Sub's only failure mode is an exponent-range
		// overflow (see decimal.Decimal's own doc comments) -- an
		// extreme but legitimately parseable input, not a programming
		// error. SPEC-MU §2.6 does not let one check's arithmetic
		// failure discard its siblings' results, so this is
		// INDETERMINATE for MU-12 alone, never an aborting error, exactly
		// as checkMU07Money and checkMU15 already treat the identical
		// class of failure in their own arithmetic.
		return indeterminateResult("MU-12")
	}
	if !reconciles {
		return failResult("MU-12")
	}
	return passResult("MU-12")
}

// sumSequence adds every non-miss, coercible element of seq to running,
// in order, and reports ok == false the moment it finds a miss or an
// uncoercible element -- SPEC-MU §5's rule that either one makes the whole
// reconciliation INDETERMINATE, whether the sequence in question is
// Input.Components or one entry of Input.Adjustments (see checkMU12's own
// doc comment for the one place the two are treated differently: an empty
// or nil seq itself, which this function never sees for Components,
// checkMU12 having already handled that case, and which is exactly the
// "contributes zero" case for an Adjustments entry -- an empty seq here
// simply adds nothing and returns running unchanged, ok true).
func sumSequence(seq []SequenceElement, running decimal.Decimal) (decimal.Decimal, bool) {
	for _, elem := range seq {
		if elem.Miss || elem.CoercionFailed {
			return decimal.Decimal{}, false
		}
		sum, err := running.Add(elem.Value)
		if err != nil {
			return decimal.Decimal{}, false
		}
		running = sum
	}
	return running, true
}
