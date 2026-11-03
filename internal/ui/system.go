package ui

import (
	"encoding/json"
	"os/exec"

	"github.com/egoist/mygo"
	"github.com/secato/slom/internal/catalog"
)

// detectDefaults derives gamescope's starting values from the machine: the primary
// display's size in pixels, and its refresh rate when the compositor can tell
// (Hyprland, via hyprctl).
func detectDefaults() catalog.Defaults {
	d := catalog.Defaults{}
	if disp := mygo.Screen.PrimaryDisplay(); disp.Bounds.Width > 0 {
		scale := disp.ScaleFactor
		if scale <= 0 {
			scale = 1
		}
		d["screen.width"] = int(float64(disp.Bounds.Width)*scale + 0.5)
		d["screen.height"] = int(float64(disp.Bounds.Height)*scale + 0.5)
	}
	if hz := hyprlandRefresh(); hz > 0 {
		d["screen.refresh"] = hz
	}
	return d
}

// hyprlandRefresh returns the refresh rate of the focused Hyprland monitor, or 0.
func hyprlandRefresh() int {
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return 0
	}
	var mons []struct {
		Refresh float64 `json:"refreshRate"`
		Focused bool    `json:"focused"`
	}
	if json.Unmarshal(out, &mons) != nil {
		return 0
	}
	best := 0
	for _, m := range mons {
		if hz := int(m.Refresh + 0.5); m.Focused || best == 0 {
			best = hz
		}
	}
	return best
}
