#!/usr/bin/env python3
"""Exercise Unix setup from a separate Go module, including workspace reuse."""
import json
import os
from pathlib import Path
import platform
import shutil
import subprocess
import tempfile

repo = Path(__file__).resolve().parent.parent
setup = repo / 'scripts/setup.sh'
assert setup.is_file(), 'Unix workspace setup script is missing'

with tempfile.TemporaryDirectory(prefix='highs consumer ') as scratch:
    project = Path(scratch)
    (project / 'go.mod').write_text(
        'module consumer\n\ngo 1.26\n\n'
        'require github.com/VladBagat/highs-go v0.0.0\n'
        'replace github.com/VladBagat/highs-go => ' + json.dumps(str(repo)) + '\n'
    )
    shutil.copyfile(repo / 'cmd/example/main.go', project / 'main.go')
    workspace = project / 'highs-go.code-workspace'
    subprocess.run(['bash', str(setup)], cwd=project, check=True)
    config = json.loads(workspace.read_text())
    config['settings']['editor.fontSize'] = 17
    config['settings']['go.toolsEnvVars']['EXTRA_TEST_VARIABLE'] = 'preserved'
    config['folders'].append({'path': '../another-folder'})
    workspace.write_text(json.dumps(config))
    subprocess.run(['bash', str(setup), '--project-path', str(project)], check=True)
    config = json.loads(workspace.read_text())
    assert config['settings']['editor.fontSize'] == 17
    assert config['folders'][-1]['path'] == '../another-folder'
    assert config['settings']['go.toolsEnvVars']['EXTRA_TEST_VARIABLE'] == 'preserved'
    terminal_key = 'terminal.integrated.env.' + ('osx' if platform.system() == 'Darwin' else 'linux')
    for key in ['go.toolsEnvVars', terminal_key]:
        env = dict(os.environ, **config['settings'][key])
        subprocess.run(['go', 'build', '-o', 'app', '.'], cwd=project, env=env, check=True)
        # A linked application must find HiGHS even outside the configured terminal.
        clean_env = {k: v for k, v in os.environ.items() if k not in ('LD_LIBRARY_PATH', 'DYLD_LIBRARY_PATH')}
        output = subprocess.check_output([str(project / 'app')], env=clean_env, text=True)
        assert 'objective=2 x=4 y=2' in output, output
    # Failed setup must not destroy an existing workspace.
    original = workspace.read_bytes()
    result = subprocess.run(['bash', str(setup), '--project-path', str(project),
                             '--highs-root', str(project / 'missing')])
    assert result.returncode != 0
    assert workspace.read_bytes() == original
    workspace.write_text('{ // user comment\n}')
    original = workspace.read_bytes()
    result = subprocess.run(['bash', str(setup), '--project-path', str(project)])
    assert result.returncode != 0
    assert workspace.read_bytes() == original
    workspace.write_text(json.dumps(config))
    # Relocated native installations must also work with spaces in their path.
    native_copy = project / "native libraries O'Brien"
    shutil.copytree(repo / '.native/highs', native_copy, symlinks=True)
    pc = native_copy / 'lib/pkgconfig/highs.pc'
    pc.write_text(pc.read_text().replace(str(repo / '.native/highs'), str(native_copy)))
    subprocess.run(['bash', str(setup), '--project-path', str(project),
                    '--highs-root', str(native_copy)], check=True)
    # Reject an incompatible integer ABI before attempting a native solve.
    header = native_copy / 'include/highs/HConfig.h'
    header.write_text(header.read_text() + '\n#define HIGHSINT64\n')
    original = workspace.read_bytes()
    result = subprocess.run(['bash', str(setup), '--project-path', str(project),
                             '--highs-root', str(native_copy)])
    assert result.returncode != 0
    assert workspace.read_bytes() == original
    # Sourcing is usable in both common Unix login shells, with spaces in arguments.
    for shell in ('bash', 'zsh'):
        if shutil.which(shell):
            subprocess.run([shell, '-c', '''before=$-; source "$1" --highs-root "$2" || exit
                            first=$CGO_LDFLAGS
                            source "$1" --highs-root "$2" || exit
                            test "$first" = "$CGO_LDFLAGS" && test "$before" = "$-" && go run .''',
                            'test', str(repo / 'scripts/enter-dev.sh'),
                            str(repo / '.native/highs')], cwd=project, check=True)
print('Unix workspace and consumer checks passed.')
