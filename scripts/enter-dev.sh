#!/usr/bin/env bash
# Source from Bash or Zsh to configure only the current terminal.
_highs_enter_dev() {
    local script_path script_dir exports
    if [ -n "${BASH_VERSION:-}" ]; then
        script_path=${BASH_SOURCE[0]}
    elif [ -n "${ZSH_VERSION:-}" ]; then
        eval 'script_path=${(%):-%x}'
    else
        echo 'Source enter-dev.sh from Bash or Zsh.' >&2
        return 1
    fi
    script_dir=$(cd -- "$(dirname -- "$script_path")" && pwd -P) || return
    command -v python3 >/dev/null || { echo 'Python 3 is required; see README.md.' >&2; return 1; }
    if [ "${1:-}" = '--help' ] || [ "${1:-}" = '-h' ]; then
        python3 "$script_dir/unix-dev.py" env --help
        return
    fi
    exports=$(python3 "$script_dir/unix-dev.py" env "$@") || return
    eval "$exports"
}
# No set -e/-u here: sourcing must preserve the caller's shell options.
if _highs_enter_dev "$@"; then
    unset -f _highs_enter_dev
else
    unset -f _highs_enter_dev
    return 1 2>/dev/null || exit 1
fi
