# Changelog

## [Unreleased]

### Breaking Changes
- **Controller Signature Refactor**  
  The `Job`-only signature in the admission controller has been replaced with a new `types.Payload` struct.  
  - All mutators and validators now receive a `Payload` object containing both the `Job` definition and additional context (e.g., client IP, resolved token details).  
  - Any custom integrations using the old `Job`-based method signatures must be updated to use `types.Payload`.

- **OPA Input Changes**  
  The embedded OPA validator has been updated to accept a new input structure containing job and caller context.  
  - Policies and data references relying on the previous input format must be updated accordingly.

- **Remote Webhook Contract Change**  
  Webhook mutators and validators now receive a request body with the combined job and context data instead of job-only information.  
  - Downstream services expecting the old JSON schema must be updated to parse the new `Payload` format.

### Added
- **Token Resolution & Context Passing**  
  Hooks can now resolve Nomad tokens (with optional policy extraction) and pass the accessor ID, client IP, and other metadata through mutators and validators.  
  - New configuration flag `resolveToken` enables token resolution for specific hooks to avoid unnecessary overhead when not required.
  - Enhanced support for use cases like CIDR-based validation, custom ACL logic, and extended audit logging.

- **Changelog Initialization**  
  Introduced a `CHANGELOG.md` to track significant updates, especially breaking changes and added features.

### Rational 

With these changes, you can now:
- Perform CIDR-based validations by leveraging the client IP.
- Create advanced ACL logic by passing resolved ACL token details (accessor ID, policies) to OPA or remote webhooks.
- Implement more granular auditing or custom workflows by integrating the new, richer `context` data available in each request.
