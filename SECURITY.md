# Security Policy

## Supported Versions

Security fixes are provided for the latest released version and the current main development branch.

## Reporting a Vulnerability

Please do not report security vulnerabilities in public issues.

Use GitHub's private vulnerability reporting for this repository if it is available. If private reporting is not available, open a minimal public issue asking for a private contact path and do not include exploit details, logs, tokens, or sensitive environment information.

When reporting, please include:

- affected version or commit
- a concise description of the issue
- reproduction steps or proof of concept
- expected impact
- any known mitigations

## Response

This is a small open-source project without a formal security response SLA. Maintainers will make a best effort to acknowledge valid reports, investigate them, and publish a fix or mitigation when appropriate.

## Scope

`monitor` is a lightweight `net/http` middleware. Security reports are most useful when they affect:

- exposure of sensitive runtime data
- unsafe HTTP behavior in the monitor endpoint
- request handling bugs that can affect the wrapped service
- dependency vulnerabilities with a practical impact on this package
