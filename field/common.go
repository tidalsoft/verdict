package field

import "fmt"

// NullSemantics distinguishes how a field treats an explicit JSON null from
// an omitted field (SPEC-MU MU-08 null_vs_absent).
type NullSemantics int

const (
	// NullSemanticsUnspecified is the zero value: the field carries no
	// null_semantics declaration at all. MU-08 must tell this apart from
	// NullSemanticsDistinct -- see common.NullSemantics's comma-ok return.
	NullSemanticsUnspecified NullSemantics = iota
	// NullSemanticsDistinct marks a field where an explicit null and an
	// omitted field carry different meanings to the target system
	// (typically: omitted means "leave unchanged", null means "clear the
	// value"). It is the only value SPEC-MU §2.3 names.
	NullSemanticsDistinct
)

// String renders the null-semantics value using the vocabulary SPEC-MU
// §2.3 uses (`null_semantics: distinct`).
func (n NullSemantics) String() string {
	if n == NullSemanticsDistinct {
		return "distinct"
	}
	return "unspecified"
}

func (n NullSemantics) valid() bool {
	return n == NullSemanticsDistinct
}

// common holds the one declaration attribute SPEC-MU §2.3 does not scope
// to a single Kind (see the package doc comment). It is embedded, never
// used as a standalone value, by every concrete Declaration type in this
// package.
type common struct {
	nullSemantics    NullSemantics
	hasNullSemantics bool
}

// NullSemantics implements Declaration for every type that embeds common.
func (c common) NullSemantics() (NullSemantics, bool) {
	return c.nullSemantics, c.hasNullSemantics
}

// withNullSemantics returns a copy of c with null_semantics set to n, or an
// error if n is not NullSemanticsDistinct. It is unexported: each concrete
// Declaration type exposes its own WithNullSemantics method, returning its
// own type, that just forwards to this helper so the fluent chain never
// leaves the concrete type.
func (c common) withNullSemantics(n NullSemantics) (common, error) {
	if !n.valid() {
		return common{}, fmt.Errorf("field: invalid null_semantics %v", n)
	}
	c.nullSemantics = n
	c.hasNullSemantics = true
	return c, nil
}
