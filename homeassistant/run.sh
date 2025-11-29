#!/usr/bin/with-contenv bashio

bashio::log.info "Starting HomeDDNS..."

# Read configuration
AUTH_USERNAME=$(bashio::config 'auth_username')
AUTH_PASSWORD_HASH=$(bashio::config 'auth_password_hash')
DNS_PROVIDER=$(bashio::config 'dns_provider')
DOMAIN=$(bashio::config 'domain')
DNS_TTL=$(bashio::config 'dns_ttl')
PORT=$(bashio::config 'port')

# Validate required fields
if [ -z "$AUTH_PASSWORD_HASH" ]; then
    bashio::exit.nok "auth_password_hash is required. Generate it using: docker run --rm ghcr.io/markussiebert/homeddns:latest hash-password"
fi

if [ -z "$DOMAIN" ]; then
    bashio::exit.nok "domain is required"
fi

# Set common environment variables
export AUTH_USERNAME
export AUTH_PASSWORD_HASH
export DNS_PROVIDER
export DOMAIN
export DNS_TTL
export PORT

bashio::log.info "DNS Provider: ${DNS_PROVIDER}"
bashio::log.info "Domain: ${DOMAIN}"
bashio::log.info "TTL: ${DNS_TTL}"

# Provider-specific configuration
if [ "$DNS_PROVIDER" = "netcup_ccp" ]; then
    NETCUP_CUSTOMER_NUMBER=$(bashio::config 'netcup_customer_number')
    NETCUP_API_KEY=$(bashio::config 'netcup_api_key')
    NETCUP_API_PASSWORD=$(bashio::config 'netcup_api_password')
    
    if [ -z "$NETCUP_CUSTOMER_NUMBER" ] || [ -z "$NETCUP_API_KEY" ] || [ -z "$NETCUP_API_PASSWORD" ]; then
        bashio::exit.nok "Netcup credentials are required when using netcup_ccp provider"
    fi
    
    export NETCUP_CUSTOMER_NUMBER
    export NETCUP_API_KEY
    export NETCUP_API_PASSWORD
    
    bashio::log.info "Netcup Customer Number: ${NETCUP_CUSTOMER_NUMBER}"

elif [ "$DNS_PROVIDER" = "route53" ]; then
    # AWS credentials are optional if using IAM roles
    AWS_ACCESS_KEY_ID=$(bashio::config 'aws_access_key_id')
    AWS_SECRET_ACCESS_KEY=$(bashio::config 'aws_secret_access_key')
    AWS_REGION=$(bashio::config 'aws_region')
    
    if [ -n "$AWS_ACCESS_KEY_ID" ] && [ -n "$AWS_SECRET_ACCESS_KEY" ]; then
        export AWS_ACCESS_KEY_ID
        export AWS_SECRET_ACCESS_KEY
        bashio::log.info "Using AWS credentials from configuration"
    else
        bashio::log.info "Using AWS IAM role credentials (if available)"
    fi
    
    export AWS_REGION
    bashio::log.info "AWS Region: ${AWS_REGION}"
fi

bashio::log.info "Starting HomeDDNS server on port ${PORT}..."

# Run the server
exec /usr/local/bin/homeddns server
