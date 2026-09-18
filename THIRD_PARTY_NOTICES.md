# Third-party notices

The Go wrapper is licensed under the root `LICENSE`. HiGHS is built separately and is not vendored into the Go source package.

The build scripts use HiGHS v1.15.1, commit `04024d701f79feb8e2f18bc3df0dffc04ef05088`, copyright (c) 2026 HiGHS, under the MIT license:

- [HiGHS license](https://github.com/ERGO-Code/HiGHS/blob/v1.15.1/LICENSE.txt)
- [HiGHS third-party notices](https://github.com/ERGO-Code/HiGHS/blob/v1.15.1/THIRD_PARTY_NOTICES.md)

The Windows build copies both files to `.native/highs/share/doc/HIGHS`. The Docker image includes them under `/usr/share/doc/highs`.

Both builds explicitly disable HiPO and the HiGHS command-line executable. HiPO's optional dependencies include additional licenses; see upstream notices before changing build options. Do not assume this wrapper's MIT license covers every possible HiGHS configuration.

If you redistribute native binaries, retain applicable HiGHS notices and comply with the licenses of compiler runtime libraries and other dependencies you include. Toolchains are installed separately. `scripts/bundle-windows.ps1` creates a local application bundle and copies the HiGHS, wrapper, and toolchain notices into its `licenses` directory; supply notices for any additional application dependencies yourself.
