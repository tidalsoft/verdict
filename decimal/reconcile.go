package decimal

// Sum returns the exact sum of values, or the exact zero Decimal if values
// is empty. SPEC-MU MU-12 (total_reconciliation) needs the sum of an
// arbitrary-length line-item collection plus an adjustments collection,
// computed in exact decimal arithmetic throughout; Sum folds Add across the
// whole collection so no intermediate step touches float64.
func Sum(values ...Decimal) (Decimal, error) {
	total := Decimal{}
	for _, v := range values {
		var err error
		total, err = total.Add(v)
		if err != nil {
			return Decimal{}, err
		}
	}
	return total, nil
}

// Reconciles reports whether computed and total agree within tolerance:
// abs(computed - total) <= tolerance, evaluated entirely in exact decimal
// arithmetic. This is SPEC-MU MU-12's own comparison — "compute sum(...) in
// exact decimal arithmetic[;] difference from total exceeding tolerance
// fails" — and CLAUDE.md invariant #2 ("money never floats") applies to it
// directly: SPEC-MU §8 vector 33 ([0.1, 0.2] reconciling to 0.3) passes only
// because this comparison never converts through float64.
//
// tolerance is typically the exact zero Decimal (SPEC-MU MU-12's documented
// default) but callers may pass a positive tolerance to accommodate
// line-level rounding; tolerance must not be negative, and Reconciles does
// not defend against that — a negative tolerance is a caller error to
// reject before calling this, not a condition for this function to guess
// about.
func Reconciles(computed, total, tolerance Decimal) (bool, error) {
	delta, err := computed.Sub(total)
	if err != nil {
		return false, err
	}
	return delta.Abs().Compare(tolerance) <= 0, nil
}
