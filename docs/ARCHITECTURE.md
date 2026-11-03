# Architecture

```
cmd/slom            main: wires internal packages, opens the MyGo window
internal/catalog    option definitions (data) + tool detection
internal/render     (selection of options) -> launch-option string, ordered by slot
internal/parse      launch-option string -> selection + unrecognized tokens (inverse of render)
internal/steam      Steam discovery, VDF read, game list, localconfig.vdf surgical writer, process control
internal/store      presets and per-game assignments (JSON under XDG config dir)
internal/ui         MyGo screens (only package allowed to import MyGo)
```

Data flow: `steam` lists games and the current `LaunchOptions` -> `parse` turns it into a
`Selection` -> user edits via controls generated from `catalog` -> `render` produces the
command preview -> Apply: `steam.WriteLaunchOptions` (guard, backup, atomic write, verify, restore).

Status of each package is tracked in `PROGRESS.md`.
