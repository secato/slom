package catalog

import (
	"errors"
	"strings"
	"testing"
)

func TestDefaultCatalogIsWellFormed(t *testing.T) {
	c := Default() // New panics on duplicate ids or dangling references
	seen := map[string]bool{}
	for _, o := range c.Options {
		if o.Platform != "linux" {
			t.Errorf("%s: platform %q, want linux for v1", o.ID, o.Platform)
		}
		if o.Label == "" || o.Help == "" || o.Group == "" {
			t.Errorf("%s: label, help and group are required (tooltips are mandatory)", o.ID)
		}
		switch o.Kind {
		case Select:
			if len(o.Choices) == 0 {
				t.Errorf("%s: select without choices", o.ID)
			}
			for _, ch := range o.Choices {
				if ch.Value == "" || ch.Label == "" {
					t.Errorf("%s: empty choice %+v", o.ID, ch)
				}
			}
		case Int:
			if o.Min >= o.Max {
				t.Errorf("%s: bad range %d-%d", o.ID, o.Min, o.Max)
			}
		}
		if (o.Slot == SlotWrapperArg || o.Slot == SlotConfigItem) && o.Host == "" {
			t.Errorf("%s: slot needs a host", o.ID)
		}
		if o.Slot == SlotConfigItem && c.Get(o.Host).ConfigEnv == "" {
			t.Errorf("%s: host %s has no ConfigEnv", o.ID, o.Host)
		}
		// no two options may be spelled the same in the same place
		for _, sp := range append([]string{o.Name}, o.Aliases...) {
			key := strings.Join([]string{string(rune('0' + o.Slot)), o.Host, sp}, "|")
			if seen[key] {
				t.Errorf("%s: spelling %q already used", o.ID, sp)
			}
			seen[key] = true
		}
		for _, x := range o.Conflicts {
			back := c.Get(x)
			found := false
			for _, y := range back.Conflicts {
				found = found || y == o.ID
			}
			if !found {
				t.Errorf("%s conflicts with %s but not the other way round", o.ID, x)
			}
		}
	}
}

func TestSetRequiresAndCascade(t *testing.T) {
	c := Default()
	var sel Selection
	notes, err := c.Set(&sel, "dlss.sr_preset", "render_preset_m")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Values["dlss.sr_override"] != "1" || len(notes) != 1 || !strings.Contains(notes[0], "turned on") {
		t.Fatalf("requires not applied: %v %v", sel.Values, notes)
	}
	// switching HDR on switches Wayland on; switching Wayland off switches HDR off
	if _, err := c.Set(&sel, "proton.hdr", "1"); err != nil {
		t.Fatal(err)
	}
	if sel.Values["proton.wayland"] != "1" {
		t.Fatal("wayland not required by hdr")
	}
	notes, _ = c.Set(&sel, "proton.wayland", "")
	if _, on := sel.Values["proton.hdr"]; on || len(notes) == 0 {
		t.Fatalf("hdr should turn off with wayland: %v %v", sel.Values, notes)
	}
	// turning a wrapper off clears its flags
	c.Set(&sel, "gs.out_w", "2560")
	if sel.Values[IDGamescope] != "1" {
		t.Fatal("gamescope not enabled by its flag")
	}
	c.Set(&sel, IDGamescope, "")
	if _, on := sel.Values["gs.out_w"]; on {
		t.Fatal("flag survived its wrapper")
	}
}

func TestSetValidationAndConflicts(t *testing.T) {
	c := Default()
	var sel Selection
	if _, err := c.Set(&sel, "gs.refresh", "abc"); err == nil {
		t.Error("non-number accepted")
	}
	if _, err := c.Set(&sel, "gs.sharpness", "99"); err == nil {
		t.Error("out-of-range accepted")
	}
	if _, err := c.Set(&sel, "gs.scaler", "bogus"); err == nil {
		t.Error("bad choice accepted")
	}
	if _, err := c.Set(&sel, "nope", "1"); err == nil {
		t.Error("unknown option accepted")
	}
	c.Set(&sel, "gs.fullscreen", "1")
	_, err := c.Set(&sel, "gs.borderless", "1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("want conflict, got %v", err)
	}
	if _, on := sel.Values["gs.borderless"]; on {
		t.Fatal("conflicting option was set")
	}
	tools := AllTools(c)
	if r := c.Reason(sel, tools, "gs.borderless"); !strings.Contains(r, "Fullscreen") {
		t.Errorf("reason %q", r)
	}
	if r := c.Reason(sel, tools, "gs.fullscreen"); r != "" {
		t.Errorf("set option should not be disabled: %q", r)
	}
	if r := c.Reason(sel, Tools{}, IDGamescope); !strings.Contains(r, "not installed") {
		t.Errorf("missing tool reason %q", r)
	}
}

func TestSelectionClone(t *testing.T) {
	a := Selection{Values: map[string]string{"x": "1"}, CustomArgs: []string{"-a"}, ExtraArgs: map[string][]string{"g": {"--z"}}}
	b := a.Clone()
	b.Values["x"] = "2"
	b.CustomArgs[0] = "-b"
	b.ExtraArgs["g"][0] = "--y"
	if a.Values["x"] != "1" || a.CustomArgs[0] != "-a" || a.ExtraArgs["g"][0] != "--z" {
		t.Fatal("clone shares state")
	}
}

func TestGroupsAreDescribedAndSectionedConsistently(t *testing.T) {
	c := Default()
	for _, g := range c.Groups() {
		if c.Summary(g) == "" {
			t.Errorf("group %q has no summary", g)
		}
		secs := c.Sections(g)
		if len(secs) > 1 {
			for _, s := range secs {
				if s == "" {
					t.Errorf("group %q mixes sectioned and unsectioned options", g)
				}
			}
		}
	}
	if m := c.Master("Gamescope"); m == nil || m.ID != IDGamescope {
		t.Errorf("gamescope master: %+v", m)
	}
	if m := c.Master("MangoHud"); m == nil || m.ID != IDMangoHud {
		t.Errorf("mangohud master: %+v", m)
	}
	if c.Master("Proton") != nil || c.Master("Other") != nil {
		t.Error("only host groups have a master switch")
	}
}

func TestStopsAreUsualValuesInRange(t *testing.T) {
	c := Default()
	for id := range stops {
		o := c.Get(id)
		if o == nil || o.Kind != Int {
			t.Errorf("stops for %q, which is not an Int option", id)
			continue
		}
		for i, v := range o.Stops {
			if v < o.Min || v > o.Max {
				t.Errorf("%s: stop %d is outside [%d, %d]", id, v, o.Min, o.Max)
			}
			if i > 0 && v <= o.Stops[i-1] {
				t.Errorf("%s: stops are not ascending: %v", id, o.Stops)
			}
		}
	}
	for _, o := range c.Options {
		sizes := o.ID == "gs.out_w" || o.ID == "gs.out_h" || o.ID == "gs.in_w" || o.ID == "gs.in_h" // chosen with size pills
		if o.Kind == Int && len(o.Stops) == 0 && !sizes {
			t.Errorf("%s: an Int option without usual values", o.ID)
		}
	}
}
