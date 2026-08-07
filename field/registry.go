package field

import (
	"errors"
	"fmt"
)

// Registry is the set of field declarations a ruleset supplies, keyed by
// argument path (e.g. "arguments.amount"). It makes the field-level
// absence rule concrete: where a declaration is absent, Class D checks
// depending on it return INDETERMINATE. They do not guess. Lookup's second
// return value is that absence signal, true only for a path the Registry
// was actually built with.
//
// Registry's zero value (an empty Registry{}) is itself usable and behaves
// as "no field in the ruleset carries a declaration" -- every Lookup
// returns (nil, false) -- which matters because it means a caller that
// simply never constructs a Registry (e.g. a ruleset with no `fields:`
// section at all) gets fail-safe INDETERMINATE-everywhere behaviour rather
// than a nil-map panic.
//
// A Registry is immutable once built: NewRegistry copies its input map, so
// a caller retaining and later mutating the map it passed in has no effect
// on the Registry already returned. This package holds no package-level
// state of its own -- see the package doc comment.
type Registry struct {
	declarations map[string]Declaration
}

// NewRegistry builds a Registry from a set of field-path -> Declaration
// pairs. Every path must be non-empty and every Declaration must be
// non-nil.
func NewRegistry(decls map[string]Declaration) (Registry, error) {
	copied := make(map[string]Declaration, len(decls))
	for path, decl := range decls {
		if path == "" {
			return Registry{}, errors.New("field: registry: field path must not be empty")
		}
		if decl == nil {
			return Registry{}, fmt.Errorf("field: registry: field %q: declaration must not be nil", path)
		}
		copied[path] = decl
	}
	return Registry{declarations: copied}, nil
}

// Lookup returns the Declaration supplied for fieldPath, and whether one
// was supplied at all. A false second return value is the "declaration
// absent" case Class D checks must treat as INDETERMINATE, never as a
// default: it is not distinguishable from any
// particular Declaration value, because no such sentinel value is ever
// stored here -- the only way a path is absent is that it was never passed
// to NewRegistry.
func (r Registry) Lookup(fieldPath string) (Declaration, bool) {
	decl, ok := r.declarations[fieldPath]
	return decl, ok
}
