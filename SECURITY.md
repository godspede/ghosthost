# Security Policy

## Supported Versions

Only the latest released version of `ghosthost` is supported.

## Scope

`ghosthost` exists to serve local files temporarily over a trusted network
(typically a Tailscale tailnet). In scope:

- Unauthenticated access to shared files from outside the network the
  daemon is bound to.
- Local code execution or privilege escalation via the daemon.
- Path traversal, symlink-swap, or TOCTOU attacks against served files.
- HTML or header injection via filenames or display names.
- Leaking raw tokens via logs, history files, or telemetry.

Out of scope:

- Denial of service from inside the trusted network.
- Users who already hold local credentials on the host running the daemon
  and can read the lockfile.
- Vulnerabilities in Tailscale, the operating system, or other components
  ghosthost depends on.

## Reporting a Vulnerability

Please email `zackary.frank@gmail.com`. Do not open public GitHub issues
for undisclosed vulnerabilities.

Expect acknowledgement within 7 days and a fix or mitigation plan within
30 days depending on severity.
