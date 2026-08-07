# Contributing

## Developer Certificate of Origin, not a CLA

Contributions are accepted under the [Developer Certificate of Origin
(DCO)](https://developercertificate.org/), not a Contributor License
Agreement. You retain copyright in your contribution and
license it under Apache 2.0, the same license as the rest of this project.

Certify each commit by adding a `Signed-off-by` trailer with your real name
and email:

```
Signed-off-by: Jane Doe <jane@example.com>
```

`git commit -s` adds this for you. A pull request whose commits are missing
this trailer will not be merged.

## Adding a new catalogue rule

New catalogue entries (checks and gates) require a benchmark case
demonstrating the failure they catch: a contribution arrives with a
benchmark case, or it does not arrive. A rule with no evidence of what it
prevents is not reviewable.

## The MU-/PG-/PC- namespace is reserved

The `MU-`, `PG-`, and `PC-` rule ID prefixes are reserved to the
specification maintainer; the MU and PG rule catalogues remain under the
maintainer's editorial control even though this implementation is
Apache-licensed. A contributed rule under one of these prefixes will not be
accepted unless it corresponds to a change already made to the governing
specification.

If you're forking this project or adding your own rules, use your own
prefix. Fleet-scale measurement data is keyed by rule ID; a fork that
redefines or collides with an existing ID makes that data incomparable.

## Before you send a pull request

- Run the full validation gate: `make check`. It must pass, including
  100% file/package/total test coverage — there are no exclusions in this
  module.
- Keep the purity invariant: no network, filesystem, or wall-clock access
  anywhere in this module. `make lint` enforces this mechanically via a
  `depguard` rule; see `doc.go`.
