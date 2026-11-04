# Architecture

```
cmd/slom            main: opens the store, builds ui.App, opens the MyGo window; e2e.go (tag e2e) adds MyGo's automation socket
internal/catalog    option definitions (data), Selection, requires/conflicts/host rules, tool detection, group text and sections
internal/render     Selection -> launch-option string (slot order, quoting)
internal/parse      launch-option string -> Selection (inverse of render; keeps unknown tokens); Group/Ungroup chips; Units/Diff
internal/steam      Steam discovery, VDF spans, game list, localconfig.vdf surgical writer, process control
internal/store      presets and per-game assignments (JSON under the XDG config dir), Effective/Derive
internal/ui         MyGo screens, the only package allowed to import MyGo: app.go (state, actions), view.go (window,
                    sidebar, header, tabs, dock, dialogs), cards.go (option cards, More example), theme.go (palette,
                    fonts, icons), assets/ (fonts with licenses, SVG icons)
docs/               SPEC, ARCHITECTURE, CATALOG, MYGO, TESTING, RELEASING, screenshot.png (generated)
scripts/            e2e.py (drives the real window)
packaging/          slom.svg, the application icon (shipped with each release)
install.sh          Linux installer: release download, checksum, binary, desktop entry, icon
.github/workflows/  ci.yml (tests on push and PR), release.yml (binaries on a v* tag)
```

## Data flow
1. `steam` lists installed games and reads each game's `LaunchOptions` from the user's `localconfig.vdf`.
2. Selecting a game loads its stored `store.Assignment` (`Effective` = preset + overrides + custom pieces) or, if none, `parse.Parse` of Steam's line.
3. Controls are generated from `catalog.Default()`; every change goes through `catalog.Set` (validation, `Host`/`Requires` auto-on, `Conflicts`, cascade off) and autosaves the assignment via `store.Derive`. Editing never touches Steam.
4. `render.Render(selection, tools)` produces the preview; `parse.Diff` compares it with Steam's line.
5. Apply: `steam.WriteManyLaunchOptions` (refuse if Steam runs, backup, atomic write, read-back verify, restore on failure). Force apply wraps it with `Install.Shutdown` / `Install.Start`.

## Model
- `Selection`: `Values` (option id to value, Bool = "1") plus verbatim `CustomEnv` / `CustomWrap` / `CustomArgs` and `ExtraArgs[host]` for unknown wrapper flags and `MANGOHUD_CONFIG` items.
- Render order: catalog env, config env (`MANGOHUD_CONFIG`), `CustomEnv`, wrappers by `Order` (gamemoderun, gamescope ... `--`, mangohud), `CustomWrap`, `%command%`, `CustomArgs`. A bare `%command%` is written as "no launch options".
- Parse normalizes long gamescope flags to short ones and reorders to canonical order; it never drops a token. `MANGOHUD_CONFIG` without the `mangohud` wrapper stays a custom env token; values that are not an option's "on" value (`PROTON_LOG=0`) stay custom.
- The editor always edits a game's *effective* `Selection`; `Derive` and `Effective` are inverses. Presets never carry custom pieces. Deleting a preset folds its values into linked games' overrides, so no game's line changes. `ForceSet` (an override wins over a conflicting preset value) and `Normalize` (drops orphaned values) exist for the merge. A corrupt JSON file is an error on `Open`, never overwritten.
- Pending is computed by the app, not stored: `lineOf(effective)` differs from Steam's line parsed and re-rendered (`App.steamNormalized`).

## Steam facts
- `userdata/<id>` is the 32-bit account id; `loginusers.vdf` keys are SteamID64 (`id64 - 76561197960265728`). `MostRecent` is absent on single-user setups, so users sort by `MostRecent`, then newest `Timestamp`.
- `steam.ParseVDF` records byte spans; `steam.SetLaunchOptions` is a pure splice of the affected bytes.
- `HasProtonPrefix` (a `compatdata/<appid>` directory) says the game ran through Proton once. It is not a native-or-Proton signal and is not shown.
- The owner's real lines are test cases: `PROTON_ENABLE_NGX_UPDATER=1 DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE=on DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION=render_preset_m` and `gamescope --force-grab-cursor -W 2560 -H 1440 -r 180 -- %command%`.

## Invariants (tested)
- `render(parse(x))` is stable; parse never drops a token; unknown pieces keep their position.
- `SetLaunchOptions` changes only the bytes of the target value (or inserts or removes one line); the rest of `localconfig.vdf` is byte-identical.
- Catalog: every option has a label, tooltip and group; `Conflicts` are symmetric; ids and spellings unique; every group has a summary.
- `Derive(Effective(a)) ~ a`; deleting a preset does not change any game's rendered line.
