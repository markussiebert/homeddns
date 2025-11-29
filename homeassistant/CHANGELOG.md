## 0.1.0 (2025-11-29)

### Features

- Initial release of HomeDDNS Home Assistant Add-on
- Support for Netcup CCP API provider
- Support for AWS Route53 provider
- Ingress support for easy access through Home Assistant UI
- Automatic bcrypt password hashing
- Wildcard DNS support
- Multi-architecture support (aarch64, amd64, armhf, armv7, i386)
- Compatible with all standard DynDNS clients
- Health check endpoint
- Automatic IP detection from request source

### Security

- Passwords automatically hashed with bcrypt
- Runs as non-root user
- Minimal container image based on distroless
- Read-only root filesystem
- No privilege escalation
