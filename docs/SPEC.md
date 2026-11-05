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
- **Look**: "gamer", and it follows the desktop: dark or light, and the desktop's accent (see **Theme following** below). One accent color means "this will be applied", amber means
  Steam is in the way, cyan is information, red/green mark removed/added. Titles in Rajdhani,
  commands in JetBrains Mono. Everything must be buildable from MyGo parts, no webview.
- **Sidebar**: games with a tile in their own color. The subtitle is the **preset** the game uses, or, without one, the **categories the
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

## Distribution
GitHub Actions run the tests on every push and pull request, and build Linux binaries (amd64,
arm64, `CGO_ENABLED=0`) for a release when a `v*` tag is pushed. `install.sh` installs a release
into `~/.local` (binary, desktop entry, icon) after checking its checksum. Details in
`docs/RELEASING.md`. Packaging beyond that (deb, rpm, Flatpak, AUR) is not planned.

## Ideas not built
Windows; Proton version selection (`CompatToolMapping`); uninstalled games; packaging beyond
the release binaries and `install.sh`; a progress state while Steam restarts; arrow-key navigation in the game list; a legend
for the sidebar dots (amber pending, cyan has options in Steam); suggestions in the More fields
(slom does not invent variables).

## Theme following
Decided 2026-11-05 (replaces "always dark"). Rules, in order:
- **Dark or light**: a Dark / Light / System toggle at the bottom of the sidebar (saved in `~/.config/slom/settings.json`, the only setting; System is the default). `SLOM_THEME=dark|light`, when set, overrides the toggle and locks it. Under System: Omarchy's active theme (`mode` in `~/.local/state/omarchy/current/theme/colors.toml`) > the system's appearance as MyGo reads it (GTK setting, then the desktop portal's `color-scheme`, then a GTK theme name with "dark"). MyGo answers light when the desktop says nothing at all, so a bare window manager gets the light look; choosing Dark fixes it.
- **Accent**: Omarchy's `accent` > the desktop portal's `accent-color` (GNOME 47+, KDE Plasma; MyGo reads it) > slom's own lime. Only the accent follows. Omarchy's other colors are not used, because theme authors pick their named colors (`yellow`, `green`) by taste, not by role, and amber / cyan / red / green / violet mean something in slom.
- **Making a chosen accent usable** (`usableAccent`): chroma raised to at least 0.09 so a muted accent still stands out from the grey surfaces (a muted rose becomes a clearer rose, same hue); lightness clamped so it reads on its ground (dark: 0.70-0.95, light: 0.40-0.55); a colorless accent (grey, chroma under 0.015) has no hue, so slom keeps its own. The tints around the accent (selected card, borders, glow) come from its hue.
- **Live**: MyGo redraws on portal changes. Omarchy replaces its theme directory on every switch and a switch between two dark themes changes nothing in the portal, so slom polls `current/theme.generation` once a second and keeps the last good look while a switch is half done.
- **Known limits**: an accent near red or amber sits close to the removed or Steam-in-the-way colors (the +/- marks and labels still say which is which). Not followed: high contrast and text scale, other themes' palettes (KDE `kdeglobals`, GTK CSS).
