# Architecture

```
cmd/slom            main: opens the store, builds ui.App, opens the MyGo window
internal/catalog    option definitions (data), Selection, requires/conflicts/host rules, tool detection
internal/render     Selection -> launch-option string (slot order, quoting)
internal/parse      launch-option string -> Selection (inverse of render; keeps unknown tokens)
internal/steam      Steam discovery, VDF spans, game list, localconfig.vdf surgical writer, process control
internal/store      presets and per-game assignments (JSON under the XDG config dir), Effective/Derive
internal/ui         MyGo screens (the only package allowed to import MyGo)
docs/               SPEC, ARCHITECTURE, CATALOG, MYGO, PROGRESS, SPIKE-TEST-BUTTON, screenshot.png
scripts/            spike-test-launch.sh (re-verifies the dropped Test button idea)
```

## Data flow
1. `steam` lists installed games and reads each game's current `LaunchOptions` from the user's `localconfig.vdf`.
2. Selecting a game loads its stored `store.Assignment` (`Effective` = preset + overrides + custom pieces) or, if none, `parse.Parse` of Steam's line.
3. Controls are generated from `catalog.Default()`; every change goes through `catalog.Set` (validation, `Host`/`Requires` auto-on, `Conflicts`, cascade off) and autosaves the assignment via `store.Derive`.
4. `render.Render(selection, tools)` produces the live preview.
5. Apply: `steam.WriteLaunchOptions` (refuse if Steam runs -> backup -> atomic write -> read-back verify -> restore on failure). "Close Steam, apply, reopen" wraps it with `Install.Shutdown` / `Install.Start`.

## Invariants (tested)
- `render(parse(x))` is stable; parse never drops a token; unknown pieces keep their position (custom env / wrap / args, `ExtraArgs` for wrapper flags and `MANGOHUD_CONFIG` items).
- `SetLaunchOptions` changes only the bytes of the target value (or inserts one line); the rest of `localconfig.vdf` is byte-identical.
- Catalog: every option has a label, tooltip and group; `Conflicts` are symmetric; ids and spellings unique.
- `Derive(Effective(a)) ~ a`; deleting a preset does not change any game's rendered line.
