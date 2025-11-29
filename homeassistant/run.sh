#!/usr/bin/with-contenv bash
set -euo pipefail

export ADDON_OPTIONS_PATH="${ADDON_OPTIONS_PATH:-/data/options.json}"
exec /usr/local/bin/homeddns server
