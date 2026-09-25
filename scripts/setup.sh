#!/usr/bin/env bash
# Run from a consuming project to create its VS Code workspace.
set -euo pipefail
command -v python3 >/dev/null || { echo 'Python 3 is required; see README.md.' >&2; exit 1; }
script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
exec python3 "$script_dir/unix-dev.py" setup "$@"
