package verdict

// Version is this module's release version. It is a plain string constant
// rather than a build-time injected value because a library package has no
// linker-flag injection point of its own; callers that need a build-time
// version read this constant and may combine it with their own build
// metadata.
//
// The value follows semver and is bumped as part of the commit that changes
// this package's behaviour in a way third-party importers would care about.
// Before the first tagged release it stays at "0.0.0-dev".
const Version = "0.0.0-dev"
