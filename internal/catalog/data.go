package catalog

import "strings"

// Option ids that other packages refer to by name.
const (
	IDGamescope = "gamescope"
	IDMangoHud  = "mangohud"
	IDGameMode  = "gamemode"
)

const (
	supportValve   = "Valve's Proton (all recent versions)."
	supportGE      = "Proton-GE and Proton-CachyOS only; Valve's Proton ignores it."
	supportNVAPI   = "NVIDIA GPUs, through DXVK-NVAPI (bundled with Valve's Proton 9 and newer)."
	supportGamescp = "Needs gamescope."
	supportMango   = "Needs MangoHud."
)

func letters(from, to byte, prefix string) []Choice {
	var out []Choice
	for c := from; c <= to; c++ {
		out = append(out, Choice{Value: prefix + string(c), Label: "Preset " + strings.ToUpper(string(c))})
	}
	return out
}

// Default returns the built-in catalog. Variable names and values were checked against
// Valve's Proton 11 (launcher script and bundled DXVK-NVAPI), the DXVK-NVAPI and MangoHud
// READMEs and `gamescope --help`; see docs/CATALOG.md for the sources and caveats.
func Default() *Catalog {
	dlssPresets := append([]Choice{
		{Value: "render_preset_latest", Label: "Latest"},
		{Value: "render_preset_default", Label: "Driver default"},
	}, letters('a', 'm', "render_preset_")...)

	scalers := []Choice{{"auto", "Auto"}, {"integer", "Integer (pixel-perfect)"}, {"fit", "Fit (keep aspect ratio)"}, {"fill", "Fill (crop to fill)"}, {"stretch", "Stretch"}}
	filters := []Choice{{"linear", "Linear"}, {"nearest", "Nearest (sharp pixels)"}, {"fsr", "AMD FSR"}, {"nis", "NVIDIA NIS"}, {"pixel", "Pixel"}, {"sgsr", "Snapdragon SGSR"}}
	positions := []Choice{{"top-left", "Top left"}, {"top-center", "Top center"}, {"top-right", "Top right"}, {"middle-left", "Middle left"}, {"middle-right", "Middle right"}, {"bottom-left", "Bottom left"}, {"bottom-center", "Bottom center"}, {"bottom-right", "Bottom right"}}
	hudPresets := []Choice{{"0", "No HUD"}, {"1", "FPS only"}, {"2", "Horizontal"}, {"3", "Extended"}, {"4", "Detailed"}}

	with := func(o Option, f func(*Option)) Option { f(&o); return o }
	gs := func(id, label, help, flag string, kind Kind, long ...string) Option {
		return Option{ID: id, Group: "Gamescope", Label: label, Help: help, Support: supportGamescp,
			Kind: kind, Slot: SlotWrapperArg, Name: flag, Aliases: long, Host: IDGamescope}
	}
	gsStart := map[string]string{"gs.out_w": "1920", "gs.out_h": "1080", "gs.in_w": "1920", "gs.in_h": "1080", "gs.refresh": "60", "gs.fps_limit": "60"}
	gsInt := func(id, label, help, flag string, min, max int, long ...string) Option {
		return with(gs(id, label, help, flag, Int, long...), func(o *Option) { o.Min, o.Max, o.Default = min, max, gsStart[id] })
	}
	gsSel := func(id, label, help, flag string, choices []Choice, long ...string) Option {
		return with(gs(id, label, help, flag, Select, long...), func(o *Option) { o.Choices = choices })
	}
	mh := func(id, label, help, key string, kind Kind) Option {
		return Option{ID: id, Group: "MangoHud", Label: label, Help: help, Support: supportMango,
			Kind: kind, Slot: SlotConfigItem, Name: key, Host: IDMangoHud}
	}
	return New([]Option{
		// ---- Proton ----
		{ID: "proton.log", Group: "Proton", Label: "Write a Proton log", Support: supportValve,
			Help: "Saves a debug log to ~/steam-<appid>.log. Turn it on when you need to report a bug or find out why a game fails to start; it slows loading a little.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_LOG"},
		{ID: "proton.wined3d", Group: "Proton", Label: "Use OpenGL instead of Vulkan (wined3d)", Support: supportValve,
			Help: "Replaces DXVK (Direct3D on Vulkan) with the older OpenGL-based wined3d for Direct3D 9/10/11. Slower; only for games that break on DXVK.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_USE_WINED3D"},
		{ID: "proton.no_d3d11", Group: "Proton", Label: "Disable Direct3D 11", Support: supportValve,
			Help: "Hides d3d11 so a game that can fall back to Direct3D 9 uses that instead. Helps some older games that run better on d3d9.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_NO_D3D11"},
		{ID: "proton.no_d3d10", Group: "Proton", Label: "Disable Direct3D 10", Support: supportValve,
			Help: "Hides d3d10 and dxgi so a Direct3D 10 game falls back to Direct3D 9.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_NO_D3D10"},
		{ID: "proton.no_fsync", Group: "Proton", Label: "Disable fsync", Support: supportValve,
			Help: "Turns off futex-based synchronization (fsync), which Proton uses by default for speed. Try it if a game hangs or stutters badly.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_NO_FSYNC"},
		{ID: "proton.no_ntsync", Group: "Proton", Label: "Disable ntsync", Support: "Valve's Proton 11 (present in its wineserver; not in the README).",
			Help: "Turns off ntsync, the kernel-based synchronization used when the ntsync module is loaded. Try it if a game hangs.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_NO_NTSYNC", Conflicts: []string{"proton.ntsync"}},
		{ID: "proton.ntsync", Group: "Proton", Label: "Use ntsync", Support: supportGE,
			Help: "Uses the kernel's ntsync module for Windows synchronization, which can reduce stutter. Needs the ntsync kernel module.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_USE_NTSYNC", Conflicts: []string{"proton.no_ntsync"}},
		{ID: "proton.wayland", Group: "Proton", Label: "Native Wayland", Support: supportGE,
			Help: "Runs the game with Wine's Wayland driver instead of XWayland. Needed for HDR; some games misbehave.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_ENABLE_WAYLAND"},
		{ID: "proton.hdr", Group: "Proton", Label: "HDR", Support: supportGE + " Needs a Wayland session with HDR enabled.",
			Help: "Enables HDR output for games that support it. Requires native Wayland, which is switched on for you.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_ENABLE_HDR", Requires: []string{"proton.wayland"}},
		{ID: "proton.hide_nvidia", Group: "Proton", Label: "Report NVIDIA GPU as AMD", Support: supportValve,
			Help: "Makes an NVIDIA GPU look like an AMD one. Some games refuse to start, or show a wrong error, when they see NVIDIA through Wine.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_HIDE_NVIDIA_GPU"},

		// ---- DLSS ----
		{ID: "dlss.force_nvapi", Group: "DLSS", Label: "Force-enable NVAPI", Support: supportNVAPI,
			Help: "Turns on DXVK-NVAPI for games where Proton keeps it off by default. DLSS needs NVAPI.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_FORCE_NVAPI"},
		{ID: "dlss.sr_override", Group: "DLSS", Label: "Override DLSS Super Resolution settings", Support: supportNVAPI,
			Help: "Master switch for the DLSS Super Resolution overrides below. Without it the game's own choice is used.",
			Kind: Bool, Slot: SlotEnv, Name: "DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE", OnValue: "on"},
		{ID: "dlss.sr_preset", Group: "DLSS", Label: "DLSS Super Resolution preset", Support: supportNVAPI,
			Help: "Forces which DLSS model (preset) the game uses, like NVIDIA's app does on Windows. 'Latest' always picks the newest; a letter pins one, for example M.",
			Kind: Select, Slot: SlotEnv, Name: "DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION",
			Choices: dlssPresets, Requires: []string{"dlss.sr_override"}},
		{ID: "dlss.dlaa", Group: "DLSS", Label: "Force DLAA", Support: supportNVAPI,
			Help: "DLAA runs DLSS at the full render resolution for the best anti-aliasing instead of upscaling. Forces it on in games that offer DLSS.",
			Kind: Bool, Slot: SlotEnv, Name: "DXVK_NVAPI_DRS_NGX_DLAA_OVERRIDE", OnValue: "on"},
		{ID: "dlss.indicator", Group: "DLSS", Label: "DLSS indicator on screen", Support: supportNVAPI + " Permanently changes the game's Wine registry.",
			Help: "Show displays NVIDIA's on-screen DLSS and frame generation indicator, so you can see that DLSS is active. The setting is written to the game's Wine prefix registry and STAYS after you remove this option. To get rid of the indicator, set Hide and start the game once, then set it back to Not set.",
			Kind: Select, Slot: SlotEnv, Name: "DXVK_NVAPI_SET_NGX_DEBUG_OPTIONS",
			Choices: []Choice{{"DLSSIndicator=1024,DLSSGIndicator=2", "Show"}, {"DLSSIndicator=0,DLSSGIndicator=0", "Hide (clears a shown indicator)"}}},
		{ID: "dlss.log", Group: "DLSS", Label: "Log DXVK-NVAPI", Support: supportNVAPI,
			Help: "Prints DXVK-NVAPI log messages. Useful when DLSS does not show up in a game.",
			Kind: Bool, Slot: SlotEnv, Name: "DXVK_NVAPI_LOG_LEVEL", OnValue: "info"},
		{ID: "dlss.ngx_updater", Group: "DLSS", Label: "Auto-update DLSS files (NGX updater)", Support: "Not documented by Valve; found in community setups. Check that your Proton build supports it.",
			Help: "Lets NGX download newer DLSS model files. Used in many community DLSS setups; if your Proton ignores it, it does nothing.",
			Kind: Bool, Slot: SlotEnv, Name: "PROTON_ENABLE_NGX_UPDATER"},

		// ---- Gamescope ----
		{ID: IDGamescope, Group: "Gamescope", Label: "Run inside gamescope", Support: "Needs gamescope.", Tool: "gamescope",
			Help: "Runs the game in gamescope, Valve's micro-compositor. It lets you set the resolution the game renders at, upscale to your screen, cap the frame rate and capture the mouse.",
			Kind: Bool, Slot: SlotWrapper, Name: "gamescope", Order: 20, Terminator: "--"},
		gsInt("gs.out_w", "Output width", "Width of the window gamescope shows on your display, in pixels. Usually your screen's width.", "-W", 1, 16384, "--output-width"),
		gsInt("gs.out_h", "Output height", "Height of the window gamescope shows on your display, in pixels.", "-H", 1, 16384, "--output-height"),
		gsInt("gs.in_w", "Game width", "Width the game renders at, in pixels. Lower than the output width makes the game cheaper to run; gamescope upscales it.", "-w", 1, 16384, "--nested-width"),
		gsInt("gs.in_h", "Game height", "Height the game renders at, in pixels.", "-h", 1, 16384, "--nested-height"),
		gsInt("gs.refresh", "Refresh rate", "Refresh rate gamescope reports to the game, in frames per second. Usually your monitor's refresh rate.", "-r", 1, 1000, "--nested-refresh"),
		with(gs("gs.fullscreen", "Fullscreen", "Makes the gamescope window fullscreen.", "-f", Bool, "--fullscreen"), func(o *Option) { o.Conflicts = []string{"gs.borderless"} }),
		with(gs("gs.borderless", "Borderless window", "Makes the gamescope window borderless instead of fullscreen.", "-b", Bool, "--borderless"), func(o *Option) { o.Conflicts = []string{"gs.fullscreen"} }),
		gs("gs.force_grab", "Always capture the mouse", "Keeps the mouse captured in relative mode all the time, instead of switching with the cursor's visibility. Fixes mouse escaping or camera issues in games.", "--force-grab-cursor", Bool),
		gs("gs.grab", "Grab the keyboard", "Makes gamescope take exclusive keyboard focus.", "-g", Bool, "--grab"),
		gsSel("gs.scaler", "Scaling mode", "How the game's image is placed on the output when the sizes differ.", "-S", scalers, "--scaler"),
		gsSel("gs.filter", "Upscale filter", "The upscaling method used when the game renders below the output size: FSR and NIS add sharpening, Nearest keeps pixels crisp.", "-F", filters, "--filter"),
		gsInt("gs.sharpness", "Upscale sharpness", "Sharpness of FSR/NIS upscaling, from 0 (sharpest) to 20 (softest).", "--sharpness", 0, 20, "--fsr-sharpness"),
		gs("gs.hdr", "HDR output", "Turns on HDR output. Needs an HDR display and a game that supports HDR through gamescope's color layer.", "--hdr-enabled", Bool),
		gsInt("gs.fps_limit", "Frame rate limit", "Caps the frame rate. Gamescope uses it as a divisor of the refresh rate, so values round to 1x, 1/2x, 1/3x... of it.", "--framerate-limit", 1, 1000),
		gs("gs.adaptive", "Adaptive sync (VRR)", "Enables variable refresh rate (FreeSync/G-Sync) if the display supports it.", "--adaptive-sync", Bool),

		// ---- MangoHud ----
		{ID: IDMangoHud, Group: "MangoHud", Label: "Show the MangoHud overlay", Support: "Needs MangoHud.", Tool: "mangohud",
			Help: "Shows MangoHud's performance overlay: frame rate, frame times, temperatures and more. With gamescope on, MangoHud runs inside it.",
			Kind: Bool, Slot: SlotWrapper, Name: "mangohud", Order: 30, ConfigEnv: "MANGOHUD_CONFIG"},
		with(mh("mh.preset", "HUD layout", "Which built-in layout MangoHud starts with.", "preset", Select), func(o *Option) { o.Choices = hudPresets }),
		with(mh("mh.position", "HUD position", "Where the overlay appears on screen.", "position", Select), func(o *Option) { o.Choices = positions }),
		with(mh("mh.fps_limit", "FPS limit", "Caps the game's frame rate with MangoHud's limiter. 0 means unlimited.", "fps_limit", Int), func(o *Option) { o.Min, o.Max, o.Unit, o.Default = 0, 1000, "fps", "60" }),
		with(mh("mh.font_size", "Font size", "Size of the overlay text.", "font_size", Int), func(o *Option) { o.Min, o.Max, o.Default = 8, 96, "24" }),
		mh("mh.frametime", "Show frame time", "Shows how long each frame takes, next to the FPS counter. Spikes show stutter that the average FPS hides.", "frametime", Bool),
		mh("mh.gpu_temp", "Show GPU temperature", "Shows the GPU temperature.", "gpu_temp", Bool),
		mh("mh.cpu_temp", "Show CPU temperature", "Shows the CPU temperature.", "cpu_temp", Bool),
		mh("mh.ram", "Show RAM use", "Shows system memory in use.", "ram", Bool),
		mh("mh.vram", "Show VRAM use", "Shows graphics memory in use.", "vram", Bool),
		mh("mh.compact", "Compact HUD", "Uses a smaller, denser overlay.", "hud_compact", Bool),

		// ---- Other ----
		{ID: IDGameMode, Group: "Other", Label: "Use GameMode", Support: "Needs gamemode (gamemoderun).", Tool: "gamemoderun",
			Help: "Runs the game through Feral's GameMode, which switches the CPU governor to performance and applies other tweaks while the game runs.",
			Kind: Bool, Slot: SlotWrapper, Name: "gamemoderun", Order: 10},
	})
}
