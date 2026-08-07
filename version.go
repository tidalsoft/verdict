package engine

// Version is the engine package's release version, tracked independently of
// the platform module's own version because engine/ is what gets tagged and
// published as a standalone Apache 2.0 module at task 2-18. It is a plain
// string constant rather than a build-time injected value because a library
// package has no linker-flag injection point of its own; callers that need a
// build-time version (the CLI, the hosted service) read this constant and
// may combine it with their own build metadata.
//
// The value follows semver and is bumped as part of the commit that changes
// engine behaviour in a way third-party importers would care about. Before
// the first tagged release it stays at "0.0.0-dev".
const Version = "0.0.0-dev"
