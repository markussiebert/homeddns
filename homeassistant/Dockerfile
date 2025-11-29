ARG BUILD_FROM
FROM ${BUILD_FROM}

# Install bashio for easier config parsing
RUN apk add --no-cache \
    bash \
    jq

# Copy the pre-built multi-arch binary from our Ko-built image
# This leverages our Ko build pipeline (distroless, optimized, multi-platform)
# The appropriate architecture is automatically selected by Docker
COPY --from=ghcr.io/markussiebert/homeddns:latest /ko-app/homeddns /usr/local/bin/homeddns

# Copy run script
COPY run.sh /
RUN chmod a+x /run.sh

# Home Assistant add-on labels
LABEL \
    io.hass.name="HomeDDNS" \
    io.hass.description="Self-hosted Dynamic DNS service" \
    io.hass.type="addon" \
    io.hass.version="0.1.0"

CMD [ "/run.sh" ]
