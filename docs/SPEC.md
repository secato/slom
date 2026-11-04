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
- Steam running: there is no banner. The sidebar shows "Steam is running / Changes wait for Apply", and applying becomes **Force apply** It still asks for confirmation (the dialog says Steam will be closed, the options saved, Steam opened again, and that games are closed too), then runs `steam -shutdown`, waits, writes, relaunches.
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

## UX decisions added with the redesign
- Pending = slom's line differs from Steam's line **after Steam's line is parsed and re-rendered** (order and spacing do not count).
- **Discard changes** = forget slom's configuration for the game (and its preset link) and show what Steam has. There is no separate re-import.
- Every tab starts with a short description of the group; hosts (gamescope, MangoHud) have their master switch there.
- Resolution is chosen with pills (output and game size together), refresh rate and limits with sliders; the option IDs and the rendered flags are unchanged.

## Applying lives in the sidebar
- The dock has no apply button. Each game under **PENDING APPLY** has a bolt button (tooltip says what it does), and when two or more games are pending a full-width **Apply all · N** sits under them. With Steam running the same buttons read **Force apply** / **Force apply all · N**, are amber, and ask for confirmation (the dialog names the number of games). One write and one backup per batch (`steam.WriteManyLaunchOptions`).
- The dock keeps Copy, Edit as text, **Discard changes** and **Remove all options**. Removing all options empties the game's selection (including custom pieces and the preset link) as a normal pending change; applying it deletes the `LaunchOptions` line, Discard undoes it.

## The command dock
Two labeled parts, each with an info tooltip: **PREVIEW** (what slom will write, colored) and **DIFFERENCE** (against what Steam has now; Steam's own line is not shown separately) (`parse.Diff`: units added in green, removed in red; a piece that only moved is not a change; "no difference" when equal). A pill by the title says "not saved yet" or "matches Steam". Below: Copy, Edit as text, Discard changes, Remove all options. No other wording: automatic changes ("Run inside gamescope turned on because…", "Preset applied…") are toasts (`App.notify`), not text in the dock. The three parts scroll inside a 280 px area; the default window is 1180×860.

## Sidebar subtitle and the More tab
- Under each game's name the sidebar lists the **categories it sets** ("DLSS, Gamescope"; "More" for pieces slom has no control for; "Nothing set"). It does not say Proton or Native: the Proton prefix only shows the game ran through Proton once, so it was wrong for native builds. The header no longer says it either.
- The More tab shows an invented line with the places of the three fields lit up (variables cyan, commands amber, arguments violet, `%command%` green), and each field's label carries the same color dot (`slotColor`, `placeExample` in `cards.go`).

The More tab's fields are named **Env variables**, **Commands** and **Arguments** (the example line captions use the same words).
