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

## Dashboard Exposure

The monitor endpoint exposes service identity, build metadata, process, runtime, host, network, container, and HTTP traffic statistics. Treat it as an operational endpoint rather than a public application page.

- Use `Config.Authorize` or an upstream authenticated proxy in production.
- Restrict the endpoint at the network layer where practical and serve it only over TLS.
- Keep bearer tokens and other credentials out of source control, URLs, and logs.
- Prefer a same-origin `FaviconURL`; remote favicons disclose dashboard visits to another host.

`Authorize` runs only for the configured monitor path and before HTTP method handling. A denied request receives `401 Unauthorized` for HTML, JSON, `HEAD`, and unsupported methods. The package does not implement TLS, token storage, rate limiting, or role management.

JSON build metadata is limited to the Go version, module, module version, VCS revision, and modified state. It intentionally excludes the executable path, working directory, environment variables, and remote repository URL. Collection failures expose stable identifiers instead of underlying operating-system error text.
