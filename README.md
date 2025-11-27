# HomeDDNS - Self-hosted Dynamic DNS for Your Home

A lightweight, self-hosted DynDNS (Dynamic DNS) service with pluggable DNS provider support. Currently supports **Netcup** and **AWS Route53**. Perfect for home labs, homeservers, and Home Assistant users. Deploy with Docker Compose, Kubernetes, or as a Home Assistant addon.

## Features

- **Multi-Provider Plugin System**: Pluggable DNS provider architecture
  - **Netcup CCP API**: Updates DNS records via Netcup's Customer Control Panel API
  - **AWS Route53**: Updates DNS records via AWS Route53 API
- **Session Management**: Automatic session handling (Netcup: 15-minute timeout with proactive refresh)
- **Multiple Formats**: Supports both standard DynDNS format and UniFi custom provider format
- **Wildcard DNS**: Full support for wildcard DNS records (e.g., `*.example.com`)
- **Basic Authentication**: Secured with bcrypt password hashing
- **Flexible Deployment**: Run with Docker Compose, Kubernetes, or Home Assistant addon
- **Health Checks**: Built-in health endpoint for Kubernetes probes
- **Minimal Attack Surface**: Uses distroless base image and runs as non-root
- **IAM Role Support**: AWS Route53 provider supports IRSA (IAM Roles for Service Accounts)

## Why Self-Host Dynamic DNS?

Most modern routers already support the DynDNS (inadyn) API, and homelabbers typically own one or more domains. Instead of relying on third-party DynDNS services, you can reuse an existing domain and keep full ownership of your update pipeline. Self-hosting brings:

- **Control**: You choose the auth rules, uptime strategy, and see every update request.
- **Privacy**: Your Dynamic DNS credentials and logs stay within your own network or trusted infrastructure.
- **Reliability**: No upstream DynDNS provider to throttle or sunset a free tier.
- **Integration**: Works with any standard router or device that speaks DynDNS/HTTP update protocols, so you can keep your existing-compatible clients.

## Third-Party References

Mentions of **Netcup**, **AWS**, and **Route53** are strictly descriptive: they explain which external services the project integrates with. This project is independently developed and is not affiliated with, endorsed by, or sponsored by those companies or their products.

## Architecture

```
┌─────────────┐
│   Router    │  (UniFi, Fritz!Box, etc.)
│  /inadyn    │
└──────┬──────┘
       │ HTTP GET with Basic Auth
       ▼
┌─────────────────────────────┐
│  HomeDDNS Service           │
│  (Docker/K8s/Home Assistant)│
│  ┌───────────────────────┐  │
│  │ HTTP Server           │  │
│  │ - Auth Middleware     │  │
│  │ - DynDNS Handler      │  │
│  └───────┬───────────────┘  │
│          ▼                  │
│  ┌───────────────────────┐  │
│  │ Netcup CCP API Client │  │
│  │ - Session Management  │  │
│  │ - DNS Record Updates  │  │
│  └───────┬───────────────┘  │
└──────────┼──────────────────┘
           │ HTTPS
           ▼
   ┌───────────────┐
   │  Netcup CCP   │
   │  API Endpoint │
   └───────────────┘
```

## Prerequisites

- **Deployment Platform** (choose one):
  - Docker / Docker Compose
  - Kubernetes cluster
  - Home Assistant (addon coming soon)
- **DNS Provider** (choose one):
  - **Netcup**: Account with API credentials, domain managed by Netcup
  - **Route53**: AWS account with Route53 hosted zone

## Quick Start

### Docker Compose (Recommended for Home Use)

1. **Create `docker-compose.yml`:**

```yaml
version: '3.8'
services:
  homeddns:
    image: ghcr.io/markussiebert/homeddns:latest
    container_name: homeddns
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - AUTH_USERNAME=dyndns
      - AUTH_PASSWORD_HASH=$2a$10$... # Generate with script below
      - DNS_PROVIDER=netcup_ccp
      - DOMAIN=example.com
      - DNS_TTL=60
      # Netcup credentials
      - NETCUP_CUSTOMER_NUMBER=123456
      - NETCUP_API_KEY=your-api-key
      - NETCUP_API_PASSWORD=your-api-password
```

2. **Generate password hash:**

```bash
docker run --rm ghcr.io/markussiebert/homeddns:latest \
  /usr/bin/htpasswd -bnBC 10 "" your-password | tr -d ':\n'
```

3. **Start the service:**

```bash
docker-compose up -d
```

## Deployment Options

### Option 1: Docker Compose (Above)

### Option 2: Kubernetes

### 1. Generate Password Hash

Use the provided script to generate a bcrypt hash for your DynDNS password:

```bash
../scripts/generate-password-hash.sh your-password-here
```

This will output a bcrypt hash like: `$2a$10$...`

### 2. Create Kubernetes Secret

Create a secret with your credentials:

```bash
kubectl create secret generic homeddns-secret \
  --from-literal=auth-username=dyndns \
  --from-literal=auth-password-hash='$2a$10$...' \
  --from-literal=netcup-customer-number=123456 \
  --from-literal=netcup-api-key=YOUR_API_KEY \
  --from-literal=netcup-api-password=YOUR_API_PASSWORD
```

**Netcup API Credentials:**

- Login to Netcup Customer Control Panel (CCP)
- Navigate to: Stammdaten → API
- Generate API Key and API Password
- Customer Number is your Netcup customer number

### 3. Deploy to Kubernetes

```bash
# Apply manifests
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### 4. Expose the Service

**Option A: Ingress (Recommended)**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: homeddns
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - dyndns.example.com
      secretName: dyndns-tls
  rules:
    - host: dyndns.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: homeddns
                port:
                  number: 80
```

**Option B: LoadBalancer**

```bash
kubectl patch service homeddns -p '{"spec":{"type":"LoadBalancer"}}'
```

## DNS Provider Configuration

This service uses a plugin system that supports multiple DNS providers. Choose the provider that matches your DNS hosting.

### Netcup Provider

The Netcup provider uses the CCP (Customer Control Panel) API to update DNS records.

**Environment Variables:**

- `DNS_PROVIDER=netcup_ccp` (default)
- `NETCUP_CUSTOMER_NUMBER` - Your Netcup customer number (required)
- `NETCUP_API_KEY` - API key from CCP (required)
- `NETCUP_API_PASSWORD` - API password from CCP (required)

**Example Deployment:**

```bash
ko apply -f k8s/deployment.yaml
```

**Features:**

- Automatic session management (15-min timeout, 10-min refresh)
- Support for all DNS record types (A, AAAA, CNAME, etc.)
- Wildcard DNS support

### AWS Route53 Provider

The Route53 provider uses the AWS SDK to update DNS records in Route53 hosted zones.

**Environment Variables:**

- `DNS_PROVIDER=route53`
- AWS credentials via one of:
  - **IRSA (Recommended for EKS)**: IAM Role for Service Accounts
  - **IAM Instance Profile**: Attached to EC2 instances
  - **Environment Variables**: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
  - **Shared Credentials**: `~/.aws/credentials`

**Example Deployment:**

```bash
ko apply -f k8s/deployment-route53.yaml
```

**Required IAM Permissions:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "route53:ListHostedZones",
        "route53:ListResourceRecordSets",
        "route53:ChangeResourceRecordSets"
      ],
      "Resource": "*"
    }
  ]
}
```

**IRSA Setup (EKS):**

1. Create IAM role with the above policy
2. Associate role with ServiceAccount:
   ```yaml
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: dyndns-route53
     annotations:
       eks.amazonaws.com/role-arn: arn:aws:iam::ACCOUNT_ID:role/dyndns-route53
   ```
3. Deploy using `k8s/deployment-route53.yaml`

**Features:**

- Automatic hosted zone discovery
- Hosted zone caching for performance
- Support for all DNS record types
- Wildcard DNS support
- Configurable TTL

### Adding Custom Providers

The plugin system makes it easy to add new DNS providers:

1. Implement the `provider.Provider` interface in `internal/provider/`
2. Add provider initialization in `cmd/server/main.go`
3. Update documentation

**Provider Interface:**

```go
type Provider interface {
    Name() string
    GetRecord(ctx context.Context, domain, hostname, recordType string) (*DNSRecord, error)
    UpdateRecord(ctx context.Context, domain string, record *DNSRecord) error
    Close(ctx context.Context) error
}
```

## Usage

### Standard DynDNS Format

```bash
curl -u "dyndns:your-password" \
  "https://dyndns.example.com/nic/update?hostname=home.example.com&myip=1.2.3.4"
```

Response: `good 1.2.3.4` or `nochg 1.2.3.4`

### UniFi Custom Provider Format

```bash
curl -u "dyndns:your-password" \
  "https://dyndns.example.com/home.example.com"
```

Response: `good` or `nochg`

### Automatic IP Detection

Omit the `myip` parameter to automatically use the request source IP:

```bash
curl -u "dyndns:your-password" \
  "https://dyndns.example.com/nic/update?hostname=home.example.com"
```

### Wildcard DNS

```bash
curl -u "dyndns:your-password" \
  "https://dyndns.example.com/nic/update?hostname=*.example.com&myip=1.2.3.4"
```

## Configuration

### Environment Variables

| Variable                 | Required | Default | Description               |
| ------------------------ | -------- | ------- | ------------------------- |
| `PORT`                   | No       | `8080`  | HTTP server port          |
| `AUTH_USERNAME`          | Yes      | -       | Basic auth username       |
| `AUTH_PASSWORD_HASH`     | Yes      | -       | Bcrypt hash of password   |
| `NETCUP_CUSTOMER_NUMBER` | Yes      | -       | Netcup customer number    |
| `NETCUP_API_KEY`         | Yes      | -       | Netcup API key            |
| `NETCUP_API_PASSWORD`    | Yes      | -       | Netcup API password       |
| `DNS_TTL`                | No       | `60`    | DNS record TTL in seconds |

### Response Codes

| Code      | Description                            |
| --------- | -------------------------------------- |
| `good`    | DNS record updated successfully        |
| `nochg`   | IP address unchanged, no update needed |
| `notfqdn` | Invalid hostname format                |
| `911`     | Server error or invalid IP address     |

## Router Configuration

### UniFi Dream Machine / UDM-Pro

1. Navigate to: **Settings → Internet → WAN**
2. Scroll to **Dynamic DNS**
3. Click **Create New Dynamic DNS**
4. Configure:
   - **Service**: `custom`
   - **Hostname**: `home.example.com` (or `*.example.com` for wildcard)
   - **Username**: `dyndns` (or your configured username)
   - **Password**: Your DynDNS password
   - **Server**: `dyndns.example.com/%h` (replace with your domain)

**Note**: UniFi uses inadyn internally. You need to edit `/etc/inadyn.conf` to add `ddns-path = "/"`:

```bash
# SSH into UniFi device
ssh admin@192.168.1.1

# Edit inadyn config
vi /etc/inadyn.conf

# Add this line in the custom provider section:
ddns-path = "/"

# Restart inadyn
killall inadyn
```

### Fritz!Box

1. Navigate to: **Internet → Freigaben → DynDNS**
2. Configure:
   - **DynDNS-Anbieter**: `Benutzerdefiniert`
   - **Update-URL**: `https://dyndns.example.com/nic/update?hostname=<domain>&myip=<ipaddr>`
   - **Domainname**: `home.example.com`
   - **Benutzername**: `dyndns`
   - **Kennwort**: Your DynDNS password

## Development

### Prerequisites

- [mise](https://mise.jdx.dev/) - Development environment manager (recommended)
- Go 1.25.4 (managed by mise)
- [Ko](https://ko.build/) - Container image builder (optional)

### Quick Start with mise

All common tasks are available via mise:

```bash
# List all available tasks
mise tasks

# Build binary for current platform
mise run build

# Build for Linux (deployment)
mise run build:linux

# Build container image with Ko (no Dockerfile needed!)
mise run build:ko

# Run server locally
mise run run

# Run tests
mise run test

# Clean build artifacts
mise run clean
```

### Available mise Tasks

| Task | Description |
|------|-------------|
| `build` | Build homeddns binary for current platform |
| `build:linux` | Build for Linux AMD64 |
| `build:linux-arm` | Build for Linux ARM64 (Raspberry Pi) |
| `build:ko` | Build container image with Ko (local) |
| `build:ko-push` | Build and push container to registry |
| `build:ko-multiplatform` | Build multi-platform container image |
| `run` | Run homeddns server |
| `run:update` | Update DNS record (pass hostname as arg) |
| `test` | Run all tests |
| `test:provider` | Test provider package only |
| `fmt` | Format Go code |
| `clean` | Remove build artifacts |

### Local Testing

```bash
# Copy the example environment file
cp .env.local.example .env.local

# Edit .env.local with your credentials
# Then run with mise (automatically loads .env.local)
mise run run:dev

# Or export environment variables manually
export AUTH_USERNAME=dyndns
export AUTH_PASSWORD_HASH='$2a$10$...'
export DNS_PROVIDER=netcup_ccp
export DOMAIN=example.com
export NETCUP_CUSTOMER_NUMBER=123456
export NETCUP_API_KEY=your-api-key
export NETCUP_API_PASSWORD=your-api-password

# Run server
mise run run

# Test update command
mise run run:update -- test.example.com
```

### Test with curl

```bash
# Test health endpoint (no auth required)
curl http://localhost:8080/health

# Test DNS update
curl -u "dyndns:your-password" \
  "http://localhost:8080/nic/update?hostname=test.example.com&myip=1.2.3.4"
```

### Why Ko for Container Builds?

Ko builds optimized container images directly from Go code:
- ✅ **No Dockerfile needed** - builds directly from Go source
- ✅ **Smaller images** - optimized layers, distroless base (~15MB total)
- ✅ **SBOM included** - automatic software bill of materials
- ✅ **Multi-platform** - easy cross-compilation
- ✅ **Fast builds** - aggressive optimization flags built-in

### Binary Size Optimizations

Our build is optimized for minimal size:

| Build Type | Size | Optimizations |
|------------|------|---------------|
| Standard | ~19MB | Default Go build |
| **Optimized** | **~14MB** | Aggressive optimization flags (26% smaller) |

**Optimization flags explained:**
- `-trimpath` - Remove file system paths from binary
- `-s -w` - Strip debug info and symbol table
- `-extldflags=-static` - Create fully static binary (no external dependencies)
- `-tags netgo` - Use pure Go networking (no CGO required)

Result: **14MB static binary** that runs anywhere with no dependencies!

## Netcup API Reference

- **Endpoint**: `https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON`
- **Documentation**: https://www.netcup-wiki.de/wiki/CCP_API
- **Session Timeout**: 15 minutes
- **Supported Operations**:
  - `login` - Authenticate and obtain session ID
  - `logout` - Invalidate session
  - `infoDnsRecords` - Retrieve DNS records for a domain
  - `updateDnsRecords` - Update DNS records

## Security Considerations

- All passwords are hashed using bcrypt (cost factor 10)
- Container runs as non-root user (UID 65532)
- Read-only root filesystem
- No privilege escalation
- Minimal container image (distroless)
- All secrets stored in Kubernetes Secrets

## Troubleshooting

### Check Logs

**Docker Compose:**
```bash
docker logs -f homeddns
```

**Kubernetes:**
```bash
kubectl logs -l app=homeddns --tail=100 -f
```

### Common Issues

**401 Unauthorized**

- Check that Basic Auth credentials are correct
- Verify password hash was generated correctly

**"notfqdn" Response**

- Hostname must be a valid FQDN
- Ensure domain is managed by Netcup

**"911" Response**

- Check Netcup API credentials
- Verify domain exists in Netcup account
- Check service logs for detailed error messages

**Session Timeout**

- Sessions are automatically refreshed every 10 minutes
- If you see frequent re-logins, check network connectivity to Netcup API

## License

This project inherits the license from the parent repository.

## Contributing

Contributions are welcome! Please ensure:

- Code follows Go best practices
- Tests are added for new features
- Documentation is updated
