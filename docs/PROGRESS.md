# Progress

Milestones (one commit each):

- [x] M0 Scaffold, spec and agent docs
- [x] M1 `steam` package: discovery, games, VDF read, localconfig writer, process detection
- [ ] M2 `catalog`, `render`, `parse` with round-trip tests
- [ ] M3 `store`: presets, assignments (backups already live in `steam.WriteLaunchOptions`)
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
