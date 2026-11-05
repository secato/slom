# slom: agent guide

slom is a native Go desktop tool (MyGo `ui`, no webview, no cgo) that lists installed Steam games and builds their launch options (Proton env vars, DLSS, gamescope, MangoHud, gamemode) from labeled controls, then writes them into Steam's `localconfig.vdf`. Linux only for v1.

Read first: `docs/SPEC.md` (decisions already made, with the reasons) and `docs/ARCHITECTURE.md` (packages, data flow, invariants). Then, when the work touches:
- the window (`internal/ui`): `docs/MYGO.md`, the MyGo patterns, the look, and test gotchas.
- an option, group or variable name: `docs/CATALOG.md`, where each name was verified and how to add one.
- running or writing tests, or anything that touches the real Steam: `docs/TESTING.md`.
- CI, releases, `install.sh` or the desktop entry: `docs/RELEASING.md`.

## Rules
- Go 1.27, module `github.com/secato/slom`; the binary is `slom` (`go build ./cmd/slom`). Verify with `go build ./... && go vet ./... && go test ./...`.
- `internal/ui` is the only package that imports MyGo; everything below it stays testable without a window.
- Options are data in the catalog; the controls, preview and parser are generated from it. Add an option to `data.go` (with a source for its name) and its `sections` entry, not UI code.
- Starting values are fixed usual ones (1920×1080, 60 Hz). Never read the screen, resolution or refresh rate from the machine: the owner has two monitors and detection reported the wrong one.
- `localconfig.vdf`: write it only while Steam is closed, edit only the `LaunchOptions` value as a text splice, back up first, verify after, restore on failure. Imported tokens slom does not know stay, in order.
- Never close, restart or write to the owner's real Steam to test something. Real-machine checks are opt-in (`docs/TESTING.md`) and need the owner's say-so.
- The look is "gamer" and follows the desktop (dark or light, and its accent; `docs/SPEC.md`). Colors are named by meaning in `internal/ui/theme.go` (`pal`, one dark and one light palette): accent = will be applied, amber = Steam in the way, cyan = information, red/green = removed/added. Never put a color literal in the view code. Buttons and labels keep normal case, because tests and `scripts/e2e.py` find them by text.
- A change to what the window looks like: regenerate `docs/screenshot.png` and `docs/screenshot-light.png` and look at both. A decision from the owner: write it in `docs/SPEC.md`, in the same commit.
- Wording shown to the user is plain English, short, and says what will happen.

## Commits
One commit per logical change, message in the imperative with a short reason. The repo's git config already has the author (Felipe Secato, GitHub noreply email); keep it. Local only: do not push or create a remote unless asked.
