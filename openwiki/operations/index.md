# Files

- [Configuration, deployment, and lifecycle runbook](configuration-and-deployment.md) - Configure and deploy NACP as a protected Nomad admission proxy, including TLS boundaries, file mounts, startup dependencies, resource limits, and graceful shutdown. Diagnose failures by lifecycle stage and review compatibility risks before upgrades.
- [Logs, metrics, and request tracing](observability.md) - Configure NACP's local and OTLP telemetry, follow admission and remote-call traces, and interpret controller counters without mistaking them for enforcement outcomes. Includes lifecycle, troubleshooting, and safe instrumentation changes.
