// Package store keeps slom's own data: reusable presets and the per-game assignments
// that link a game to a preset plus overrides. Plain JSON under the XDG config dir.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/secato/slom/internal/catalog"
)

// NoPresetLabel is what the UI shows for a game without a preset, so no preset may use it.
const NoPresetLabel = "No preset"

// Preset is a named, reusable selection.
type Preset struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Selection catalog.Selection `json:"selection"`
}

// Assignment is a game's configuration: an optional preset plus what differs from it.
type Assignment struct {
	PresetID string `json:"preset,omitempty"`
	// Overrides are values the game sets on top of (or instead of) the preset's.
	Overrides map[string]string `json:"overrides,omitempty"`
	// Off lists options the preset turns on that this game turns off.
	Off []string `json:"off,omitempty"`
	// Custom holds the game's own unrecognized launch-line pieces (see catalog.Selection).
	Custom catalog.Selection `json:"custom,omitempty"`
}

// Store is safe for concurrent use.
type Store struct {
	mu          sync.Mutex
	dir         string
	presets     []Preset
	assignments map[string]Assignment
}

// DefaultDir is $XDG_CONFIG_HOME/slom, or ~/.config/slom.
func DefaultDir() (string, error) {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "slom"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "slom"), nil
}

// DefaultBackupDir is $XDG_STATE_HOME/slom/backups, or ~/.local/state/slom/backups.
func DefaultBackupDir() (string, error) {
	if x := os.Getenv("XDG_STATE_HOME"); x != "" {
		return filepath.Join(x, "slom", "backups"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "slom", "backups"), nil
}

// Open loads the store in dir; missing files mean an empty store. A file that cannot be
// parsed is an error (and is left untouched) rather than silently discarded.
func Open(dir string) (*Store, error) {
	s := &Store{dir: dir, assignments: map[string]Assignment{}}
	if err := readJSON(filepath.Join(dir, "presets.json"), &s.presets); err != nil {
		return nil, err
	}
	if err := readJSON(filepath.Join(dir, "assignments.json"), &s.assignments); err != nil {
		return nil, err
	}
	if s.assignments == nil {
		s.assignments = map[string]Assignment{}
	}
	return s, nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("%s: %w (fix or remove the file)", path, err)
	}
	return nil
}

func (s *Store) writeJSON(name string, v any) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.dir, name+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(s.dir, name))
}

// Presets returns a copy of the presets, sorted by name.
func (s *Store) Presets() []Preset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Preset, len(s.presets))
	for i, p := range s.presets {
		out[i] = Preset{ID: p.ID, Name: p.Name, Selection: p.Selection.Clone()}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

// Preset returns the preset with the id.
func (s *Store) Preset(id string) (Preset, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.preset(id)
}

func (s *Store) preset(id string) (Preset, bool) {
	for _, p := range s.presets {
		if p.ID == id {
			return Preset{ID: p.ID, Name: p.Name, Selection: p.Selection.Clone()}, true
		}
	}
	return Preset{}, false
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// SavePreset creates the preset (when p.ID is empty, an id is made from the name) or
// replaces the one with that id, and returns it.
func (s *Store) SavePreset(p Preset) (Preset, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return Preset{}, errors.New("a preset needs a name")
	}
	if strings.EqualFold(p.Name, NoPresetLabel) {
		return Preset{}, fmt.Errorf("%q is reserved; pick another name", NoPresetLabel)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, q := range s.presets {
		if q.ID != p.ID && strings.EqualFold(q.Name, p.Name) {
			return Preset{}, fmt.Errorf("a preset named %q already exists", q.Name)
		}
	}
	p.Selection = p.Selection.Clone()
	// A preset is a set of options; verbatim pieces of one game's launch line do not belong in it.
	p.Selection.CustomEnv, p.Selection.CustomWrap, p.Selection.CustomArgs, p.Selection.ExtraArgs = nil, nil, nil, nil
	if p.ID == "" {
		base := strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(p.Name), "-"), "-")
		if base == "" {
			base = "preset"
		}
		p.ID = base
		for n := 2; s.hasPreset(p.ID); n++ {
			p.ID = fmt.Sprintf("%s-%d", base, n)
		}
	}
	replaced := false
	for i := range s.presets {
		if s.presets[i].ID == p.ID {
			s.presets[i], replaced = p, true
		}
	}
	if !replaced {
		s.presets = append(s.presets, p)
	}
	return p, s.writeJSON("presets.json", s.presets)
}

func (s *Store) hasPreset(id string) bool { _, ok := s.preset(id); return ok }

// DeletePreset removes a preset. Games that used it keep their resulting configuration:
// the preset's values are folded into their overrides first.
func (s *Store) DeletePreset(c *catalog.Catalog, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.preset(id)
	if !ok {
		return nil
	}
	for app, a := range s.assignments {
		if a.PresetID == id {
			eff := Effective(c, &p, a)
			s.assignments[app] = Derive(c, nil, eff)
		}
	}
	kept := s.presets[:0:0]
	for _, q := range s.presets {
		if q.ID != id {
			kept = append(kept, q)
		}
	}
	s.presets = kept
	if err := s.writeJSON("assignments.json", s.assignments); err != nil {
		return err
	}
	return s.writeJSON("presets.json", s.presets)
}

// Assignment returns the game's assignment.
func (s *Store) Assignment(appID string) (Assignment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, ok := s.assignments[appID]
	return a, ok
}

// SetAssignment stores the game's assignment.
func (s *Store) SetAssignment(appID string, a Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.assignments[appID] = a
	return s.writeJSON("assignments.json", s.assignments)
}

// ClearAssignment forgets the game's assignment.
func (s *Store) ClearAssignment(appID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.assignments, appID)
	return s.writeJSON("assignments.json", s.assignments)
}

// GamesUsing lists app ids linked to the preset, sorted.
func (s *Store) GamesUsing(presetID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var ids []string
	for app, a := range s.assignments {
		if a.PresetID == presetID {
			ids = append(ids, app)
		}
	}
	sort.Strings(ids)
	return ids
}

// Effective is the selection of a game: the preset's values, minus Off, with the
// overrides on top, and the game's own custom pieces.
func Effective(c *catalog.Catalog, p *Preset, a Assignment) catalog.Selection {
	var sel catalog.Selection
	if p != nil {
		sel = p.Selection.Clone()
		sel.CustomEnv, sel.CustomWrap, sel.CustomArgs, sel.ExtraArgs = nil, nil, nil, nil
	}
	for _, id := range a.Off {
		c.Set(&sel, id, "")
	}
	for _, o := range c.Options { // catalog order keeps this deterministic
		if v, ok := a.Overrides[o.ID]; ok {
			c.ForceSet(&sel, o.ID, v)
		}
	}
	cu := a.Custom.Clone()
	sel.CustomEnv, sel.CustomWrap, sel.CustomArgs, sel.ExtraArgs = cu.CustomEnv, cu.CustomWrap, cu.CustomArgs, cu.ExtraArgs
	c.Normalize(&sel)
	return sel
}

// Derive is the inverse of Effective: the assignment that, on top of preset p (nil for
// none), yields sel.
func Derive(c *catalog.Catalog, p *Preset, sel catalog.Selection) Assignment {
	a := Assignment{Custom: catalog.Selection{
		CustomEnv: sel.CustomEnv, CustomWrap: sel.CustomWrap, CustomArgs: sel.CustomArgs, ExtraArgs: sel.ExtraArgs,
	}.Clone()}
	if p != nil {
		a.PresetID = p.ID
	}
	var base map[string]string
	if p != nil {
		base = p.Selection.Values
	}
	for _, o := range c.Options {
		want, has := sel.Values[o.ID]
		have, inBase := base[o.ID]
		switch {
		case has && (!inBase || want != have):
			if a.Overrides == nil {
				a.Overrides = map[string]string{}
			}
			a.Overrides[o.ID] = want
		case !has && inBase:
			a.Off = append(a.Off, o.ID)
		}
	}
	return a
}

// Overridden lists the option ids the game changes relative to its preset.
func (a Assignment) Overridden() map[string]bool {
	out := map[string]bool{}
	for id := range a.Overrides {
		out[id] = true
	}
	for _, id := range a.Off {
		out[id] = true
	}
	return out
}
