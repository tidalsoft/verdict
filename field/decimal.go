package field

// DecimalDeclaration is the field declaration for `kind: decimal` (MU-02).
// This kind is the escape hatch for a field that legitimately carries more
// decimal places than any currency's minor unit
// permits -- an intermediate calculation value, for instance -- so that
// declaring it does not trigger MU-01/MU-14's money-specific scale checks.
// It carries no attributes of its own beyond the common ones (currently
// just NullSemantics): MU-02's precision_loss evaluation needs nothing
// from the declaration beyond the fact that the field is kind: decimal at
// all.
type DecimalDeclaration struct {
	common
}

// NewDecimalDeclaration returns a DecimalDeclaration declaring kind:
// decimal and nothing else. Chain WithNullSemantics to declare that
// attribute.
func NewDecimalDeclaration() DecimalDeclaration { return DecimalDeclaration{} }

// Kind implements Declaration.
func (d DecimalDeclaration) Kind() Kind { return KindDecimal }

// WithNullSemantics declares the field's null-vs-absent handling (MU-08).
// n must be NullSemanticsDistinct.
func (d DecimalDeclaration) WithNullSemantics(n NullSemantics) (DecimalDeclaration, error) {
	c, err := d.withNullSemantics(n)
	if err != nil {
		return DecimalDeclaration{}, err
	}
	d.common = c
	return d, nil
}
