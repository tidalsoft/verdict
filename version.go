package verdict

// Version is this module's release version. It is a plain string constant
// rather than a build-time injected value because a library package has no
// linker-flag injection point of its own; callers that need a build-time
// version read this constant and may combine it with their own build
// metadata.
//
// The value follows semver. It always reads the latest released version and
// is bumped atomically with the tag via "make release".
const Version = "0.1.0"
