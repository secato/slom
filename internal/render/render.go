// Package render turns a catalog.Selection into a Steam launch-options string.
package render

import (
	"regexp"
	"sort"
	"strings"

	"github.com/secato/slom/internal/catalog"
)

// Command is the placeholder Steam replaces with the game's own command.
const Command = "%command%"

var safe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./-]+$`)

// Quote returns s quoted for the shell Steam runs launch options through, only when it
// needs quoting.
func Quote(s string) string {
	if s != "" && safe.MatchString(s) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Render builds the launch-options string:
//
//	ENV=… [CONFIG_ENV=…] custom-env wrapper [wrapper-flags --] … custom-wrap %command% custom-args
//
// Options whose tool is missing, or whose requirements are not active, are left out.
func Render(c *catalog.Catalog, sel catalog.Selection, tools catalog.Tools) string {
	return strings.Join(Parts(c, sel, tools), " ")
}

// Parts is Render before joining: one element per token group.
func Parts(c *catalog.Catalog, sel catalog.Selection, tools catalog.Tools) []string {
	var parts []string
	active := func(o *catalog.Option) bool { return c.Active(sel, tools, o.ID) }

	// Environment variables, in catalog order.
	for i := range c.Options {
		o := &c.Options[i]
		if o.Slot != catalog.SlotEnv || !active(o) {
			continue
		}
		parts = append(parts, o.Name+"="+Quote(envValue(o, sel.Values[o.ID])))
	}

	// Config variables (MANGOHUD_CONFIG): the items of each active wrapper that has one.
	wrappers := wrapperOptions(c)
	for _, w := range wrappers {
		if w.ConfigEnv == "" || !active(w) {
			continue
		}
		var items []string
		for i := range c.Options {
			o := &c.Options[i]
			if o.Slot != catalog.SlotConfigItem || o.Host != w.ID || !active(o) {
				continue
			}
			if o.Kind == catalog.Bool {
				items = append(items, o.Name)
			} else {
				items = append(items, o.Name+"="+sel.Values[o.ID])
			}
		}
		items = append(items, sel.ExtraArgs[w.ID]...)
		if len(items) > 0 {
			parts = append(parts, w.ConfigEnv+"="+Quote(strings.Join(items, ",")))
		}
	}
	parts = append(parts, sel.CustomEnv...)

	// Wrappers, by Order.
	for _, w := range wrappers {
		if !active(w) {
			continue
		}
		parts = append(parts, w.Name)
		for i := range c.Options {
			o := &c.Options[i]
			if o.Slot != catalog.SlotWrapperArg || o.Host != w.ID || !active(o) {
				continue
			}
			parts = append(parts, o.Name)
			if o.Kind != catalog.Bool {
				parts = append(parts, Quote(sel.Values[o.ID]))
			}
		}
		if w.ConfigEnv == "" {
			parts = append(parts, sel.ExtraArgs[w.ID]...)
		}
		if w.Terminator != "" {
			parts = append(parts, w.Terminator)
		}
	}
	parts = append(parts, sel.CustomWrap...)
	parts = append(parts, Command)
	parts = append(parts, sel.CustomArgs...)
	return parts
}

func envValue(o *catalog.Option, v string) string {
	if o.Kind == catalog.Bool {
		if o.OnValue != "" {
			return o.OnValue
		}
		return "1"
	}
	return v
}

func wrapperOptions(c *catalog.Catalog) []*catalog.Option {
	var ws []*catalog.Option
	for i := range c.Options {
		if c.Options[i].Slot == catalog.SlotWrapper {
			ws = append(ws, &c.Options[i])
		}
	}
	sort.SliceStable(ws, func(i, j int) bool { return ws[i].Order < ws[j].Order })
	return ws
}
