# slom — agent guide

slom is a native Go desktop tool (MyGo `ui`, no webview, no cgo) that lists installed
Steam games and builds their launch options (Proton env vars, DLSS, gamescope,
MangoHud, gamemode) from labeled controls, then writes them into Steam's
`localconfig.vdf`. Linux only for v1.

Read in this order:
1. `docs/SPEC.md` — what we build and the decisions already made (do not re-litigate).
2. `docs/ARCHITECTURE.md` — packages, data flow, key invariants.
3. `docs/PROGRESS.md` — milestones, what is done, what is next, open risks.
4. `docs/MYGO.md` — notes on the MyGo UI API as used here.

## Rules
- Go 1.27, module `github.com/secato/slom`. Binary is `slom` (`go build ./cmd/slom`).
- Everything below `internal/ui` must not import MyGo (keeps it testable and reusable).
- Data-driven: options live in the catalog as data; the UI and the preview are generated from it. No per-option UI code. No hardcoded values that can be derived.
- Never write `localconfig.vdf` while Steam runs. Always back up first, verify after, restore on failure.
- Edit only the `LaunchOptions` value in `localconfig.vdf` (surgical text edit); never re-serialize the whole file.
- Unrecognized tokens in an imported launch line are preserved in order, never dropped.
- One commit per milestone. Commit author: Felipe Secato. Keep `docs/PROGRESS.md` current in the same commit.
- Verify: `go build ./... && go vet ./... && go test ./...`.
