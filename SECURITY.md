# Security policy

## Reporting a vulnerability

Please do not open a public issue for a suspected vulnerability. Use GitHub's private vulnerability reporting for this repository, or contact the repository owner through the security contact listed on their GitHub profile.

Include the affected version or commit, deployment assumptions, reproduction steps, impact, and any suggested mitigation. Avoid including live credentials, Nomad ACL tokens, signing keys, or production job specifications.

## Supported versions

Security fixes are applied to the latest release and the `main` branch. Older releases may require upgrading to receive a fix.

## Deployment boundary

NACP is an admission proxy and should be treated as part of the Nomad control plane:

- Restrict network access and enable listener and upstream TLS in production.
- Sanitize `X-Forwarded-For` at the trusted edge before forwarding traffic to NACP.
- Treat configured webhooks and OPA bundle services as trusted policy infrastructure.
- Keep Notation credentials, TLS keys, OPA credentials, and Nomad tokens outside source control.
- Do not enable `insecure_skip_verify` or `repo_plain_http` outside controlled development environments.

NACP sanitizes resolved Nomad ACL token context and does not send `SecretID` to policies or webhooks.
