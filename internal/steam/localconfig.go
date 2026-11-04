package steam

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ErrSteamRunning is returned when a write is attempted while Steam is running; Steam
// rewrites localconfig.vdf from memory and would discard the change.
var ErrSteamRunning = errors.New("steam is running")

var appsPath = []string{"UserLocalConfigStore", "Software", "Valve", "Steam", "apps"}

const launchOptionsKey = "LaunchOptions"

// LaunchOptions returns the launch options of every app that has some, keyed by app id.
func LaunchOptions(src []byte) (map[string]string, error) {
	root, err := ParseVDF(src)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	apps := root.Path(appsPath...)
	if apps == nil {
		return out, nil
	}
	for _, app := range apps.Children {
		if v := app.Str(launchOptionsKey); v != "" {
			out[app.Key] = v
		}
	}
	return out, nil
}

// SetLaunchOptions returns src with the launch options of appID set to value. Only the
// affected bytes change; an empty value removes the key. It is a pure function.
func SetLaunchOptions(src []byte, appID, value string) ([]byte, error) {
	root, err := ParseVDF(src)
	if err != nil {
		return nil, err
	}
	apps := root.Path(appsPath...)
	if apps == nil {
		return nil, fmt.Errorf("localconfig: no %s block", strings.Join(appsPath, "/"))
	}
	app := apps.Child(appID)
	if app != nil && !app.IsBlock {
		return nil, fmt.Errorf("localconfig: app %s is not a block", appID)
	}
	var lo *Node
	if app != nil {
		if c := app.Child(launchOptionsKey); c != nil && !c.IsBlock {
			lo = c
		}
	}

	switch {
	case lo != nil && value == "":
		start, end := lineSpan(src, lo.KeyStart, lo.ValEnd)
		return splice(src, start, end, ""), nil
	case lo != nil:
		return splice(src, lo.ValStart, lo.ValEnd, `"`+EscapeVDF(value)+`"`), nil
	case value == "":
		return src, nil
	case app != nil:
		ind := childIndent(src, app)
		line := ind + `"` + launchOptionsKey + `"` + "\t\t" + `"` + EscapeVDF(value) + `"` + "\n"
		return insertBeforeClose(src, app, line), nil
	default:
		ind := childIndent(src, apps)
		var b strings.Builder
		fmt.Fprintf(&b, "%s\"%s\"\n%s{\n", ind, appID, ind)
		fmt.Fprintf(&b, "%s\t\"%s\"\t\t\"%s\"\n", ind, launchOptionsKey, EscapeVDF(value))
		fmt.Fprintf(&b, "%s}\n", ind)
		return insertBeforeClose(src, apps, b.String()), nil
	}
}

func splice(src []byte, start, end int, repl string) []byte {
	out := make([]byte, 0, len(src)-(end-start)+len(repl))
	out = append(out, src[:start]...)
	out = append(out, repl...)
	return append(out, src[end:]...)
}

// lineSpan widens [start,end) to whole lines when only whitespace shares them.
func lineSpan(src []byte, start, end int) (int, int) {
	ls := start
	for ls > 0 && src[ls-1] != '\n' {
		ls--
	}
	for i := ls; i < start; i++ {
		if src[i] != ' ' && src[i] != '\t' {
			return start, end
		}
	}
	le := end
	for le < len(src) && src[le] != '\n' {
		if src[le] != ' ' && src[le] != '\t' && src[le] != '\r' {
			return start, end
		}
		le++
	}
	if le < len(src) {
		le++ // the newline
	}
	return ls, le
}

// childIndent is the indentation used by the children of block n.
func childIndent(src []byte, n *Node) string {
	if len(n.Children) > 0 {
		return indentOf(src, n.Children[0].KeyStart)
	}
	return indentOf(src, n.KeyStart) + "\t"
}

func indentOf(src []byte, pos int) string {
	ls := pos
	for ls > 0 && src[ls-1] != '\n' {
		ls--
	}
	i := ls
	for i < pos && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	return string(src[ls:i])
}

// insertBeforeClose adds text as the last lines of block n.
func insertBeforeClose(src []byte, n *Node, text string) []byte {
	ls := n.Close
	for ls > 0 && src[ls-1] != '\n' {
		ls--
	}
	for i := ls; i < n.Close; i++ {
		if src[i] != ' ' && src[i] != '\t' {
			// '}' shares its line with other content: break the line before it.
			return splice(src, n.Close, n.Close, "\n"+text)
		}
	}
	return splice(src, ls, ls, text)
}

// WriteOptions controls WriteLaunchOptions.
type WriteOptions struct {
	BackupDir    string      // where timestamped copies go; "" disables backups
	Keep         int         // backups to keep; <=0 keeps all
	SteamRunning func() bool // defaults to Running
	Now          func() time.Time
}

// WriteLaunchOptions sets the launch options of appID in the localconfig.vdf at path.
// It refuses while Steam runs, backs the file up first, writes atomically, re-reads the
// result to verify it, and restores the backup if verification fails.
func WriteLaunchOptions(path, appID, value string, o WriteOptions) error {
	return WriteManyLaunchOptions(path, map[string]string{appID: value}, o)
}

// WriteManyLaunchOptions does the same for several games at once, with one backup and one
// write: the file is either all changed or, if anything fails, as it was.
func WriteManyLaunchOptions(path string, values map[string]string, o WriteOptions) error {
	running := o.SteamRunning
	if running == nil {
		running = Running
	}
	if running() {
		return ErrSteamRunning
	}
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := src
	for _, id := range ids {
		if out, err = SetLaunchOptions(out, id, values[id]); err != nil {
			return err
		}
	}
	if string(out) == string(src) {
		return nil
	}
	backup, err := backupFile(path, src, o)
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}
	if err := atomicWrite(path, out); err != nil {
		return err
	}
	if verr := verifyLaunchOptions(path, values); verr != nil {
		if backup != "" {
			if rerr := atomicWrite(path, src); rerr != nil {
				return fmt.Errorf("verification failed (%v) and restore failed: %w", verr, rerr)
			}
		}
		return fmt.Errorf("verification failed, original restored: %w", verr)
	}
	return nil
}

func verifyLaunchOptions(path string, want map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	got, err := LaunchOptions(data)
	if err != nil {
		return err
	}
	for id, w := range want {
		if got[id] != w {
			return fmt.Errorf("app %s: read back %q, want %q", id, got[id], w)
		}
	}
	return nil
}

func backupFile(path string, src []byte, o WriteOptions) (string, error) {
	if o.BackupDir == "" {
		return "", nil
	}
	if err := os.MkdirAll(o.BackupDir, 0o755); err != nil {
		return "", err
	}
	now := time.Now
	if o.Now != nil {
		now = o.Now
	}
	name := fmt.Sprintf("%s.%s.bak", filepath.Base(path), now().Format("20060102-150405.000"))
	dst := filepath.Join(o.BackupDir, name)
	if err := os.WriteFile(dst, src, 0o600); err != nil {
		return "", err
	}
	if o.Keep > 0 {
		pruneBackups(o.BackupDir, filepath.Base(path), o.Keep)
	}
	return dst, nil
}

func pruneBackups(dir, base string, keep int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), base+".") && strings.HasSuffix(e.Name(), ".bak") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names) // timestamp in the name sorts oldest first
	for len(names) > keep {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}

func atomicWrite(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if st, err := os.Stat(path); err == nil {
		mode = st.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".slom-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
