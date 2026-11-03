package steam

import (
	"os"
	"testing"
	"time"
)

// TestRealSteam exercises the real install read-only; set SLOM_REAL_STEAM=1 to run it.
func TestRealSteam(t *testing.T) {
	if os.Getenv("SLOM_REAL_STEAM") == "" {
		t.Skip("set SLOM_REAL_STEAM=1")
	}
	home, _ := os.UserHomeDir()
	for _, in := range FindInstalls(home) {
		games, err := in.Games()
		if err != nil {
			t.Fatal(err)
		}
		users, _ := in.Users()
		t.Logf("install %+v: %d games, users %+v", in, len(games), users)
		for _, g := range games {
			t.Logf("  %s %s prefix=%v", g.AppID, g.Name, g.HasProtonPrefix)
		}
		for _, u := range users {
			src, _ := os.ReadFile(u.ConfigPath)
			lo, err := LaunchOptions(src)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("  user %s: %d with launch options", u.AccountID, len(lo))
			for id, v := range lo {
				t.Logf("    %s: %s", id, v)
			}
			// in-memory edit must change only a small span
			for id := range lo {
				out, err := SetLaunchOptions(src, id, "X=1 %command%")
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("  size delta for %s: %d", id, len(out)-len(src))
				break
			}
			if _, err := SetLaunchOptions(src, "424242", "Y=1 %command%"); err != nil {
				t.Fatal(err)
			}
		}
		t.Logf("steam running: %v", Running())
	}
}

// TestRealApply writes to the REAL localconfig.vdf and puts it back: set SLOM_REAL_APPLY to
// an installed app id. Steam must be closed. The file must be byte-identical afterwards.
func TestRealApply(t *testing.T) {
	app := os.Getenv("SLOM_REAL_APPLY")
	if app == "" {
		t.Skip("set SLOM_REAL_APPLY=<appid> (writes and restores the real localconfig.vdf)")
	}
	home, _ := os.UserHomeDir()
	in := FindInstalls(home)[0]
	users, _ := in.Users()
	path := users[0].ConfigPath
	orig, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	prev := mustLine(t, orig, app)
	opts := WriteOptions{BackupDir: t.TempDir(), Keep: 10}
	if err := WriteLaunchOptions(path, app, "SLOM_REAL_TEST=1 %command%", opts); err != nil {
		t.Fatal(err)
	}
	mid, _ := os.ReadFile(path)
	if got := mustLine(t, mid, app); got != "SLOM_REAL_TEST=1 %command%" {
		t.Fatalf("read back %q", got)
	}
	t.Logf("size delta while set: %d bytes", len(mid)-len(orig))
	if err := WriteLaunchOptions(path, app, prev, opts); err != nil {
		t.Fatal(err)
	}
	end, _ := os.ReadFile(path)
	if string(end) != string(orig) {
		t.Fatalf("file not restored byte-identically (%d vs %d bytes)", len(end), len(orig))
	}
}

func mustLine(t *testing.T, src []byte, app string) string {
	t.Helper()
	m, err := LaunchOptions(src)
	if err != nil {
		t.Fatal(err)
	}
	return m[app]
}

// TestRealCloseApplyReopen runs the "Close Steam, apply, reopen" sequence on the real
// install: SLOM_REAL_CLOSE=<appid>. Steam should be running. It sets and restores a launch
// option, restarts Steam, then shuts it down again to leave the machine as it was.
func TestRealCloseApplyReopen(t *testing.T) {
	app := os.Getenv("SLOM_REAL_CLOSE")
	if app == "" {
		t.Skip("set SLOM_REAL_CLOSE=<appid> (closes and restarts the real Steam)")
	}
	home, _ := os.UserHomeDir()
	in := FindInstalls(home)[0]
	users, _ := in.Users()
	path := users[0].ConfigPath
	orig, _ := os.ReadFile(path)
	prev := mustLine(t, orig, app)

	if !Running() {
		t.Fatal("start Steam first")
	}
	if err := WriteLaunchOptions(path, app, "X=1 %command%", WriteOptions{}); err != ErrSteamRunning {
		t.Fatalf("write while running: got %v, want ErrSteamRunning", err)
	}
	if err := in.Shutdown(60 * time.Second); err != nil {
		t.Fatal(err)
	}
	if Running() {
		t.Fatal("still running after Shutdown")
	}
	opts := WriteOptions{BackupDir: t.TempDir(), Keep: 10}
	if err := WriteLaunchOptions(path, app, "SLOM_REAL_TEST=1 %command%", opts); err != nil {
		t.Fatal(err)
	}
	if err := WriteLaunchOptions(path, app, prev, opts); err != nil {
		t.Fatal(err)
	}
	if err := in.Start("-silent"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 60 && !Running(); i++ {
		time.Sleep(time.Second)
	}
	if !Running() {
		t.Fatal("Steam did not come back")
	}
	t.Log("Steam restarted")
	time.Sleep(20 * time.Second) // let it settle before closing it again
	if err := in.Shutdown(60 * time.Second); err != nil {
		t.Fatal(err)
	}
}
