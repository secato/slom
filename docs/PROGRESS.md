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

## Redesign pass (2026-11-04)
- Implemented the "gamer" redesign from the design canvas (https://claude.ai/artifact/7rR968RT4KVTKFx8Ndzpdz): dark theme, embedded fonts and icons, sidebar with game tiles and a **PENDING APPLY** section (replaces "N game(s) out of date"), hero header, tabs with counts of set options, option cards in two columns, and a command dock with a colored command, "Steam currently has: …", Copy / Edit as text / Re-import, and the main button.
- The Steam-running banner and its "Close Steam, apply, reopen" button are gone. While Steam runs the main button is **Force apply** (amber, with the explanation next to it); the confirmation dialog stays. Otherwise it is **Apply** (green).
- "Everything else" tab is now **More**; "Import from Steam" is **Re-import from Steam**; the tab dot is a count.
- Not built from the canvas (follow-ups): resolution preset pills and a refresh-rate slider (still number inputs), a progress state while Steam restarts, arrow-key navigation in the game list, a hover legend for the sidebar dots (amber = pending, cyan = has options in Steam).
- Verified: unit/Tester tests, `docs/screenshot.png` regenerated, and a read-only launch of the real window (Steam running) against the real games. `scripts/e2e.py` was not rerun after the redesign because Steam was running (it writes the real file); rerun it with Steam closed.

## Redesign, second pass (2026-11-04)
- Every tab opens with an intro card (`catalog.Summary`) saying what the group is; Gamescope and MangoHud carry their master switch in it (`catalog.Master`). Options sit under small headings (`Option.Section`, set from the `sections` table in `internal/catalog/groups.go`, ordered by `sectionOrder`).
- Option cards by kind (`internal/ui/cards.go`): switches in two columns, selects as cards, resolution as pills (your screen's size is included, plus a Custom pill with width and height fields) for output and game size, sliders for refresh rate, frame rate limits, sharpness and font size. A slider is dimmed until touched; "Clear" unsets it.
- Pending fix: Steam's line is read and re-rendered before comparing (`App.steamNormalized`), so a line that only differs in order or spacing is no longer "pending" forever. **Discard changes** (dock) drops the game's assignment and preset link and goes back to what Steam has; it replaces "Re-import from Steam". It is disabled when nothing differs.
- The header wash and the sidebar tile share the game's hue (`gameHue`, `washColor`, `tileColors` in `theme.go`).
- `scripts/e2e.py` reads the line from the label of the command box. The Copy and Apply steps were not rerun after this pass (Steam was running); a read-only variant (select, tab, switches, preview) passed against the real window.

## Known limitations
- Flatpak Steam is only covered by fixture tests (`FindInstalls`, `Command`): Flatpak Steam is not installed on the dev machine, so `flatpak run com.valvesoftware.Steam -shutdown` and its `/proc` detection were never run.
- Multiple Steam accounts are covered by `TestSwitchingSteamAccount` (fixtures), not by a real second account.
- Chip fields: editing a chip in place does not re-group it until the game is reopened; grouping is a heuristic (`parse.Group`: a program with its flags and their values, or a flag with its value).

## Verified on the real machine (2026-11-03)
- `SLOM_REAL_APPLY=<appid> go test ./internal/steam -run RealApply`: wrote a launch option to the real `localconfig.vdf` with Steam closed, read it back, restored it; the file was byte-identical afterwards.
- `SLOM_REAL_CLOSE=<appid> go test ./internal/steam -run RealCloseApplyReopen`: refused to write while Steam ran (`ErrSteamRunning`), `steam -shutdown` and wait, write + restore, restart Steam, shut it down again. Afterwards every `LaunchOptions` line matched the original (only Steam's own tickets and change numbers differed).
- Both tests are opt-in (they touch the real install) and skipped by default.
- Chips were reworked: `parse.Group`/`Ungroup` (one chip per program-with-flags or flag-with-value, `--` ends a group), each chip a text field edited in place, no comma splitting (`WINEDLLOVERRIDES=dxgi=n,b` stays one token). Chip state lives in `App.chips` while typing so a chip never re-splits under the cursor.
- Real-window end-to-end: `go build -tags e2e -o /tmp/slom-e2e ./cmd/slom && python3 scripts/e2e.py /tmp/slom-e2e "<game name>"` drives the built app through MyGo's automation plugin (`cmd/slom/e2e.go`, build tag `e2e`, socket from `MYGO_AUTOMATION`). Passed on 2026-11-03: selected a game, toggled gamescope switches, Copy put the line on the Wayland clipboard (`wl-paste`), Apply wrote the real `localconfig.vdf` with a backup, and the script restored it byte-identically. Needs Steam closed; slom's own config/backups go to a temp dir (`XDG_CONFIG_HOME`/`XDG_STATE_HOME`).
