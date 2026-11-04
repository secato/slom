# Catalog: sources and caveats

The catalog lives in `internal/catalog/data.go`. Every option has a label, a tooltip
(`Help`) and a `Support` note shown in the tooltip. Verified on 2026-11-03 against Proton 11.0.

## How each group was verified
| Group | Verified against |
|---|---|
| Proton (`PROTON_LOG`, `PROTON_USE_WINED3D`, `PROTON_NO_D3D11/10`, `PROTON_NO_FSYNC`, `PROTON_HIDE_NVIDIA_GPU`, `PROTON_FORCE_NVAPI`) | Valve Proton README and the `proton` launcher script shipped in `Proton 11.0` |
| `PROTON_NO_NTSYNC` | Strings in Proton 11.0's `wineserver`; **not** in the README |
| DLSS (`DXVK_NVAPI_DRS_NGX_*`, `DXVK_NVAPI_SET_NGX_DEBUG_OPTIONS`, `DXVK_NVAPI_LOG_LEVEL`) | dxvk-nvapi README (named DRS settings and the `DXVK_NVAPI_DRS_` env prefix) and strings in Proton 11.0's `nvapi64.dll` (`RENDER_PRESET_A`..`Z`, `Default`, `Latest`; values are case-insensitive, written lowercase as `render_preset_m`) |
| Gamescope flags | `gamescope --help` of the installed build |
| MangoHud keys | MangoHud README (`MANGOHUD_CONFIG`, comma-separated; `fps`, `frame_timing`, `cpu_stats`, `gpu_stats` are on by default and therefore not offered) |

## Not in Valve's Proton (shown with a Support note)
- `PROTON_USE_NTSYNC`, `PROTON_ENABLE_WAYLAND`, `PROTON_ENABLE_HDR`: Proton-GE / CachyOS options (GE-Proton changelog); absent from Valve's Proton 11.0.
- `PROTON_ENABLE_NGX_UPDATER`: used in the owner's real launch lines but found in no Valve source or doc; kept because the owner uses it, flagged as undocumented.

## Left out on purpose
`PROTON_FSR4_UPGRADE`, `PROTON_NO_STEAMINPUT`, `PROTON_ENABLE_NVAPI`, `PROTON_DLSS_UPGRADE`: not found in Proton 11.0 or in any source checked. Add them only with a source.

## DLSS indicator caveat
`DXVK_NVAPI_SET_NGX_DEBUG_OPTIONS` writes NGX registry keys into the game's Wine prefix permanently (dxvk-nvapi README). Removing the option does not hide the indicator: run once with the "Hide" value (`DLSSIndicator=0,DLSSGIndicator=0`). The option is therefore a Show/Hide select, not a switch.

## Rules the engine enforces
- `Host`: a wrapper flag or config item needs its wrapper (gamescope, mangohud); `Set` switches the wrapper on and turning the wrapper off clears its options.
- `Requires`: switched on automatically (`proton.hdr` -> `proton.wayland`, `dlss.sr_preset` -> `dlss.sr_override`).
- `Conflicts` must be symmetric (tested): `gs.fullscreen`/`gs.borderless`, `proton.ntsync`/`proton.no_ntsync`. `Set` returns `ErrConflict`; the UI uses `Reason()` to disable the control.
- Missing tools: `Render` skips the option and everything hosted by it.

## Adding an option
Add one `Option` to `Default()` only with a source for the variable or flag (see the table above, and "Left out on purpose"). Give it Label, Help, Support, Group; add it to `sections` (and `stops` for an Int) in `groups.go`. Int and Text options start from `Option.Default`, a fixed usual value: never derive it from the machine. The UI, preview, parser and tests pick it up; `TestDefaultCatalogIsWellFormed` checks the invariants.

## Groups, sections and summaries
`internal/catalog/groups.go` holds, for the UI only (no rendering effect): `groupSummaries` (text of each tab's intro, also a "More" entry), `sections` (heading of each option inside its tab; a group has sections for all its options or for none, the master switch excepted), `sectionOrder` (explicit order where the catalog's is not the one to read in), and `Master(group)` (the wrapper the group's other options are flags of). A test checks every group has a summary and that sectioning is consistent.
`stops` and `units` in the same file give Int options their usual values (a slider snaps to them) and display units. A test checks that stops are ascending and inside `[Min, Max]` and that every Int option has stops except the four size options, which use pills.
