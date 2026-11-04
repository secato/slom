# CI, releases and the installer

GitHub repository assumed by the files: `secato/slom` (`install.sh` reads `SLOM_REPO` to override it). The local repo has no remote yet; push it there for the workflows to run.

## CI (`.github/workflows/ci.yml`)
On every push to `main` and every pull request: `gofmt -l` must print nothing, `go vet`, `go test`, `go build` (also with `-tags e2e`), then a `CGO_ENABLED=0` build uploaded as the artifact `slom-linux-amd64` (14 days). The UI tests are windowless, so no display is needed. Only the official `actions/*` actions are used.

## Releases (`.github/workflows/release.yml`)
Push a tag: `git tag v0.1.0 && git push origin v0.1.0`. The workflow tests, then builds `slom-linux-amd64` and `slom-linux-arm64` (`CGO_ENABLED=0 -trimpath`, version in `main.version`, shown by `slom --version`), copies `packaging/slom.svg`, writes `checksums.txt` (sha256 of the three files) and creates the GitHub release with generated notes using `gh`. The binaries link glibc dynamically (MyGo loads the graphics libraries at run time), so musl distributions are not supported.

`install.sh` depends on those exact file names: `slom-linux-<arch>`, `slom.svg`, `checksums.txt`.

## Installer (`install.sh`)
Plain POSIX `sh`. Detects the CPU (x86_64, aarch64), downloads the three files from the latest release (or `--version vX.Y.Z`), verifies the checksums, and installs to `~/.local` (or `--prefix`): `bin/slom`, `share/applications/slom.desktop` (`Exec` is the absolute path, `StartupWMClass=slom` matches the window class), `share/icons/hicolor/scalable/apps/slom.svg`. It then refreshes the desktop and icon caches when those tools exist. `--uninstall` removes the three files and keeps the user's settings and backups.

## Testing the installer without GitHub
Build the files as the workflow does into a folder, serve it, and point the script at it:
```sh
mkdir rel && CGO_ENABLED=0 go build -trimpath -o rel/slom-linux-amd64 ./cmd/slom
cp rel/slom-linux-amd64 rel/slom-linux-arm64 && cp packaging/slom.svg rel/
(cd rel && sha256sum slom-linux-* slom.svg > checksums.txt && python3 -m http.server 8765 --bind 127.0.0.1) &
SLOM_BASE_URL=http://127.0.0.1:8765 ./install.sh --prefix /tmp/slom-prefix
desktop-file-validate /tmp/slom-prefix/share/applications/slom.desktop
```
A tampered binary must stop with "checksum mismatch" and install nothing. Checked this way on 2026-11-04 (install, bad checksum, unreachable server, uninstall, a prefix with a space); the GitHub side (workflows, the `latest/download` URLs) has not run yet.
