#!/usr/bin/env python3
"""Shared environment and workspace handling for the Unix shell entry points."""
import argparse
import json
import os
from pathlib import Path
import platform
import shlex
import shutil
import subprocess
import sys
import tempfile

REPO = Path(__file__).resolve().parent.parent
DEFAULT_ROOT = REPO / '.native/highs'


def require_tool(name):
    found = shutil.which(name)
    if not found:
        raise ValueError('Missing {}. See README.md prerequisites.'.format(name))
    return found


def native_environment(root):
    system = platform.system()
    if system not in ('Linux', 'Darwin'):
        raise ValueError('Only Linux and macOS are supported.')
    root = root.resolve()
    if any(c in str(root) for c in ':,\n\r'):
        raise ValueError('HiGHS paths cannot contain colons, commas or newlines.')
    if not (root / 'include/highs/interfaces/highs_c_api.h').is_file():
        raise ValueError('HiGHS headers not found. Run scripts/build-highs.sh first.')
    # Accept lib64 installations as well as the local build's lib directory.
    lib = next((root / name for name in ('lib', 'lib64')
                if (root / name / 'pkgconfig/highs.pc').is_file()), None)
    if lib is None:
        raise ValueError('HiGHS highs.pc not found under lib/pkgconfig or lib64/pkgconfig.')
    library = lib / ('libhighs.dylib' if system == 'Darwin' else 'libhighs.so')
    if not library.is_file():
        raise ValueError('HiGHS shared library not found: {}'.format(library))
    pc_dir = str(lib / 'pkgconfig')
    search = [p for p in os.environ.get('PKG_CONFIG_PATH', '').split(':') if p and p != pc_dir]
    env = {
        'CGO_ENABLED': '1',
        'CC': require_tool(os.environ.get('CC', 'cc')),
        'CXX': require_tool(os.environ.get('CXX', 'c++')),
        'PKG_CONFIG': require_tool(os.environ.get('PKG_CONFIG', 'pkg-config')),
        'PKG_CONFIG_PATH': ':'.join([pc_dir] + search),
    }
    # Embed a search path in binaries so macOS does not depend on DYLD_* variables
    # (which protected system processes may strip). Also works on Linux.
    rpath = '-Wl,-rpath,' + str(lib)
    # Go's flag parser accepts one pair of quotes, without shell unescaping.
    if '"' not in rpath:
        quoted_rpath = '"' + rpath + '"'
    elif "'" not in rpath:
        quoted_rpath = "'" + rpath + "'"
    else:
        raise ValueError('HiGHS paths cannot contain both single and double quotes.')
    flags = os.environ.get('CGO_LDFLAGS', '').rstrip()
    if not flags.endswith(quoted_rpath):
        flags = (flags + ' ' + quoted_rpath).lstrip()
    env['CGO_LDFLAGS'] = flags
    full_env = dict(os.environ, **env)
    version = subprocess.check_output([env['PKG_CONFIG'], '--modversion', 'highs'],
                                      env=full_env, text=True).strip()
    if version != '1.15.1':
        raise ValueError('Expected HiGHS 1.15.1, found {}.'.format(version))
    cflags = shlex.split(subprocess.check_output(
        [env['PKG_CONFIG'], '--cflags', 'highs'], env=full_env, text=True))
    subprocess.run([env['CC'], '-x', 'c', '-fsyntax-only', '-'] + cflags,
                   input='#include "interfaces/highs_c_api.h"\n'
                         '_Static_assert(sizeof(HighsInt) == 4, "32-bit HighsInt required");\n',
                   text=True, env=full_env, check=True)
    return env


def read_workspace(path):
    if not path.exists():
        return {'folders': [{'path': '.'}], 'settings': {}}
    try:
        config = json.loads(path.read_text(encoding='utf-8-sig'))
    except ValueError as exc:
        raise ValueError('Existing workspace must be valid JSON (no comments or trailing commas); '
                         'it has not been changed.') from exc
    if not isinstance(config, dict):
        raise ValueError('Workspace must be a JSON object.')
    if not isinstance(config.get('settings', {}), dict):
        raise ValueError('Workspace settings must be a JSON object.')
    return config


def setup(args):
    project = args.project_path.resolve()
    if not project.is_dir():
        raise ValueError('Project directory does not exist: {}'.format(project))
    workspace = project / 'highs-go.code-workspace'
    config = read_workspace(workspace)
    settings = config.setdefault('settings', {})
    keys = ('go.toolsEnvVars', 'terminal.integrated.env.' +
            ('osx' if platform.system() == 'Darwin' else 'linux'))
    for key in keys:
        if not isinstance(settings.get(key, {}), dict):
            raise ValueError('{} must be a JSON object.'.format(key))
    require_tool('go')
    for name in ('CC', 'CXX', 'PKG_CONFIG'):
        require_tool(os.environ.get(name, {'CC': 'cc', 'CXX': 'c++', 'PKG_CONFIG': 'pkg-config'}[name]))
    root = args.highs_root.resolve()
    if root == DEFAULT_ROOT.resolve() and not (root / 'lib/pkgconfig/highs.pc').is_file():
        subprocess.run(['bash', str(REPO / 'scripts/build-highs.sh'), '--jobs', str(args.jobs)], check=True)
    env = native_environment(root)
    full_env = dict(os.environ, **env)
    # Check ABI/version compatibility and an actual solve before writing the workspace.
    subprocess.run(['go', 'test', '-count=1', '-run', '^TestStatisticsAndVersion$', '.'],
                   cwd=REPO, env=dict(full_env, HIGHS_EXPECT_VERSION='1.15.1'), check=True)
    subprocess.run(['go', 'run', './cmd/example'], cwd=REPO, env=full_env, check=True)
    for key in keys:
        settings.setdefault(key, {}).update(env)
    # Atomic replacement keeps an existing workspace intact on write failures.
    name = None
    try:
        with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', dir=project,
                                         prefix='.highs-workspace-', delete=False) as stream:
            name = stream.name
            json.dump(config, stream, indent=2)
            stream.write('\n')
        os.replace(name, workspace)
    finally:
        if name and os.path.exists(name):
            os.unlink(name)
    print('Open {} in VS Code. New terminals and Go tools are configured automatically.'.format(workspace))


def positive_int(value):
    number = int(value)
    if number < 1:
        raise argparse.ArgumentTypeError('must be a positive integer')
    return number


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=('setup', 'env'))
    parser.add_argument('--highs-root', type=Path, default=DEFAULT_ROOT,
                        help='existing HiGHS 1.15.1 installation (default: checkout/.native/highs)')
    parser.add_argument('--project-path', type=Path, default=Path.cwd(),
                        help='workspace directory (setup only; default: current directory)')
    parser.add_argument('--jobs', type=positive_int, default=4, help='native build parallelism (default: 4)')
    args = parser.parse_args()
    try:
        if platform.system() not in ('Linux', 'Darwin'):
            raise ValueError('Only Linux and macOS are supported.')
        if args.action == 'setup':
            setup(args)
        else:
            for key, value in native_environment(args.highs_root).items():
                print('export {}={}'.format(key, shlex.quote(value)))
    except (OSError, ValueError, subprocess.CalledProcessError) as exc:
        print('HiGHS setup: {}'.format(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
