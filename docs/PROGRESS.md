# Progress

Milestones (one commit each):

- [x] M0 Scaffold, spec and agent docs
- [x] M1 `steam` package: discovery, games, VDF read, localconfig writer, process detection
- [x] M2 `catalog`, `render`, `parse` with round-trip tests
- [x] M3 `store`: presets, assignments (backups already live in `steam.WriteLaunchOptions`)
- [x] M4 `ui`: game list, preset bar, tabbed option editor, extras chips, preview, apply flows
- [x] M5 Test-button spike: run for real; Steam shows a ShowGameArgs dialog and never applies %command%/wrappers, so Test is dropped (`docs/SPIKE-TEST-BUTTON.md`)
- [x] M6 README with screenshot, final docs

Open risks: Test spike outcome; MyGo native UI widget fit (tooltips, select, list).

## Notes from M1
- `userdata/<id>` is the 32-bit account id; `loginusers.vdf` keys are SteamID64 (`id64 - 76561197960265728`). `MostRecent` is absent on single-user setups, so users sort by `MostRecent`, then newest `Timestamp`.
- `steam.ParseVDF` records byte spans; `steam.SetLaunchOptions` is a pure function that splices only the affected bytes. `steam.WriteLaunchOptions` adds the Steam-running guard, timestamped backup (+prune), atomic write, read-back verify and restore.
- `HasProtonPrefix` (a `compatdata/<appid>` dir exists) is the only native-vs-Proton signal; false means native or never run.
- Real launch lines on the owner's machine use `PROTON_ENABLE_NGX_UPDATER=1 DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE=on DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION=render_preset_m` and `gamescope --force-grab-cursor -W 2560 -H 1440 -r 180 -- %command%`; the catalog must parse these (use as test cases).
- `SLOM_REAL_STEAM=1 go test ./internal/steam -run Real -v` reads the real install (read-only).

## Notes from M2
- Selection model: `Values` (option id -> value, Bool = "1") plus verbatim `CustomEnv` / `CustomWrap` / `CustomArgs` and `ExtraArgs[host]` for unknown wrapper flags/config items. Render order: catalog env, config env (`MANGOHUD_CONFIG`), CustomEnv, wrappers by `Order` (gamemoderun, gamescope ... `--`, mangohud), CustomWrap, `%command%`, CustomArgs.
- Parse normalizes long gamescope flags to short ones and reorders into canonical order; it never drops tokens (tested). `PROTON_LOG=0`-style values that are not the "on" value stay as custom env. `MANGOHUD_CONFIG` without the `mangohud` wrapper stays a custom env token.
- See `docs/CATALOG.md` for variable verification and which options are Proton-GE only.

## Notes from M3
- The editor always edits a game's *effective* `Selection`. `store.Derive(c, preset, sel)` turns it into an `Assignment` (preset id + `Overrides` + `Off` + game-only custom pieces); `store.Effective` is the inverse. Editing a preset therefore changes every linked game's effective line.
- Presets never carry custom pieces. Deleting a preset folds its values into linked games' overrides so their command lines do not change.
- `catalog.ForceSet` (override wins over a conflicting preset value) and `catalog.Normalize` (drops orphaned/invalid values) exist for this merge.
- "Out of date" for a game = rendered effective line differs from the line currently in `localconfig.vdf` (computed by the app layer, not stored).
- A corrupt JSON file is an error on `Open`; it is never overwritten silently.

## Notes from M4
- Files: `internal/ui/app.go` (state and actions: select/import, commit, presets, apply, close-Steam-and-apply), `view.go` (layout), `system.go` (display defaults), `files.go`, `ui_test.go` (Tester-based tests).
- Behavior: selecting a game imports Steam's line (no assignment is saved until the user edits). Every edit autosaves the game's `Assignment` (never touches Steam). Apply writes `localconfig.vdf` via `steam.WriteLaunchOptions`; a bare `%command%` is written as "no launch options". "Pending" = configured line != line in Steam.
- Int options start from the system: `catalog.Initial` uses `Option.DefaultFrom` (`screen.width/height/refresh`) resolved by `detectDefaults`.
- The binary was smoke-tested for 8 s against the real Steam install (read-only; nothing written).
- Not done / ideas: Windows, Proton version selection, a visible "overridden" legend, keyboard shortcuts.

## State at the end of the first pass
All milestones are done and committed; `go build ./... && go vet ./... && go test ./...` pass. Known gaps / next ideas:
- Test button: dropped, confirmed empirically (see `docs/SPIKE-TEST-BUTTON.md`).
- Proton version selection (`CompatToolMapping` in `config.vdf`) is out of scope for v1.
- The UI has only been exercised through MyGo's `Tester` and an 8 s real launch smoke test; a human pass on a real desktop (Apply against a real Steam, Flatpak Steam, multiple accounts) is still due.
- Windows support: the catalog has no platform field in use yet (the spec reserved one); `steam/detach_linux.go` is Linux-only.

## Follow-up fixes after review
- Preset names are unique (case-insensitive) and "No preset" is reserved (`store.SavePreset`); the preset select only relinks on `Changed()` (it used to relink games every frame when names collided).
- DLSS indicator is a Show/Hide select (see CATALOG.md).
- `Option.Platform` exists and is `"linux"` for every option (asserted in tests).
- Spike script now warns that the game may really start and asks for confirmation.

## Known limitations (not fixed)
- `TokenField` splits on commas, so a custom chip like `WINEDLLOVERRIDES=dxgi=n,b` breaks into two chips. Use "Edit as text…" for such values.
- Chips are one per token; the spec wanted a chip per unrecognized token *or `-- wrapper` group* (`taskset -c 0-3` shows as three chips).
- Not exercised: Flatpak Steam, multiple accounts, and the UI buttons on a real desktop (the underlying steps below were).

## Verified on the real machine (2026-11-03)
- `SLOM_REAL_APPLY=<appid> go test ./internal/steam -run RealApply`: wrote a launch option to the real `localconfig.vdf` with Steam closed, read it back, restored it; the file was byte-identical afterwards.
- `SLOM_REAL_CLOSE=<appid> go test ./internal/steam -run RealCloseApplyReopen`: refused to write while Steam ran (`ErrSteamRunning`), `steam -shutdown` and wait, write + restore, restart Steam, shut it down again. Afterwards every `LaunchOptions` line matched the original (only Steam's own tickets and change numbers differed).
- Both tests are opt-in (they touch the real install) and skipped by default.
