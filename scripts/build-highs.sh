#!/usr/bin/env bash
# Build the same pinned, 32-bit HighsInt shared library as the Windows helper.
set -euo pipefail
jobs=4
while (($#)); do
    case "$1" in
        --jobs) [[ $# -ge 2 ]] || { echo 'Missing --jobs value' >&2; exit 1; }; jobs=$2; shift 2 ;;
        -h|--help) echo 'Usage: build-highs.sh [--jobs N]'; exit 0 ;;
        *) echo "Unknown argument: $1" >&2; exit 1 ;;
    esac
done
[[ $jobs =~ ^[1-9][0-9]*$ ]] || { echo '--jobs must be a positive integer' >&2; exit 1; }
case "$(uname -s)" in Linux|Darwin) ;; *) echo 'Only Linux and macOS are supported.' >&2; exit 1 ;; esac
for tool in git cmake make python3 "${CC:-cc}" "${CXX:-c++}"; do
    command -v "$tool" >/dev/null || { echo "Missing $tool; see README.md prerequisites." >&2; exit 1; }
done
repo=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
native_root="$repo/.native"
source_dir="$native_root/src"
build_dir="$native_root/build"
install_dir="$native_root/highs"
commit=04024d701f79feb8e2f18bc3df0dffc04ef05088
mkdir -p "$native_root"
if [[ ! -e $source_dir ]]; then
    git clone --depth 1 --branch v1.15.1 https://github.com/ERGO-Code/HiGHS.git "$source_dir"
fi
actual=$(git -C "$source_dir" rev-parse HEAD)
[[ $actual == "$commit" ]] || { echo 'HiGHS source does not match the pinned release.' >&2; exit 1; }
dirty=$(git -C "$source_dir" status --porcelain)
[[ -z $dirty ]] || { echo 'HiGHS source has local changes; use a clean pinned checkout.' >&2; exit 1; }
cmake -S "$source_dir" -B "$build_dir" -G 'Unix Makefiles' \
    "-DCMAKE_C_COMPILER=$(command -v "${CC:-cc}")" \
    "-DCMAKE_CXX_COMPILER=$(command -v "${CXX:-c++}")" \
    -DCMAKE_BUILD_TYPE=Release "-DCMAKE_INSTALL_PREFIX=$install_dir" \
    -DCMAKE_INSTALL_LIBDIR=lib -DFAST_BUILD=ON -DBUILD_SHARED_LIBS=ON \
    -DBUILD_SHARED_EXTRAS_LIB=OFF -DBUILD_CXX_EXE=OFF -DBUILD_EXAMPLES=OFF \
    -DBUILD_TESTING=OFF -DZLIB=OFF -DHIPO=OFF -DHIGHSINT64=OFF
cmake --build "$build_dir" --parallel "$jobs"
cmake --install "$build_dir"
# Upstream's .pc template leaves include/library paths unquoted.
python3 - "$install_dir/lib/pkgconfig/highs.pc" <<'PYTHON'
from pathlib import Path
import sys
pc = Path(sys.argv[1])
text = pc.read_text().replace('-L${libdir}', '-L"${libdir}"')
text = text.replace('-I${includedir}', '-I"${includedir}"')
pc.write_text(text)
PYTHON
mkdir -p "$install_dir/share/doc/HIGHS"
cp "$source_dir/LICENSE.txt" "$source_dir/THIRD_PARTY_NOTICES.md" "$install_dir/share/doc/HIGHS/"
echo "HiGHS installed in $install_dir"
