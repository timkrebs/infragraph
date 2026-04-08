# Security Policy

## Reporting a Vulnerability

The InfraGraph team takes security seriously. If you believe you have found a
security vulnerability in InfraGraph, please report it responsibly.

**Please do NOT file a public GitHub issue for security vulnerabilities.**

Instead, please send an email to:

**security@timkrebs.dev**

Include the following in your report:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact
- Suggested fix (if any)

## Response Timeline

- **Acknowledgement**: Within 48 hours of receiving your report.
- **Assessment**: We will assess the severity and impact within 5 business days.
- **Fix**: Critical vulnerabilities will be patched as soon as possible. We aim
  to release a fix within 30 days for non-critical issues.
- **Disclosure**: We will coordinate with you on disclosure timing. We ask that
  you give us reasonable time to address the issue before public disclosure.

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | Yes       |
| < latest | No       |

## Scope

The following are in scope:

- The InfraGraph server and CLI binary
- The REST API endpoints
- The bbolt storage layer
- Collector plugins shipped with the project

The following are out of scope:

- Third-party collector plugins not maintained in this repository
- Issues in dependencies (please report those upstream)

Thank you for helping keep InfraGraph and its users safe.
