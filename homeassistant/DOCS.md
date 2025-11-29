# HomeDDNS Add-on Documentation

## What is HomeDDNS?

HomeDDNS is a self-hosted Dynamic DNS service that automatically updates your DNS records when your IP address changes. It's perfect for home users who want to access their Home Assistant instance remotely without relying on third-party DynDNS services.

## Why Use This Add-on?

- **Privacy**: Your DynDNS credentials and update logs stay in your network
- **Control**: You own the domain and update pipeline
- **Reliability**: No third-party service to depend on
- **Integration**: Works with any router that supports DynDNS/inadyn protocol
- **Cost-Effective**: Use your existing domain instead of paying for DynDNS services

## Quick Start

### 1. Choose Your DNS Provider

This add-on supports two DNS providers:

**Netcup** (German hosting provider)
- Popular in Europe
- Affordable domains
- Simple API

**AWS Route53** (Amazon's DNS service)
- Global availability
- Enterprise-grade reliability
- Pay-as-you-go pricing

### 2. Get API Credentials

#### For Netcup:
1. Log in to [Netcup CCP](https://www.customercontrolpanel.de/)
2. Go to: Stammdaten → API
3. Click "API-Key erstellen"
4. Save your API Key and API Password
5. Find your Customer Number in the account overview

#### For AWS Route53:
1. Create an IAM user in AWS Console
2. Attach a policy with these permissions:
   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Action": [
         "route53:ListHostedZones",
         "route53:ListResourceRecordSets",
         "route53:ChangeResourceRecordSets"
       ],
       "Resource": "*"
     }]
   }
   ```
3. Generate access keys for the IAM user
4. Save the Access Key ID and Secret Access Key

### 3. Generate Password Hash

Before configuring, generate a bcrypt hash of your password:

```bash
docker run --rm -i ghcr.io/markussiebert/homeddns:latest hash-password
# Type your password and press Enter
# Copy the output hash
```

### 4. Configure the Add-on

Example configuration for Netcup:

```yaml
auth_username: "dyndns"
auth_password_hash: "$2a$10$VpADQ4ns1gr1LbHZr/2/f.LdrKT8chhHUJVoMyjOv1A3Y5msQQJVi"
dns_provider: "netcup_ccp"
domain: "example.com"
dns_ttl: 60
netcup_customer_number: "123456"
netcup_api_key: "abc123def456"
netcup_api_password: "xyz789uvw012"
```

Example configuration for AWS Route53:

```yaml
auth_username: "dyndns"
auth_password_hash: "$2a$10$VpADQ4ns1gr1LbHZr/2/f.LdrKT8chhHUJVoMyjOv1A3Y5msQQJVi"
dns_provider: "route53"
domain: "example.com"
dns_ttl: 60
aws_access_key_id: "AKIAIOSFODNN7EXAMPLE"
aws_secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
aws_region: "us-east-1"
```

### 5. Configure Your Router

Most modern routers support DynDNS. Here's how to set it up:

**Generic DynDNS Client:**
- Server: `homeassistant.local:8080`
- Path: `/nic/update?hostname=%h&myip=%i`
- Username: Your `auth_username` from config
- Password: Your `auth_password` from config

**UniFi Devices:**
- Service: `custom`
- Server: `homeassistant.local:8080/%h`

**Fritz!Box:**
- Update-URL: `http://homeassistant.local:8080/nic/update?hostname=<domain>&myip=<ipaddr>`

## Advanced Features

### Wildcard DNS

Update all subdomains at once:

```bash
curl -u "dyndns:password" \
  "http://homeassistant.local:8080/nic/update?hostname=*.example.com"
```

This will update `*.example.com` to point to your current IP, so `anything.example.com` will resolve correctly.

### Multiple Hostnames

You can configure multiple DynDNS entries in your router, each updating different hostnames:
- `home.example.com` for your main domain
- `*.example.com` for wildcard
- `vpn.example.com` for VPN access

### Using with Nginx Proxy Manager

If you're using Nginx Proxy Manager or another reverse proxy:

1. Set up HomeDDNS to update your domain
2. Configure your reverse proxy to forward requests
3. Use Let's Encrypt to get SSL certificates
4. Access Home Assistant securely via `https://home.example.com`

## Troubleshooting

### Common Issues

**"911" Error Response**
- Check your DNS provider credentials
- Verify the domain exists in your DNS provider account
- Check the add-on logs for detailed error messages

**"notfqdn" Response**
- Hostname must be a fully qualified domain name
- Examples: `home.example.com`, `*.example.com`
- Not valid: `home`, `example`

**401 Unauthorized**
- Check username and password match your configuration
- Verify your router is sending Basic Auth headers

**No IP Update**
- Check if your router is sending the correct IP
- Look for "nochg" responses (IP hasn't changed)
- Review add-on logs for update attempts

### Checking Logs

1. Go to the add-on page in Home Assistant
2. Click "Log" tab
3. Look for:
   - Successful updates: `DNS record updated`
   - No changes: `IP address unchanged`
   - Errors: Any lines with `ERROR`

### Testing Manually

Test the add-on with curl:

```bash
# From another machine on your network
curl -u "dyndns:password" \
  "http://homeassistant.local:8080/nic/update?hostname=test.example.com&myip=1.2.3.4"
```

Expected responses:
- `good 1.2.3.4` - Update successful
- `nochg 1.2.3.4` - IP hasn't changed

## Security Considerations

- **Password Storage**: Passwords are automatically hashed with bcrypt
- **Network Access**: Consider using Ingress instead of exposing the port
- **HTTPS**: Use a reverse proxy for SSL/TLS encryption
- **Credentials**: Never commit credentials to version control

## Performance

- **Memory Usage**: ~15MB RAM
- **CPU Usage**: Minimal (only active during updates)
- **Network**: Low bandwidth (only small DNS API calls)
- **Update Frequency**: Typically every 5-30 minutes (configured in router)

## Support

- **GitHub Issues**: https://github.com/markussiebert/homeddns/issues
- **Documentation**: https://github.com/markussiebert/homeddns
- **Home Assistant Community**: https://community.home-assistant.io/
