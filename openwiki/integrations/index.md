# Files

- [Notation image verification and policy enforcement](image-verification.md) - How NACP verifies registry image signatures using Notation trust policies, X.509 trust stores, and ORAS credentials, and how direct warning-only validation differs from OPA policy enforcement.
- [Authoring and running embedded and bundle OPA policies](opa-policies.md) - Choose NACP's embedded Rego queries or shared OPA SDK bundles, author validation messages and ordered JSON patches, and verify the exact decision contracts and startup behavior.
- [Webhook policy services and transport contracts](webhooks.md) - Request and response contracts for external admission validators and JSON-Patch mutators, including shared payloads, transport limits, sanitized identity context, and adapter failure semantics.
