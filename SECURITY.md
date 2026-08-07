# Security Policy

This engine and rule catalogue are published under Apache 2.0, so
vulnerability reports will arrive whether or not there is a process for
them. This document is that process.

## Reporting a vulnerability

Email **security@tidalsoft.ai** with a description of the issue, the
affected version or commit, and reproduction steps or a proof of concept.
Do not open a public GitHub issue for a suspected vulnerability.

We do not currently publish a PGP key for this address; if you need
encrypted transport for a sensitive report, say so in your initial email and
we will arrange a channel.

## Scope

In scope:

- This repository (`github.com/tidalsoft/verdict`): the evaluation engine
  and rule catalogue
- Gatepost's hosted service, which imports this module

Out of scope:

- Denial of service
- Social engineering (of maintainers, staff, or users)
- Findings against third-party providers or dependencies not maintained in
  this repository (report those upstream)

## What to expect

- **Acknowledgement within 3 business days** of your report.
- **Assessment within 10 business days**: we will tell you whether we
  consider it a valid vulnerability, its severity, and our intended next
  steps.
- **Coordinated disclosure**, with a default embargo of **90 days** from
  acknowledgement, negotiable in either direction depending on complexity
  and severity. Because this repository has no supported self-host
  distribution, a fix ships as a release with a changelog entry — that
  changelog is the
  only channel that reaches anyone running this code outside the hosted
  service, so we do not disclose before a fix is available to install.

## Safe harbour

We will not pursue legal action against good-faith security research
conducted within the scope above, that avoids privacy violations, data
destruction, and service disruption, and that gives us the reporting window
described above before any public disclosure.

## Bug bounty

There is no bug bounty program. We are glad to credit reporters (by name or
pseudonym, at your preference) in the release notes for a fix, but there is
no monetary reward.
