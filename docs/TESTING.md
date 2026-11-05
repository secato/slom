# Testing and verification

Everything: `go build ./... && go vet ./... && go test ./...` (also `go build -tags e2e ./...`).
Run it before every commit.

## What the tests cover
- Table-driven render and parse tests with a round-trip check; `steam` against fixture `.vdf` and `.acf` files (`internal/steam/testdata`); catalog invariants; store flows.
- `internal/ui/ui_test.go`: MyGo's windowless `Tester` drives the real view (see `docs/MYGO.md` for its gotchas). `testApp` builds an `App` without a window over fixture games.

## Screenshot
`docs/screenshot.png` (dark) and `docs/screenshot-light.png` are generated: `SLOM_SCREENSHOT=$PWD/docs/screenshot.png go test ./internal/ui -run Screenshot` (absolute path; the light one is written next to it). Regenerate both in any commit that changes how the window looks, and look at them.

The look: `look_test.go` drives the dark/light choice with `Tester.SetDark` and `SetPreferences` (a desktop accent), checks the Omarchy and `SLOM_THEME` precedence, and checks the contrast of every palette with several accents. `internal/desktop` has the Omarchy file parser and the switch detection. Tests never read the real Omarchy files: `App.omarchy` is only filled by `Start`.

## Checks on the real machine (opt-in)
These touch the owner's real Steam. They restore what they change, but still: **do not run them without the owner's say-so, and never while the owner is playing**. Steam must be closed unless noted.
- `SLOM_REAL_STEAM=1 go test ./internal/steam -run Real -v`: reads the real install (read-only, Steam may run).
- `SLOM_REAL_APPLY=<appid> go test ./internal/steam -run RealApply`: writes a launch option into the real `localconfig.vdf`, reads it back, restores it byte-identically.
- `SLOM_REAL_CLOSE=<appid> go test ./internal/steam -run RealCloseApplyReopen`: the real Force apply sequence (shuts Steam down, writes, restarts Steam).
- `scripts/e2e.py`: drives the built window through MyGo's automation plugin (`MYGO_AUTOMATION` socket, build tag `e2e`): select a game, toggle switches, Copy (checks the Wayland clipboard with `wl-paste`), Apply from the sidebar (writes the real file, checks the backup, restores). slom's own config and backups go to a temp dir.
  `go build -tags e2e -o /tmp/slom-e2e ./cmd/slom && python3 scripts/e2e.py /tmp/slom-e2e "<game name>"`
  It reads the line from the label "Launch options preview: <line>" and presses "Apply <game name>", so keep those labels.

Last full run of the real checks: 2026-11-03, before the redesign. After the redesign only a read-only variant of the e2e (select, tab, switches, preview) ran, because Steam was running.

## Known limitations
- Flatpak Steam is covered by fixtures only (`FindInstalls`, `Command`): `flatpak run com.valvesoftware.Steam -shutdown` and its `/proc` detection never ran for real.
- Multiple accounts are covered by `TestSwitchingSteamAccount` (fixtures), not a real second account.
- A chip edited in place keeps its shape until the game is reopened; grouping is a heuristic (`parse.Group`: a program with its flags and their values, or a flag with its value).
