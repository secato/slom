# Progress

Milestones (one commit each):

- [x] M0 Scaffold, spec and agent docs
- [x] M1 `steam` package: discovery, games, VDF read, localconfig writer, process detection
- [x] M2 `catalog`, `render`, `parse` with round-trip tests
- [x] M3 `store`: presets, assignments (backups already live in `steam.WriteLaunchOptions`)
- [ ] M4 `ui`: vertical slice, then full screens
- [ ] M5 Test-button spike (`steam://run/<id>//<options>/`)
- [ ] M6 README with screenshot, final docs

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
