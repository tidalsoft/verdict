package engine

import "testing"

// TestVersion proves the test harness actually wires up and executes: it is
// a genuine assertion on the one piece of real content the scaffold ships
// (see doc.go), not a placeholder.
func TestVersion(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	if Version != "0.0.0-dev" {
		t.Fatalf("Version = %q, want the pre-release placeholder %q until the first tagged release", Version, "0.0.0-dev")
	}
}
