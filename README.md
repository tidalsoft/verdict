# verdict

`github.com/tidalsoft/verdict` is a deterministic evaluation engine for AI
agent tool calls: magnitude/unit checks, precondition gates, and
postcondition checks over a request's declared schema.

It is a **pure library**: no network access, no filesystem access, and no
wall-clock reads anywhere in the evaluation path. A verdict is a function of
the request, the active ruleset, and any reference tables supplied to it —
the only notion of "now" it accepts is an evaluation timestamp passed in by
the caller.

This module provides:

- Three-valued check outcomes (pass / fail / indeterminate), with
  indeterminate never collapsing to pass
- Check severities and classes (deterministic vs. statistical)
- Evaluation modes (permissive / strict) and aggregate verdict computation
- Exact-decimal arithmetic for monetary values (`decimal`) — nothing
  monetary passes through `float64`
- Field declarations for the value kinds checks operate on (`field`)
- Versioned ISO reference tables — ISO 4217 currencies, ISO 3166-1 alpha-2
  countries (`tables`)

## Install

```
go get github.com/tidalsoft/verdict
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/tidalsoft/verdict"
)

func main() {
	amountCheck, err := verdict.NewResult("MU-01", verdict.ClassD, verdict.SeverityBlock, verdict.OutcomePass)
	if err != nil {
		panic(err)
	}
	fraudCheck, err := verdict.NewResult("PG-05", verdict.ClassS, verdict.SeverityWarn, verdict.OutcomeFail)
	if err != nil {
		panic(err)
	}

	agg, err := verdict.ComputeAggregate([]verdict.Result{amountCheck, fraudCheck}, verdict.ModePermissive)
	if err != nil {
		panic(err)
	}

	fmt.Println(agg.Verdict) // allow_with_warnings
}
```

`Result` and `ComputeAggregate` are the vocabulary this library exposes;
specific check implementations named by rule ID (`MU-*`, `PG-*`) are
provided by code that imports this library, not by this module itself.
[Gatepost](https://tidalsoft.ai), a hosted service from
Tidalsoft, implements this library in production if you want a running
service rather than a library to build one on.

## License

Licensed under the [Apache License, Version 2.0](LICENSE). Copyright
Tidalsoft.

## Rule ID namespace and conformance

The `MU-`, `PG-`, and `PC-` rule ID prefixes are reserved to the
specification maintainer. An implementation may claim conformance with the
MU or PG rule sets only by passing the published conformance test suite.
Forks are welcome to add rules, but under their own prefix, so that
fleet-scale measurement data keyed by rule ID stays comparable across
implementations.

This README makes no benchmark or efficacy claims (e.g. false-positive
rates, precision, or "gates improve success by X%") because none of that
measurement work has been run and published for this code.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Contributions are accepted under a
Developer Certificate of Origin, not a CLA.

## Security

See [SECURITY.md](SECURITY.md) for how to report a vulnerability.
