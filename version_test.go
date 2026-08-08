package verdict

import (
	"regexp"
	"testing"
)

// TestVersion proves the test harness actually wires up and executes: it is
// a genuine assertion on the one piece of real content the scaffold ships
// (see doc.go), not a placeholder.
func TestVersion(t *testing.T) {
	if Version == "" {
		t.Fatal("Version must not be empty")
	}
	semverRe := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	if !semverRe.MatchString(Version) {
		t.Fatalf("Version = %q, want a semver string of the form X.Y.Z", Version)
	}
}
