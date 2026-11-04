# Spec and decisions

Settled with the owner, in an interview and while using the app. Treat as decided; change one
only when the owner asks, and edit this file in the same commit.

## Goal
Setting Proton, DLSS, gamescope or MangoHud launch options in Steam means remembering many
unclear env vars and flags. slom lists installed games and lets you pick settings from labeled
controls, with a tooltip per option and a live preview of the whole command line.
Linux only for v1.

## Applying
- slom edits `localconfig.vdf` directly, only while Steam is closed. Only the `LaunchOptions`
  value of one app is rewritten (key inserted if missing, line removed when the value is
  empty); every other byte is kept.
- Before every write: backup to `~/.local/state/slom/backups/` (last 10 kept). After: re-parse;
  restore the backup if the check fails. A batch is one backup and one write.
- Steam running: the sidebar says "Steam is running / Changes wait for Apply" and applying
  becomes **Force apply**: after a confirmation (Steam will be closed, the options saved, Steam
  opened again, games closed too) slom runs `steam -shutdown`, waits, writes, relaunches.
- **Copy** puts the full command line on the clipboard.
- A **Test** button ("launch with these options without saving") was tried and dropped. Steam
  treats `steam://run/<id>//<options>/` as game arguments: it shows a confirmation dialog,
  never substitutes `%command%` (so wrappers cannot run) and cuts the options at the first
  `/`. Launching outside Steam is out of scope: it would not match what Steam does. The spike
  script is in git history (`git show 538f88c:scripts/spike-test-launch.sh`); it can start the
  real game, so run it only on an offline game without anticheat.

## Discovery
- Native (`~/.local/share/Steam`) and Flatpak (`~/.var/app/com.valvesoftware.Steam/.local/share/Steam`)
  Steam; extra libraries from `libraryfolders.vdf`. Most recent Steam user by default, switchable.
- Installed games only (`appmanifest_*.acf`); tools and runtimes filtered out (Proton, Steam
  Linux Runtime, Steamworks Common Redistributables...).

## Catalog
Data-driven Go definitions (see `docs/CATALOG.md`). Groups: Proton, DLSS (preset A to M, NVAPI,
indicator), Gamescope (sizes, refresh, window and input, scaling, HDR, frame limit), MangoHud
(through `MANGOHUD_CONFIG`), Other (`gamemoderun`).
- Missing tool (gamescope, mangohud, gamemoderun): its options show disabled with an install
  note; the preview skips them.
- `requires`: the dependency switches on automatically, and a toast says so. `conflicts`: the
  conflicting control is disabled with a tooltip saying why.
- Command order is fixed: env vars, `gamemoderun`, `gamescope ... --`, `mangohud`, custom
  commands, `%command%`, custom arguments. Each option declares its slot; the renderer sorts.

## Presets and import
- Presets are reusable and **linked** to games: a game uses one preset plus its own overrides
  (marked with a dot). Editing a preset makes the games that use it pending.
- Importing an existing line: what slom recognizes becomes controls, the rest is kept as custom
  pieces in the More tab, in order, nothing dropped.
- Data: plain JSON under `~/.config/slom/` (`presets.json`, `assignments.json`).

## UI decisions
- **Look**: always dark, "gamer". One accent color means "this will be applied", amber means
  Steam is in the way, cyan is information, red/green mark removed/added. Titles in Rajdhani,
  commands in JetBrains Mono. Everything must be buildable from MyGo parts, no webview.
- **Sidebar**: games with a tile in their own color. The subtitle lists the **categories the
  game sets** ("DLSS, Gamescope", "More", "Nothing set"). It does not say Proton or Native: a
  Proton prefix only shows the game ran through Proton once. Games whose line differs from
  Steam's sit under **PENDING APPLY**; each has a bolt button that applies it, and two or more
  get **Apply all · N** (**Force apply** wording while Steam runs). The dock has no apply button.
- **Pending** means slom's line differs from Steam's line after Steam's line is parsed and
  re-rendered, so a different order or spacing is not a change.
- **Tabs**: Proton, DLSS, Gamescope, MangoHud, Other, More, each with a count of set options. A tab
  starts with a short description of its group; gamescope and MangoHud carry their master switch
  there. Options sit under small headings.
- **Controls**: switches in two columns; selects as cards; resolutions as buttons (standard
  sizes plus Custom width and height); refresh rate, frame limits, sharpness and font size as
  sliders that snap to usual values plus a field that takes any value. Untouched sliders are
  dimmed. Starting values are fixed usual ones (1920×1080, 60 Hz): slom never reads the
  screen, because that was wrong with two monitors.
- **Dock** (bottom): **PREVIEW** (the colored line slom will write) and **DIFFERENCE** against
  what Steam has now (`parse.Diff`: added green, removed red; moved pieces are not changes). A
  pill says "not saved yet" or "matches Steam". Buttons: Copy, Edit as text, **Discard changes**
  (drop slom's configuration and preset link, show what Steam has), **Remove all options** (empty
  the game as a normal pending change; applying deletes the line). No other wording in the dock:
  automatic changes are toasts.
- **More tab**: fields **Env variables**, **Commands**, **Arguments**, with an invented example
  line showing where each lands (cyan, amber, violet; `%command%` green).
- Default window 1180×860.
- Design reference: the Claude design canvas https://claude.ai/artifact/7rR968RT4KVTKFx8Ndzpdz
  (the first design; the app has since moved on, this file wins).

## Ideas not built
Windows; Proton version selection (`CompatToolMapping`); uninstalled games; packaging beyond
`go build`; a progress state while Steam restarts; arrow-key navigation in the game list; a legend
for the sidebar dots (amber pending, cyan has options in Steam); suggestions in the More fields
(slom does not invent variables).
