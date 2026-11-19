# Security Policy

Ayumi Link can run commands on the machine it is installed on (through the Claude CLI), so security
reports are taken seriously.

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

**Please do not open a public issue.** Use the repository's private channel instead:
*Security → Report a vulnerability* (GitHub private vulnerability reporting), or contact the maintainers
privately. Include:

- what is affected and the impact,
- steps to reproduce (a minimal proof of concept is ideal),
- the version and how you run it.

You can expect an acknowledgement within a few days. Please give us reasonable time to ship a fix before
disclosing details publicly.

## In scope

Authentication/2FA bypass, session handling, CSRF/origin-check bypass, IP-allowlist bypass, path
traversal or symlink escape in the folder browser, attachment type-check bypass (uploading a disallowed file type), zip bombs or path tricks in uploaded documents, serving an upload in a way that executes it, command or argument injection into the Claude
invocation, XSS or CSP bypass, information disclosure.

## Out of scope

- Anything that requires an already-authenticated owner to do what Claude can do on their own machine.
- Exposing the app over plain HTTP or to the open internet without the hardening described in the README.
- Reading `auth.json` / the data directory while already having filesystem access as the same user
  (it is stored with mode `0600` and is not encrypted at rest).
- Denial of service by an authenticated owner.

## Hardening checklist for operators

HTTPS in front of any remote access · `AYUMI_COOKIE_SECURE=true` · `AYUMI_ALLOWED_IPS` or a private
network (Tailscale/WireGuard) · `AYUMI_TRUSTED_PROXIES` + a proxy-*set* client-IP header · recovery
codes stored offline · keep the image/CLI updated · restrict `AYUMI_ROOTS` to the folders you really need.
