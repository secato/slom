package steam

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// Install is one Steam installation.
type Install struct {
	Root    string // the directory holding steamapps/, config/ and userdata/
	Flatpak bool
}

// FindInstalls returns the Steam installs under home: native and Flatpak, deduplicated
// by resolved path (~/.steam/steam is usually a symlink to the native one).
func FindInstalls(home string) []Install {
	candidates := []Install{
		{Root: filepath.Join(home, ".local/share/Steam")},
		{Root: filepath.Join(home, ".steam/steam")},
		{Root: filepath.Join(home, ".steam/root")},
		{Root: filepath.Join(home, ".var/app/com.valvesoftware.Steam/.local/share/Steam"), Flatpak: true},
	}
	var out []Install
	seen := map[string]bool{}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c.Root, "steamapps")); err != nil {
			continue
		}
		real, err := filepath.EvalSymlinks(c.Root)
		if err != nil || seen[real] {
			continue
		}
		seen[real] = true
		c.Root = real
		out = append(out, c)
	}
	return out
}

// Libraries returns every library folder of the install, the install's own first.
func (in Install) Libraries() ([]string, error) {
	libs := []string{in.Root}
	data, err := os.ReadFile(filepath.Join(in.Root, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		if os.IsNotExist(err) {
			return libs, nil
		}
		return nil, err
	}
	root, err := ParseVDF(data)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{in.Root: true}
	if lf := root.Child("libraryfolders"); lf != nil {
		for _, entry := range lf.Children {
			p := entry.Str("path")
			if p == "" || seen[p] {
				continue
			}
			if _, err := os.Stat(filepath.Join(p, "steamapps")); err != nil {
				continue
			}
			seen[p] = true
			libs = append(libs, p)
		}
	}
	return libs, nil
}

// User is a Steam account that has local configuration on this machine.
type User struct {
	AccountID  string // the userdata directory name (32-bit account id)
	Name       string // the persona name, if known
	Timestamp  int64  // last login time
	MostRecent bool
	ConfigPath string // path of its localconfig.vdf
}

const steamID64Base = 76561197960265728

// Users lists accounts with a localconfig.vdf, most recently used first.
func (in Install) Users() ([]User, error) {
	entries, err := os.ReadDir(filepath.Join(in.Root, "userdata"))
	if err != nil {
		return nil, err
	}
	known := in.loginUsers()
	var users []User
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "0" {
			continue
		}
		cfg := filepath.Join(in.Root, "userdata", e.Name(), "config", "localconfig.vdf")
		if _, err := os.Stat(cfg); err != nil {
			continue
		}
		u := known[e.Name()]
		u.AccountID, u.ConfigPath = e.Name(), cfg
		users = append(users, u)
	}
	sort.SliceStable(users, func(i, j int) bool {
		if users[i].MostRecent != users[j].MostRecent {
			return users[i].MostRecent
		}
		return users[i].Timestamp > users[j].Timestamp
	})
	return users, nil
}

// loginUsers reads loginusers.vdf, keyed by account id.
func (in Install) loginUsers() map[string]User {
	out := map[string]User{}
	data, err := os.ReadFile(filepath.Join(in.Root, "config", "loginusers.vdf"))
	if err != nil {
		return out
	}
	root, err := ParseVDF(data)
	if err != nil {
		return out
	}
	for _, u := range root.Child("users").childrenOrNil() {
		id64, err := strconv.ParseUint(u.Key, 10, 64)
		if err != nil || id64 < steamID64Base {
			continue
		}
		ts, _ := strconv.ParseInt(u.Str("Timestamp"), 10, 64)
		out[strconv.FormatUint(id64-steamID64Base, 10)] = User{
			Name:       u.Str("PersonaName"),
			Timestamp:  ts,
			MostRecent: u.Str("MostRecent") == "1",
		}
	}
	return out
}

func (n *Node) childrenOrNil() []*Node {
	if n == nil {
		return nil
	}
	return n.Children
}
