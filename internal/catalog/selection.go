package catalog

import (
	"errors"
	"fmt"
	"sort"
)

// Selection is the state of the controls for one game or preset. It is plain data and
// stored as JSON.
type Selection struct {
	// Values maps option id to value; a Bool is "1". Absent means unset.
	Values map[string]string `json:"values,omitempty"`
	// Pieces of an imported launch line that the catalog does not know, verbatim and in
	// their original order. They are rendered back where they were found.
	CustomEnv  []string `json:"custom_env,omitempty"`  // NAME=value assignments before the wrappers
	CustomWrap []string `json:"custom_wrap,omitempty"` // tokens between the wrappers and %command%
	CustomArgs []string `json:"custom_args,omitempty"` // tokens after %command%
	// ExtraArgs holds unknown flags of a wrapper (gamescope) or unknown items of its
	// config variable (MangoHud), keyed by the wrapper's option id.
	ExtraArgs map[string][]string `json:"extra_args,omitempty"`
}

// Clone returns a deep copy.
func (s Selection) Clone() Selection {
	c := Selection{
		CustomEnv:  append([]string(nil), s.CustomEnv...),
		CustomWrap: append([]string(nil), s.CustomWrap...),
		CustomArgs: append([]string(nil), s.CustomArgs...),
	}
	if s.Values != nil {
		c.Values = make(map[string]string, len(s.Values))
		for k, v := range s.Values {
			c.Values[k] = v
		}
	}
	if s.ExtraArgs != nil {
		c.ExtraArgs = make(map[string][]string, len(s.ExtraArgs))
		for k, v := range s.ExtraArgs {
			c.ExtraArgs[k] = append([]string(nil), v...)
		}
	}
	return c
}

// IsEmpty reports whether nothing is selected.
func (s Selection) IsEmpty() bool {
	return len(s.Values) == 0 && len(s.CustomEnv) == 0 && len(s.CustomWrap) == 0 &&
		len(s.CustomArgs) == 0 && len(s.ExtraArgs) == 0
}

// Catalog is an indexed list of options.
type Catalog struct {
	Options []Option
	byID    map[string]*Option
}

// New indexes options; it panics on a malformed catalog (duplicate id, dangling
// reference), which is a programming error caught by tests.
func New(opts []Option) *Catalog {
	c := &Catalog{Options: opts, byID: map[string]*Option{}}
	for i := range c.Options {
		o := &c.Options[i]
		if o.Platform == "" {
			o.Platform = "linux"
		}
		if _, dup := c.byID[o.ID]; dup {
			panic("catalog: duplicate option id " + o.ID)
		}
		c.byID[o.ID] = o
	}
	for i := range c.Options {
		o := &c.Options[i]
		refs := append(append([]string{}, o.Requires...), o.Conflicts...)
		if o.Host != "" {
			refs = append(refs, o.Host)
		}
		for _, r := range refs {
			if c.byID[r] == nil {
				panic(fmt.Sprintf("catalog: %s refers to unknown option %s", o.ID, r))
			}
		}
	}
	return c
}

// Get returns the option with the id, or nil.
func (c *Catalog) Get(id string) *Option { return c.byID[id] }

// Groups lists the option groups in catalog order.
func (c *Catalog) Groups() []string {
	var out []string
	seen := map[string]bool{}
	for _, o := range c.Options {
		if !seen[o.Group] {
			seen[o.Group] = true
			out = append(out, o.Group)
		}
	}
	return out
}

// ErrConflict is returned by Set when another selected option excludes the change.
var ErrConflict = errors.New("conflicts with another option")

// Set changes one option, switching on whatever it requires and, when a wrapper is
// switched off, clearing the options that belong to it. v == "" unsets. The returned
// notes describe automatic changes, for the UI to show.
func (c *Catalog) Set(sel *Selection, id, v string) ([]string, error) {
	o := c.Get(id)
	if o == nil {
		return nil, fmt.Errorf("unknown option %q", id)
	}
	if sel.Values == nil {
		sel.Values = map[string]string{}
	}
	if v == "" {
		var notes []string
		delete(sel.Values, id)
		for i := range c.Options {
			d := &c.Options[i]
			if _, on := sel.Values[d.ID]; on && c.dependsOn(d, id) {
				n, _ := c.Set(sel, d.ID, "")
				notes = append(notes, fmt.Sprintf("%s turned off because it needs %s", d.Label, o.Label))
				notes = append(notes, n...)
			}
		}
		if len(sel.Values) == 0 {
			sel.Values = nil
		}
		return notes, nil
	}
	if err := o.Validate(v); err != nil {
		return nil, err
	}
	if why := c.conflictWith(sel, o); why != "" {
		return nil, fmt.Errorf("%w: %s", ErrConflict, why)
	}
	var notes []string
	sel.Values[id] = v
	needs := o.Requires
	if o.Host != "" {
		needs = append([]string{o.Host}, needs...)
	}
	for _, r := range needs {
		if _, on := sel.Values[r]; !on {
			req := c.Get(r)
			n, err := c.Set(sel, r, c.onValue(req))
			if err != nil {
				delete(sel.Values, id)
				return nil, err
			}
			notes = append(notes, fmt.Sprintf("%s turned on because %s needs it", req.Label, o.Label))
			notes = append(notes, n...)
		}
	}
	return notes, nil
}

// Defaults are values derived from the system (the display's size and refresh rate),
// keyed by Option.DefaultFrom.
type Defaults map[string]int

// onValue is the value that switches the option on when nothing else is known.
func (c *Catalog) onValue(o *Option) string { return c.Initial(o, nil) }

// Initial is the value an option takes when the user switches it on: the system-derived
// default if known (clamped to the option's range), else its Default, else the first
// choice or the minimum.
func (c *Catalog) Initial(o *Option, d Defaults) string {
	switch o.Kind {
	case Bool:
		return "1"
	case Select:
		if o.Default != "" {
			return o.Default
		}
		return o.Choices[0].Value
	case Int:
		if v, ok := d[o.DefaultFrom]; ok && o.DefaultFrom != "" {
			return fmt.Sprint(min(max(v, o.Min), o.Max))
		}
		if o.Default != "" {
			return o.Default
		}
		return fmt.Sprint(o.Min)
	}
	return o.Default
}

// dependsOn reports whether d requires id, directly or through its host or requirements.
func (c *Catalog) dependsOn(d *Option, id string) bool {
	if d.Host == id {
		return true
	}
	for _, r := range d.Requires {
		if r == id {
			return true
		}
	}
	return false
}

func (c *Catalog) conflictWith(sel *Selection, o *Option) string {
	for _, x := range o.Conflicts {
		if _, on := sel.Values[x]; on {
			return c.Get(x).Label + " is on"
		}
	}
	for id := range sel.Values {
		for _, x := range c.Get(id).Conflicts {
			if x == o.ID {
				return c.Get(id).Label + " is on"
			}
		}
	}
	return ""
}

// Reason explains why the control of option id cannot be used now, or "" if it can: the
// tool is missing, or a conflicting option is on.
func (c *Catalog) Reason(sel Selection, tools Tools, id string) string {
	o := c.Get(id)
	if o == nil {
		return ""
	}
	if !tools.Has(o.Tool) {
		return o.Tool + " is not installed; install it to use this option"
	}
	if _, on := sel.Values[id]; on {
		return ""
	}
	if why := c.conflictWith(&sel, o); why != "" {
		return "Cannot be combined: " + why
	}
	return ""
}

// Active reports whether the option is set and will appear in the command line: its tool
// is installed and everything it requires is active.
func (c *Catalog) Active(sel Selection, tools Tools, id string) bool {
	o := c.Get(id)
	if o == nil {
		return false
	}
	if _, on := sel.Values[id]; !on || !tools.Has(o.Tool) {
		return false
	}
	if o.Host != "" && !c.Active(sel, tools, o.Host) {
		return false
	}
	for _, r := range o.Requires {
		if !c.Active(sel, tools, r) {
			return false
		}
	}
	return true
}

// SetIDs lists the ids of selected options in catalog order.
func (c *Catalog) SetIDs(sel Selection) []string {
	var ids []string
	for id := range sel.Values {
		ids = append(ids, id)
	}
	idx := map[string]int{}
	for i := range c.Options {
		idx[c.Options[i].ID] = i
	}
	sort.Slice(ids, func(i, j int) bool { return idx[ids[i]] < idx[ids[j]] })
	return ids
}

// IsEmptyExceptArgs reports whether nothing but CustomArgs is set.
func (s Selection) IsEmptyExceptArgs() bool {
	s.CustomArgs = nil
	return s.IsEmpty()
}

// ForceSet sets an option like Set, but resolves conflicts by switching the conflicting
// options off instead of failing.
func (c *Catalog) ForceSet(sel *Selection, id, v string) ([]string, error) {
	o := c.Get(id)
	if o == nil {
		return nil, fmt.Errorf("unknown option %q", id)
	}
	var notes []string
	if v != "" {
		for _, x := range c.conflictIDs(sel, o) {
			n, _ := c.Set(sel, x, "")
			notes = append(notes, n...)
		}
	}
	n, err := c.Set(sel, id, v)
	return append(notes, n...), err
}

func (c *Catalog) conflictIDs(sel *Selection, o *Option) []string {
	var out []string
	for id := range sel.Values {
		if id == o.ID {
			continue
		}
		for _, x := range o.Conflicts {
			if x == id {
				out = append(out, id)
			}
		}
	}
	sort.Strings(out)
	return out
}

// Normalize drops values that cannot take effect: options whose wrapper or requirements
// are not set, and invalid values. It does not look at which tools are installed.
func (c *Catalog) Normalize(sel *Selection) {
	for changed := true; changed; {
		changed = false
		for id, v := range sel.Values {
			o := c.Get(id)
			bad := o == nil || o.Validate(v) != nil
			if !bad && o.Host != "" {
				_, bad = sel.Values[o.Host]
				bad = !bad
			}
			for _, r := range o.requires() {
				if _, on := sel.Values[r]; !on {
					bad = true
				}
			}
			if bad {
				delete(sel.Values, id)
				changed = true
			}
		}
	}
	if len(sel.Values) == 0 {
		sel.Values = nil
	}
}

func (o *Option) requires() []string {
	if o == nil {
		return nil
	}
	return o.Requires
}
