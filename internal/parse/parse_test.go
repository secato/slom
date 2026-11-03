package parse

import (
	"reflect"
	"strings"
	"testing"

	"github.com/secato/slom/internal/catalog"
	"github.com/secato/slom/internal/render"
)

func TestTokenize(t *testing.T) {
	got := Tokenize(`A="x y" 'p q' b\ c plain "esc\"d" %command% `)
	var raws, vals []string
	for _, tk := range got {
		raws, vals = append(raws, tk.Raw), append(vals, tk.Value)
	}
	wantRaw := []string{`A="x y"`, `'p q'`, `b\ c`, `plain`, `"esc\"d"`, `%command%`}
	wantVal := []string{`A=x y`, `p q`, `b c`, `plain`, `esc"d`, `%command%`}
	if !reflect.DeepEqual(raws, wantRaw) || !reflect.DeepEqual(vals, wantVal) {
		t.Fatalf("raws %q\nvals %q", raws, vals)
	}
}

// Real launch lines from the owner's Steam library, plus edge cases.
var lines = []string{
	"gamescope --force-grab-cursor -W 2560 -H 1440 -r 180 -- %command%",
	"PROTON_ENABLE_NGX_UPDATER=1 DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE=on DXVK_NVAPI_DRS_NGX_DLSS_SR_OVERRIDE_RENDER_PRESET_SELECTION=render_preset_m %command%",
	"gamescope -w 3840 -h 2160 -W 2560 -H 1440 -r 180 -f -- %command%",
	"%command% /WineDetectionEnabled:False",
	"",
	"-windowed -novid",
	"MANGOHUD_CONFIG=fps_limit=60,position=top-right,frametime,custom_thing=1 mangohud %command%",
	"FOO=bar gamemoderun taskset -c 0-3 %command% -a 'b c'",
	"gamescope --output-width=1920 --nested-height 1080 --weird-flag 12 --other -- mangohud %command%",
	"PROTON_LOG=0 PROTON_USE_WINED3D=1 %command%",
	"MANGOHUD_CONFIG=fps_limit=60 %command%", // config without the wrapper: must survive
	"gamescope -W 99999 -- %command%",        // out of range: kept as extra
}

func TestParseRealLines(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)

	s := Parse(c, lines[0])
	want := map[string]string{"gamescope": "1", "gs.force_grab": "1", "gs.out_w": "2560", "gs.out_h": "1440", "gs.refresh": "180"}
	if !reflect.DeepEqual(s.Values, want) || len(s.ExtraArgs)+len(s.CustomEnv)+len(s.CustomWrap)+len(s.CustomArgs) != 0 {
		t.Fatalf("line 0: %+v", s)
	}

	s = Parse(c, lines[1])
	if s.Values["dlss.sr_preset"] != "render_preset_m" || s.Values["dlss.sr_override"] != "1" || s.Values["dlss.ngx_updater"] != "1" {
		t.Fatalf("line 1: %+v", s)
	}

	s = Parse(c, lines[3])
	if !s.IsEmptyExceptArgs() || !reflect.DeepEqual(s.CustomArgs, []string{"/WineDetectionEnabled:False"}) {
		t.Fatalf("line 3: %+v", s)
	}

	s = Parse(c, lines[6])
	if s.Values["mh.fps_limit"] != "60" || s.Values["mh.position"] != "top-right" || s.Values["mh.frametime"] != "1" ||
		!reflect.DeepEqual(s.ExtraArgs["mangohud"], []string{"custom_thing=1"}) {
		t.Fatalf("line 6: %+v", s)
	}

	s = Parse(c, lines[7])
	if !reflect.DeepEqual(s.CustomEnv, []string{"FOO=bar"}) || !reflect.DeepEqual(s.CustomWrap, []string{"taskset", "-c", "0-3"}) ||
		!reflect.DeepEqual(s.CustomArgs, []string{"-a", "'b c'"}) || s.Values["gamemode"] != "1" {
		t.Fatalf("line 7: %+v", s)
	}

	s = Parse(c, lines[8])
	if s.Values["gs.out_w"] != "1920" || s.Values["gs.in_h"] != "1080" ||
		!reflect.DeepEqual(s.ExtraArgs["gamescope"], []string{"--weird-flag", "12", "--other"}) {
		t.Fatalf("line 8: %+v", s)
	}

	s = Parse(c, lines[9]) // PROTON_LOG=0 is not "on": it must stay custom
	if !reflect.DeepEqual(s.CustomEnv, []string{"PROTON_LOG=0"}) || s.Values["proton.wined3d"] != "1" {
		t.Fatalf("line 9: %+v", s)
	}

	s = Parse(c, lines[10])
	if got := render.Render(c, s, tools); got != "MANGOHUD_CONFIG=fps_limit=60 %command%" {
		t.Fatalf("orphan config changed: %s", got)
	}

	s = Parse(c, lines[11])
	if _, ok := s.Values["gs.out_w"]; ok || !reflect.DeepEqual(s.ExtraArgs["gamescope"], []string{"-W", "99999"}) {
		t.Fatalf("line 11: %+v", s)
	}
}

// render(parse(x)) is canonical, and rendering it again through parse changes nothing.
func TestRoundTripIsStable(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	for _, line := range lines {
		once := render.Render(c, Parse(c, line), tools)
		twice := render.Render(c, Parse(c, once), tools)
		if once != twice {
			t.Errorf("unstable for %q:\n once: %s\ntwice: %s", line, once, twice)
		}
	}
}

// Nothing is dropped: every token of the input is still in the output.
func TestNothingIsDropped(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	for _, line := range lines {
		out := render.Render(c, Parse(c, line), tools)
		count := map[string]int{}
		for _, tk := range Tokenize(out) {
			count[tk.Value]++
		}
		for _, tk := range Tokenize(line) {
			if tk.Value == "%command%" || len(tk.Value) > 0 && tk.Value[0] == 'M' && len(tk.Value) > 15 && tk.Value[:15] == "MANGOHUD_CONFIG" {
				continue
			}
			flag, _, _ := strings.Cut(tk.Value, "=")
			if isAlias(c, flag) { // long spellings are normalized to the short ones
				continue
			}
			if count[tk.Value] == 0 {
				t.Errorf("%q: token %q lost; output %q", line, tk.Value, out)
			}
			count[tk.Value]--
		}
	}
}

func TestRenderParseRenderFromSelection(t *testing.T) {
	c := catalog.Default()
	tools := catalog.AllTools(c)
	var s catalog.Selection
	for _, kv := range [][2]string{{"gs.out_w", "2560"}, {"gs.out_h", "1440"}, {"gs.filter", "fsr"}, {"gs.sharpness", "5"},
		{"mh.preset", "3"}, {"mh.fps_limit", "0"}, {"dlss.sr_preset", "render_preset_latest"}, {"proton.hdr", "1"}, {"gamemode", "1"}, {"proton.log", "1"}} {
		if _, err := c.Set(&s, kv[0], kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	line := render.Render(c, s, tools)
	back := Parse(c, line)
	if !reflect.DeepEqual(back.Values, s.Values) {
		t.Fatalf("values differ:\n%s\n got %v\nwant %v", line, back.Values, s.Values)
	}
	if again := render.Render(c, back, tools); again != line {
		t.Fatalf("not stable:\n%s\n%s", line, again)
	}
}

func isAlias(c *catalog.Catalog, flag string) bool {
	for i := range c.Options {
		for _, a := range c.Options[i].Aliases {
			if a == flag {
				return true
			}
		}
	}
	return false
}

func TestGroupAndUngroup(t *testing.T) {
	tests := []struct {
		in   []string
		want []string
	}{
		{[]string{"taskset", "-c", "0-3"}, []string{"taskset -c 0-3"}},
		{[]string{"taskset", "-c", "0-3", "nice", "-n", "5"}, []string{"taskset -c 0-3", "nice -n 5"}},
		{[]string{"-windowed", "-novid"}, []string{"-windowed", "-novid"}},
		{[]string{"-a", "'b c'", "-d"}, []string{"-a 'b c'", "-d"}},
		{[]string{"--weird-flag", "12", "--other"}, []string{"--weird-flag 12", "--other"}},
		{[]string{"/WineDetectionEnabled:False"}, []string{"/WineDetectionEnabled:False"}},
		{[]string{"env", "A=1", "--", "foo", "-x"}, []string{"env A=1 --", "foo -x"}},
		{nil, nil},
	}
	for _, tc := range tests {
		got := Group(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Group(%q) = %q, want %q", tc.in, got, tc.want)
		}
		if back := Ungroup(got); !reflect.DeepEqual(back, tc.in) && !(len(back) == 0 && len(tc.in) == 0) {
			t.Errorf("Ungroup(Group(%q)) = %q", tc.in, back)
		}
	}
	// an edited chip may hold several tokens, quoted values and commas survive, and empty chips vanish
	if got := Ungroup([]string{"A=1 B=2", "WINEDLLOVERRIDES=dxgi=n,b", "", "  "}); !reflect.DeepEqual(got, []string{"A=1", "B=2", "WINEDLLOVERRIDES=dxgi=n,b"}) {
		t.Errorf("Ungroup = %q", got)
	}
}
