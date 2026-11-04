// Package catalog defines the launch options slom can build, as data. The UI controls,
// the tooltips and the command preview are all generated from these definitions.
package catalog

import (
	"fmt"
	"os/exec"
	"strconv"
)

// Kind is the type of an option's value.
type Kind int

const (
	Bool   Kind = iota // on or off
	Select             // one of Choices
	Int                // a whole number in [Min, Max]
	Text               // free text
)

// Slot says where an option goes in the command line.
type Slot int

const (
	// SlotEnv is an environment variable assignment before the command.
	SlotEnv Slot = iota
	// SlotWrapper is a wrapper program (gamemoderun, gamescope, mangohud) before %command%.
	SlotWrapper
	// SlotWrapperArg is a command-line flag of the wrapper named by Host.
	SlotWrapperArg
	// SlotConfigItem is an item of the comma-joined environment variable (Host.ConfigEnv)
	// that configures the wrapper named by Host, such as MANGOHUD_CONFIG.
	SlotConfigItem
)

// Choice is one allowed value of a Select option.
type Choice struct {
	Value string
	Label string
}

// Option is one setting. The zero Group/Support are allowed.
type Option struct {
	ID      string // stable identifier, used in stored presets
	Group   string // tab in the UI
	Section string // heading inside the group, "" when the group has none (set from sections in groups.go)
	Label   string // short human name
	Help    string // tooltip text: what it does and when to use it
	Support string // which Proton builds or hardware it works with; shown in the tooltip

	// Platform is the OS the option applies to; New fills in "linux" when empty.
	Platform string

	Kind Kind
	Slot Slot
	Name string // env variable, wrapper program, flag or config key, as rendered

	Aliases []string // other spellings accepted when importing (long flags)
	OnValue string   // Bool: the value written for env options (default "1"); config items are written bare
	Choices []Choice
	Min     int
	Max     int
	Unit    string // shown next to Int fields

	// Default is the value an Int/Text option starts with when switched on; DefaultFrom
	// names a system-derived value (see Defaults) that takes precedence when known.
	Default     string
	DefaultFrom string

	Tool      string   // program that must be installed, "" if none
	Host      string   // SlotWrapperArg/SlotConfigItem: the wrapper option this belongs to
	Requires  []string // options that must be on; they are switched on automatically
	Conflicts []string // options that cannot be on together with this one

	// Wrapper options only.
	Order      int    // position among wrappers, ascending
	Terminator string // appended after the wrapper's flags ("--" for gamescope)
	ConfigEnv  string // env variable collecting SlotConfigItem options of this wrapper
}

// Validate reports whether v is an acceptable value for the option.
func (o *Option) Validate(v string) error {
	switch o.Kind {
	case Bool:
		if v != "1" {
			return fmt.Errorf("%s: a switch is on (1) or unset", o.ID)
		}
	case Select:
		for _, c := range o.Choices {
			if c.Value == v {
				return nil
			}
		}
		return fmt.Errorf("%s: %q is not one of the choices", o.ID, v)
	case Int:
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%s: %q is not a number", o.ID, v)
		}
		if n < o.Min || n > o.Max {
			return fmt.Errorf("%s: %d is outside %d-%d", o.ID, n, o.Min, o.Max)
		}
	case Text:
		if v == "" {
			return fmt.Errorf("%s: empty", o.ID)
		}
	}
	return nil
}

// envValue is the value written for a Bool env option.
func (o *Option) envValue() string {
	if o.OnValue != "" {
		return o.OnValue
	}
	return "1"
}

// Tools says which helper programs are installed.
type Tools map[string]bool

// DetectTools looks up every tool the options need on PATH.
func DetectTools(c *Catalog) Tools {
	t := Tools{}
	for i := range c.Options {
		if name := c.Options[i].Tool; name != "" {
			_, err := exec.LookPath(name)
			t[name] = err == nil
		}
	}
	return t
}

// AllTools reports every tool of the catalog as installed (for tests and previews).
func AllTools(c *Catalog) Tools {
	t := Tools{}
	for i := range c.Options {
		if name := c.Options[i].Tool; name != "" {
			t[name] = true
		}
	}
	return t
}

// Has reports whether name is installed; "" (no tool needed) always is.
func (t Tools) Has(name string) bool { return name == "" || t[name] }

// OnValueOrOne is the value an env Bool option is written with.
func (o *Option) OnValueOrOne() string { return o.envValue() }

// Spelled reports whether flag is the option's name or one of its aliases.
func (o *Option) Spelled(flag string) bool {
	if o.Name == flag {
		return true
	}
	for _, a := range o.Aliases {
		if a == flag {
			return true
		}
	}
	return false
}
