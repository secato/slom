package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/render"
)

func mk(c *catalog.Catalog, kv ...string) catalog.Selection {
	var s catalog.Selection
	for i := 0; i < len(kv); i += 2 {
		if _, err := c.ForceSet(&s, kv[i], kv[i+1]); err != nil {
			panic(err)
		}
	}
	return s
}

func TestPresetCRUDAndPersistence(t *testing.T) {
	c := catalog.Default()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := s.SavePreset(Preset{Name: "Native 1440p 165Hz!", Selection: mk(c, "gs.out_w", "2560")})
	b, _ := s.SavePreset(Preset{Name: "Native 1440p 165Hz!", Selection: catalog.Selection{}})
	if a.ID != "native-1440p-165hz" || b.ID != "native-1440p-165hz-2" {
		t.Fatalf("ids %q %q", a.ID, b.ID)
	}
	if _, err := s.SavePreset(Preset{Name: "  "}); err == nil {
		t.Error("empty name accepted")
	}
	// custom pieces are stripped from presets
	p, _ := s.SavePreset(Preset{Name: "x", Selection: catalog.Selection{CustomArgs: []string{"-a"}}})
	if len(p.Selection.CustomArgs) != 0 {
		t.Error("custom args stored in preset")
	}
	s.SetAssignment("570", Assignment{PresetID: a.ID, Overrides: map[string]string{"gs.refresh": "144"}})

	s2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(s2.Presets()) != 3 {
		t.Fatalf("presets %v", s2.Presets())
	}
	if got, ok := s2.Assignment("570"); !ok || got.Overrides["gs.refresh"] != "144" {
		t.Fatalf("assignment %+v", got)
	}
	if !reflect.DeepEqual(s2.GamesUsing(a.ID), []string{"570"}) {
		t.Fatal("GamesUsing")
	}
}

func TestCorruptFileIsAnErrorNotDataLoss(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "presets.json")
	os.WriteFile(path, []byte("{not json"), 0o644)
	if _, err := Open(dir); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(path); string(b) != "{not json" {
		t.Error("corrupt file was modified")
	}
}

func TestEffectiveAndDeriveRoundTrip(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	preset := Preset{ID: "p", Selection: mk(c, "gs.out_w", "2560", "gs.out_h", "1440", "gs.fullscreen", "1", "mh.frametime", "1")}

	// the game turns frametime off, changes the width, and swaps fullscreen for borderless
	want := mk(c, "gs.out_w", "1920", "gs.out_h", "1440", "gs.borderless", "1", "proton.log", "1")
	want.CustomArgs = []string{"-windowed"}
	want.CustomEnv = []string{"FOO=bar"}

	a := Derive(c, &preset, want)
	if a.PresetID != "p" || a.Overrides["gs.out_w"] != "1920" || a.Overrides["gs.borderless"] != "1" || a.Overrides["proton.log"] != "1" {
		t.Fatalf("overrides %+v", a)
	}
	if _, has := a.Overrides["gs.out_h"]; has {
		t.Error("unchanged value stored as override")
	}
	wantOff := map[string]bool{"gs.fullscreen": true, "mh.frametime": true}
	for _, id := range a.Off {
		delete(wantOff, id)
	}
	if len(wantOff) != 0 {
		t.Fatalf("off %v", a.Off)
	}
	got := Effective(c, &preset, a)
	if render.Render(c, got, tools) != render.Render(c, want, tools) {
		t.Fatalf("\n got %s\nwant %s", render.Render(c, got, tools), render.Render(c, want, tools))
	}
	if !a.Overridden()["gs.out_w"] || !a.Overridden()["mh.frametime"] {
		t.Error("Overridden misses ids")
	}
}

func TestPresetEditChangesLinkedGames(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	p := Preset{ID: "p", Selection: mk(c, "gs.refresh", "144")}
	a := Assignment{PresetID: "p", Overrides: map[string]string{"gs.out_w": "1920"}}
	before := render.Render(c, Effective(c, &p, a), tools)
	p.Selection = mk(c, "gs.refresh", "165")
	after := render.Render(c, Effective(c, &p, a), tools)
	if before == after || after != "gamescope -W 1920 -r 165 -- %command%" {
		t.Fatalf("before %q after %q", before, after)
	}
}

func TestDeletePresetKeepsGameConfiguration(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	s, _ := Open(t.TempDir())
	p, _ := s.SavePreset(Preset{Name: "p", Selection: mk(c, "gs.out_w", "2560", "proton.log", "1")})
	s.SetAssignment("1", Assignment{PresetID: p.ID, Off: []string{"proton.log"}, Overrides: map[string]string{"gs.refresh": "60"}})
	preset, _ := s.Preset(p.ID)
	a, _ := s.Assignment("1")
	before := render.Render(c, Effective(c, &preset, a), tools)

	if err := s.DeletePreset(c, p.ID); err != nil {
		t.Fatal(err)
	}
	a, _ = s.Assignment("1")
	if a.PresetID != "" {
		t.Fatal("still linked")
	}
	if after := render.Render(c, Effective(c, nil, a), tools); after != before {
		t.Fatalf("configuration changed: %q -> %q", before, after)
	}
	if len(s.Presets()) != 0 {
		t.Fatal("preset not removed")
	}
}

func TestNormalizeDropsOrphans(t *testing.T) {
	c := catalog.Default()
	s := catalog.Selection{Values: map[string]string{"gs.out_w": "1920", "dlss.sr_preset": "render_preset_m", "gs.refresh": "abc"}}
	c.Normalize(&s)
	if s.Values != nil {
		t.Fatalf("kept %v", s.Values)
	}
}
