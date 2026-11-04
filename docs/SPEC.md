# Spec and decisions

Settled in a design interview with the owner. Treat as decided.

## Goal
Setting Proton, DLSS, gamescope or MangoHud launch options in Steam means remembering
many unclear env vars and flags. slom lists installed games and lets you pick settings
from labeled controls, with a tooltip per option and a live preview of the whole command line.

## Applying settings
- **Apply** edits `localconfig.vdf` directly, only while Steam is closed.
- Only the `LaunchOptions` value of one app is rewritten (insert the key if missing); every other byte is preserved.
- Backup before every write (`~/.local/state/slom/backups/`, keep last 10), re-parse after writing, restore the backup if the check fails.
- Steam running: there is no banner. The sidebar shows "Steam is running / Changes wait for Apply", and the main button in the command dock becomes **Force apply**, with the line "Steam is running. Force apply will try to close it, apply, and reopen it." It still asks for confirmation (the dialog says Steam will be closed, the options saved, Steam opened again, and that games are closed too), then runs `steam -shutdown`, waits, writes, relaunches.
- **Copy** puts the full command line on the clipboard.
- **Test** (only if the spike works): launch the game with the current options without saving; the UI says it is temporary and Apply makes it permanent. Spike: does `steam://run/<id>//<options>/` honor wrappers/env vars? If not, drop Test entirely (do not launch outside Steam).

## Discovery
- Native (`~/.local/share/Steam`) and Flatpak (`~/.var/app/com.valvesoftware.Steam/.local/share/Steam`) Steam; extra libraries from `libraryfolders.vdf`.
- Most recent Steam user (`MostRecent`) by default; switchable.
- Installed games only (`appmanifest_*.acf`); filter tools/runtimes (Proton, Steam Linux Runtime, Steamworks Common Redistributables...).
- Searchable, sorted by name, marks games with options set. Native Linux games get a note that Proton options do not apply.

## Catalog
Data-driven Go definitions: type, allowed values, label, tooltip text, slot (`env`, `wrapper`, `after-command`), platform (`linux`; field reserved for Windows later), tool requirement, `requires`, `conflicts`.
Groups: Proton/Wine, NVIDIA/DLSS (preset A-M, NVAPI, indicator overlay), gamescope (output and internal resolution, refresh, fullscreen/borderless, mouse grab, scaler, HDR, frame limit), MangoHud (via `MANGOHUD_CONFIG`), `gamemoderun`, custom env var row.
Variable names/values must be verified against current Proton and DXVK-NVAPI docs.
- Missing tool (gamescope/mangohud/gamemoderun): options shown disabled with an install note; preview skips them.
- `requires`: auto-enable the dependency with a visible note. `conflicts`: conflicting control disabled with a tooltip saying why.

## Command assembly
Fixed order: env vars, `gamemoderun`, `gamescope ... --`, `mangohud`, `%command%`, custom args. Each option declares its slot; the renderer sorts.

## Presets and import
- Reusable presets, **linked** to games: a game uses one preset plus per-game overrides (overrides are marked, "reset to preset" exists). Editing a preset puts affected games in the sidebar's **PENDING APPLY** section until applied.
- Import of an existing launch line: recognized tokens become controls; the rest becomes custom args, shown as editable chips (one per unrecognized token or `-- wrapper` group) with a raw-text toggle; order preserved, nothing dropped.
- Data in `~/.config/slom/` as plain JSON (`presets.json`, `assignments.json`).

## Out of scope for v1
Windows, Proton version selection (`CompatToolMapping`), uninstalled games, packaging beyond `go build`, launching outside Steam.

## Tests
Table-driven render/parse tests with a round-trip check (render -> parse -> render is stable). `steam` package against fixture `.acf`/`.vdf` files. UI not unit-tested beyond MyGo's windowless view tests if cheap.
