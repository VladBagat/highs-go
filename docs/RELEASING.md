# First release

This folder is prepared for `github.com/VladBagat/highs-go`. Nothing here automatically creates a GitHub repository or publishes a release. The starting checkout had no Git repository.

## 1. Check names and contents

- Confirm that `highs-go` is the repository name you want. If you change it, update `go.mod`, the import in `cmd/example/main.go`, and the documentation to match before publishing.
- Confirm the copyright holder in `LICENSE` (`VladBagat`).
- Read the README, follow the [Windows development setup](../CONTRIBUTING.md#windows-development), then run `go test -count=1 ./...`, `go vet ./...`, and `go run ./cmd/example`.
- If Docker is available, run `docker build -t highs-go .` and `docker run --rm highs-go`.

## 2. Create the GitHub repository

Sign in as VladBagat and create a **public** repository named **highs-go**. Leave GitHub's README, license, and gitignore initialization options unchecked: those files already exist here.

In PowerShell at this project's root, initialize Git if it has not already been initialized:

```powershell
git init -b main
git add .
git diff --cached --stat
git status --short
```

Inspect what will be uploaded. `.native/`, `.idea/`, and executable outputs should be absent. Do not add credentials or generated native binaries. If Git asks for an identity, configure your preferred name and email; GitHub offers a private noreply email in account settings.

```powershell
git commit -m "Prepare initial Go HiGHS wrapper"
git remote add origin https://github.com/VladBagat/highs-go.git
git push -u origin main
```

If a repository or `origin` already exists, inspect it with `git status` and `git remote -v` and use the existing setup rather than repeating initialization or overwriting the remote.

## 3. Wait for CI

Open the repository's **Actions** tab. Both the Windows and Linux jobs in **Test** must pass on the commit you intend to release. Fix failures and push again before creating the tag. The Windows job builds with MSYS2 UCRT64; local validation using another MinGW distribution does not replace this check.

## 4. Publish v0.1.0

Once checks pass and the working tree is clean:

```powershell
git tag -a v0.1.0 -m "Initial personal-use release"
git push origin v0.1.0
```

The pushed tag makes the version available to Go tooling. On GitHub, you can also create a Release using **the existing `v0.1.0` tag**, with notes describing the supported LP surface, native prerequisites, and experimental status. No binary attachments or package-registry upload are required for a Go source library.

Do not move or overwrite a published version tag. Use `v0.1.1` for subsequent fixes. Keep incompatible API changes clearly documented while the project is pre-v1; a future v2 requires Go's `/v2` module-path convention.

## 5. Verify as a consumer

In a fresh directory outside this checkout, with the native environment configured:

```powershell
go mod init example.com/highs-smoke
go get github.com/VladBagat/highs-go@v0.1.0
```

Copy the README's complete Go example into `main.go`, then run:

```powershell
go mod tidy
go run .
```

Expect `objective=10 x=2 y=2`. Use a fresh project without a local `replace` directive so this really tests the published module. The package documentation will be available through [pkg.go.dev](https://pkg.go.dev/github.com/VladBagat/highs-go).

References: [Go module publishing](https://go.dev/doc/modules/publishing) and [module version numbering](https://go.dev/doc/modules/version-numbers).
