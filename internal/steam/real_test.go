package steam

import (
	"os"
	"testing"
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
