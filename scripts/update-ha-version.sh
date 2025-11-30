#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <version>" >&2
  exit 1
fi

version="$1"
config_file="$(git rev-parse --show-toplevel)/homeassistant/config.yaml"

python - "${version}" "${config_file}" <<'PY'
import pathlib
import re
import sys

if len(sys.argv) != 3:
    raise SystemExit("usage: python script version file")

version = sys.argv[1]
path = pathlib.Path(sys.argv[2])
text = path.read_text()
pattern = r'^(version:\s*").*"'
replacement = r"\1" + version + r"\""
new_text, count = re.subn(pattern, replacement, text, flags=re.MULTILINE)
if count == 0:
    raise SystemExit(f"Failed to update version in {path}")
path.write_text(new_text)
PY
