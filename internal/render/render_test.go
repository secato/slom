package render

import (
	"testing"

	"github.com/secato/slom/internal/catalog"
)

func sel(c *catalog.Catalog, kv ...string) catalog.Selection {
	var s catalog.Selection
	for i := 0; i < len(kv); i += 2 {
		if _, err := c.Set(&s, kv[i], kv[i+1]); err != nil {
			panic(err)
		}
	}
	return s
}

func TestRender(t *testing.T) {
	c := catalog.Default()
	all := catalog.AllTools(c)
	tests := []struct {
		name  string
		sel   catalog.Selection
		tools catalog.Tools
		want  string
	}{
		{"empty", catalog.Selection{}, all, "%command%"},
		{"env order and bool values", sel(c, "proton.log", "1", "dlss.sr_preset", "render_preset_m"),
			all, "PROTON_LOG=1 DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE=on DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION=render_preset_m %command%"},
		{"gamescope flags", sel(c, "gs.force_grab", "1", "gs.out_w", "2560", "gs.out_h", "1440", "gs.refresh", "180"),
			all, "gamescope -W 2560 -H 1440 -r 180 --force-grab-cursor -- %command%"},
		{"all wrappers in order", sel(c, "gs.fullscreen", "1", "mh.frametime", "1", "mh.position", "top-right", "gamemode", "1"),
			all, "MANGOHUD_CONFIG=position=top-right,frametime gamemoderun gamescope -f -- mangohud %command%"},
		{"missing tool is skipped with its options", sel(c, "gs.out_w", "1920", "proton.log", "1"),
			catalog.Tools{"gamescope": false}, "PROTON_LOG=1 %command%"},
		{"custom pieces keep their place",
			catalog.Selection{CustomEnv: []string{"FOO=bar"}, CustomWrap: []string{"taskset", "-c", "0-3"}, CustomArgs: []string{"-windowed", "'a b'"}},
			all, "FOO=bar taskset -c 0-3 %command% -windowed 'a b'"},
		{"indicator value is not mangled", sel(c, "dlss.indicator", "DLSSIndicator=1024,DLSSGIndicator=2"),
			all, "DXVK_NVAPI_SET_NGX_DEBUG_OPTIONS=DLSSIndicator=1024,DLSSGIndicator=2 %command%"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Render(c, tc.sel, tc.tools); got != tc.want {
				t.Errorf("\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	for in, want := range map[string]string{
		"plain": "plain", "a,b=c": "a,b=c", "a b": "'a b'", "it's": `'it'\''s'`, "": "''", "$x": "'$x'",
	} {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %s, want %s", in, got, want)
		}
	}
}
