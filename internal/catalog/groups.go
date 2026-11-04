package catalog

import "slices"

// What each group is, in a sentence or two, for the intro at the top of its tab.
var groupSummaries = map[string]string{
	"Proton":    "Compatibility-layer settings for Windows games: graphics backends, sync primitives, Wayland and HDR, and logging. They only apply to games that run through Proton.",
	"DLSS":      "NVIDIA DLSS through Proton's NVAPI: force a Super Resolution preset or DLAA, and show an on-screen indicator. Needs an NVIDIA RTX graphics card.",
	"Gamescope": "Micro-compositor that gives the game its own resolution, refresh rate, scaling and mouse capture, whatever your desktop is doing.",
	"MangoHud":  "Performance overlay drawn over the game: frame rate, frame times, temperatures and memory use.",
	"Other":     "Extras that wrap the game.",
	"More":      "Anything slom has no control for stays here, exactly as it was, and is kept in the command. Add your own too.",
}

// sections groups the options of a tab under small headings. An option without an entry is
// in no section; a group either has sections for all its options or for none.
var sections = map[string]string{
	"proton.wined3d": "Graphics", "proton.no_d3d11": "Graphics", "proton.no_d3d10": "Graphics",
	"proton.wayland": "Graphics", "proton.hdr": "Graphics", "proton.hide_nvidia": "Graphics",
	"proton.no_fsync": "Synchronization", "proton.no_ntsync": "Synchronization", "proton.ntsync": "Synchronization",
	"proton.log": "Debugging",

	"dlss.force_nvapi": "Override", "dlss.sr_override": "Override", "dlss.sr_preset": "Override",
	"dlss.dlaa": "Override", "dlss.ngx_updater": "Override",
	"dlss.indicator": "Diagnostics", "dlss.log": "Diagnostics",

	"gs.out_w": "Resolution and refresh", "gs.out_h": "Resolution and refresh", "gs.in_w": "Resolution and refresh",
	"gs.in_h": "Resolution and refresh", "gs.refresh": "Resolution and refresh", "gs.fps_limit": "Resolution and refresh",
	"gs.fullscreen": "Window and input", "gs.borderless": "Window and input", "gs.force_grab": "Window and input", "gs.grab": "Window and input",
	"gs.scaler": "Scaling", "gs.filter": "Scaling", "gs.sharpness": "Scaling",
	"gs.hdr": "Display", "gs.adaptive": "Display",

	"mh.preset": "Overlay", "mh.position": "Overlay", "mh.font_size": "Overlay", "mh.compact": "Overlay",
	"mh.frametime": "Metrics", "mh.gpu_temp": "Metrics", "mh.cpu_temp": "Metrics", "mh.ram": "Metrics", "mh.vram": "Metrics",
	"mh.fps_limit": "Limits",
}

// stops are the usual values of a number: a slider snaps to them, and the user can still type
// any other value in range. The rate options also get the screen's own refresh rate.
var stops = map[string][]int{
	"gs.refresh":   {30, 60, 90, 120, 144, 180, 240},
	"gs.fps_limit": {30, 60, 90, 120, 144, 180, 240},
	"mh.fps_limit": {30, 60, 90, 120, 144, 180, 240},
	"gs.sharpness": {0, 5, 10, 15, 20},
	"mh.font_size": {16, 20, 24, 32, 40, 48},
}

var units = map[string]string{"gs.refresh": "Hz", "gs.fps_limit": "fps", "mh.font_size": "px"}

// sectionOrder fixes the order of the headings where the catalog's own order is not the one to
// read in; a group without an entry keeps the order its options appear in.
var sectionOrder = map[string][]string{
	"Proton": {"Graphics", "Synchronization", "Debugging"},
}

// Summary is what a group is, for its tab's intro.
func (c *Catalog) Summary(group string) string { return groupSummaries[group] }

// Sections lists the headings of a group in the order they first appear: [""] when it has none.
// The master switch is not in a section: it is in the intro.
func (c *Catalog) Sections(group string) []string {
	var out []string
	seen := map[string]bool{}
	master := c.Master(group)
	for _, o := range c.Options {
		if o.Group == group && (master == nil || o.ID != master.ID) && !seen[o.Section] {
			seen[o.Section] = true
			out = append(out, o.Section)
		}
	}
	rank := func(s string) int {
		if i := slices.Index(sectionOrder[group], s); i >= 0 {
			return i
		}
		return len(sectionOrder[group])
	}
	slices.SortStableFunc(out, func(a, b string) int { return rank(a) - rank(b) })
	return out
}

// Master is the wrapper that the group's other options are flags of, such as gamescope in
// the Gamescope tab, or nil. The intro of the tab carries its switch.
func (c *Catalog) Master(group string) *Option {
	for i := range c.Options {
		if h := c.Options[i].Host; h != "" && c.Options[i].Group == group {
			if w := c.Get(h); w != nil && w.Group == group {
				return w
			}
		}
	}
	return nil
}
