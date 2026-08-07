package field

// Declaration is the field-level schema a ruleset supplies for a single
// argument path, e.g. "arguments.amount". It is a closed
// set of six concrete types -- MoneyDeclaration, QuantityDeclaration,
// TimestampDeclaration, PercentageDeclaration, DecimalDeclaration, and
// IdentifierDeclaration, one per Kind -- never implemented outside this
// package. A caller type-switches on the concrete type (using Kind() to
// choose the branch, or a Go type switch directly) to reach the
// kind-specific accessors that live only on the concrete types; every such
// accessor follows the comma-ok idiom so that an attribute the ruleset
// never mentioned is distinguishable from one explicitly set to its zero
// value -- see the package doc comment.
type Declaration interface {
	// Kind reports which concrete Declaration type this value is.
	Kind() Kind

	// NullSemantics reports the field's null-vs-absent handling (MU-08
	// null_vs_absent), if declared, and whether it was declared at
	// all. This is the one attribute available on every Kind: MU-08's
	// requirement -- an explicit JSON null on a field where omission and
	// null carry different meanings to the target system -- does not
	// depend on the field's magnitude kind, so every concrete type in this
	// package implements it identically via the embedded common type.
	NullSemantics() (NullSemantics, bool)
}
