package steam

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/localconfig.vdf")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseVDFEscapesAndComments(t *testing.T) {
	src := []byte("// c\n\"a\" { \"k\" \"x\\\"y\\\\z\\n\" unq 12 }\n")
	root, err := ParseVDF(src)
	if err != nil {
		t.Fatal(err)
	}
	a := root.Child("A") // case-insensitive
	if a == nil || a.Str("k") != "x\"y\\z\n" || a.Str("unq") != "12" {
		t.Fatalf("bad parse: %+v", a)
	}
	for _, bad := range []string{`"a" {`, `"a"`, `"a" "b`, `}`} {
		if _, err := ParseVDF([]byte(bad)); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestLaunchOptionsRead(t *testing.T) {
	got, err := LaunchOptions(fixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["570"] != "gamescope -W 2560 -H 1440 -- %command%" {
		t.Fatalf("got %v", got)
	}
}

func TestSetLaunchOptions(t *testing.T) {
	orig := fixture(t)
	tests := []struct {
		name, app, value string
		wantContains     string
		wantGone         string
	}{
		{"replace", "570", "mangohud %command%", `"LaunchOptions"` + "\t\t" + `"mangohud %command%"`, "gamescope"},
		{"insert into existing app", "1245620", "A=1 %command%", `"LaunchOptions"` + "\t\t" + `"A=1 %command%"`, ""},
		{"insert into empty app", "7", `X="a b" %command%`, `"X=\"a b\" %command%"`, ""},
		{"new app", "999", "B=2 %command%", `"999"`, ""},
		{"remove", "570", "", "", "LaunchOptions"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := SetLaunchOptions(orig, tc.app, tc.value)
			if err != nil {
				t.Fatal(err)
			}
			s := string(out)
			if tc.wantContains != "" && !strings.Contains(s, tc.wantContains) {
				t.Errorf("missing %q in:\n%s", tc.wantContains, s)
			}
			if tc.wantGone != "" && strings.Contains(s, tc.wantGone) {
				t.Errorf("still contains %q", tc.wantGone)
			}
			got, err := LaunchOptions(out)
			if err != nil {
				t.Fatalf("result does not parse: %v\n%s", err, s)
			}
			if got[tc.app] != tc.value {
				t.Errorf("read back %q, want %q", got[tc.app], tc.value)
			}
			// the decoy "apps" outside the Steam block must be untouched
			if !strings.Contains(s, `"apps"`+"\t\t"+`"decoy"`) {
				t.Error("decoy changed")
			}
		})
	}
}

func TestSetLaunchOptionsOnlyTouchesTarget(t *testing.T) {
	orig := fixture(t)
	out, err := SetLaunchOptions(orig, "570", "x %command%")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(string(orig), "gamescope -W 2560 -H 1440 -- %command%", "x %command%", 1)
	if string(out) != want {
		t.Fatalf("collateral change:\n%s", out)
	}
	// setting the same value is a no-op, and removing a missing key is too
	same, _ := SetLaunchOptions(orig, "570", "gamescope -W 2560 -H 1440 -- %command%")
	if string(same) != string(orig) {
		t.Error("same value changed bytes")
	}
	none, _ := SetLaunchOptions(orig, "1245620", "")
	if string(none) != string(orig) {
		t.Error("removing a missing key changed bytes")
	}
}

func TestSetLaunchOptionsNoAppsBlock(t *testing.T) {
	if _, err := SetLaunchOptions([]byte(`"UserLocalConfigStore" { }`), "1", "x"); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteLaunchOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "localconfig.vdf")
	if err := os.WriteFile(path, fixture(t), 0o644); err != nil {
		t.Fatal(err)
	}
	bdir := filepath.Join(dir, "backups")
	tick := 0
	o := WriteOptions{
		BackupDir: bdir, Keep: 2,
		SteamRunning: func() bool { return false },
		Now:          func() time.Time { tick++; return time.Unix(1700000000+int64(tick), 0) },
	}
	for _, v := range []string{"A=1 %command%", "A=2 %command%", "A=3 %command%"} {
		if err := WriteLaunchOptions(path, "570", v, o); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := os.ReadFile(path)
	if got, _ := LaunchOptions(data); got["570"] != "A=3 %command%" {
		t.Fatalf("got %v", got)
	}
	baks, _ := os.ReadDir(bdir)
	if len(baks) != 2 {
		t.Fatalf("want 2 backups kept, have %d", len(baks))
	}
	// the newest backup holds the state before the last write
	last, _ := os.ReadFile(filepath.Join(bdir, baks[len(baks)-1].Name()))
	if got, _ := LaunchOptions(last); got["570"] != "A=2 %command%" {
		t.Errorf("backup has %v", got)
	}
}

func TestWriteLaunchOptionsRefusesWhileRunning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "localconfig.vdf")
	os.WriteFile(path, fixture(t), 0o644)
	err := WriteLaunchOptions(path, "570", "x", WriteOptions{SteamRunning: func() bool { return true }})
	if !errors.Is(err, ErrSteamRunning) {
		t.Fatalf("got %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != string(fixture(t)) {
		t.Error("file modified while Steam running")
	}
}

// steamTree builds a fake Steam install with two libraries, two games, a tool and users.
func steamTree(t *testing.T) Install {
	t.Helper()
	root := t.TempDir()
	lib2 := filepath.Join(t.TempDir(), "SteamLibrary")
	w := func(p, s string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w(filepath.Join(root, "steamapps/libraryfolders.vdf"),
		`"libraryfolders" { "0" { "path" "`+root+`" } "1" { "path" "`+lib2+`" } "2" { "path" "/nonexistent" } }`)
	manifest := func(id, name, flags string) string {
		return `"AppState" { "appid" "` + id + `" "name" "` + name + `" "StateFlags" "` + flags + `" "installdir" "` + name + `" "LastPlayed" "5" }`
	}
	w(filepath.Join(root, "steamapps/appmanifest_1.acf"), manifest("1", "Zelda", "4"))
	w(filepath.Join(root, "steamapps/appmanifest_2.acf"), manifest("2", "Proton 11.0", "4"))
	w(filepath.Join(root, "steamapps/appmanifest_3.acf"), manifest("3", "Downloading", "1026"))
	w(filepath.Join(root, "steamapps/compatdata/1/pfx.lock"), "")
	w(filepath.Join(lib2, "steamapps/appmanifest_4.acf"), manifest("4", "alpha", "4"))
	w(filepath.Join(root, "userdata/0/config/compat.vdf"), `"x" "y"`)
	w(filepath.Join(root, "userdata/100/config/localconfig.vdf"), `"UserLocalConfigStore" {}`)
	w(filepath.Join(root, "userdata/200/config/localconfig.vdf"), `"UserLocalConfigStore" {}`)
	w(filepath.Join(root, "userdata/300/config/other.vdf"), ``)
	w(filepath.Join(root, "config/loginusers.vdf"),
		`"users" { "76561197960265928" { "PersonaName" "old" "Timestamp" "10" } "76561197960265828" { "PersonaName" "new" "Timestamp" "20" } }`)
	return Install{Root: root}
}

func TestLibrariesGamesUsers(t *testing.T) {
	in := steamTree(t)
	libs, err := in.Libraries()
	if err != nil || len(libs) != 2 {
		t.Fatalf("libs %v err %v", libs, err)
	}
	games, err := in.Games()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, g := range games {
		names = append(names, g.Name)
	}
	if strings.Join(names, ",") != "alpha,Zelda" { // sorted case-insensitively; tool and half-installed filtered
		t.Fatalf("games %v", names)
	}
	if !games[1].HasProtonPrefix || games[0].HasProtonPrefix {
		t.Error("proton prefix detection wrong")
	}
	users, err := in.Users()
	if err != nil || len(users) != 2 {
		t.Fatalf("users %v err %v", users, err)
	}
	if users[0].AccountID != "100" || users[0].Name != "new" {
		t.Errorf("most recent user wrong: %+v", users[0])
	}
}

func TestFindInstallsDedupesSymlinks(t *testing.T) {
	home := t.TempDir()
	native := filepath.Join(home, ".local/share/Steam")
	os.MkdirAll(filepath.Join(native, "steamapps"), 0o755)
	os.MkdirAll(filepath.Join(home, ".steam"), 0o755)
	os.Symlink(native, filepath.Join(home, ".steam/steam"))
	fp := filepath.Join(home, ".var/app/com.valvesoftware.Steam/.local/share/Steam")
	os.MkdirAll(filepath.Join(fp, "steamapps"), 0o755)
	got := FindInstalls(home)
	if len(got) != 2 || got[0].Flatpak || !got[1].Flatpak {
		t.Fatalf("got %+v", got)
	}
}

func TestRunningIn(t *testing.T) {
	proc := t.TempDir()
	os.MkdirAll(filepath.Join(proc, "123"), 0o755)
	os.WriteFile(filepath.Join(proc, "123/comm"), []byte("bash\n"), 0o644)
	if runningIn(proc) {
		t.Fatal("false positive")
	}
	os.MkdirAll(filepath.Join(proc, "456"), 0o755)
	os.WriteFile(filepath.Join(proc, "456/comm"), []byte("steam\n"), 0o644)
	if !runningIn(proc) {
		t.Fatal("steam not detected")
	}
}
