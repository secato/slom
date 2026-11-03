// Package parse turns an existing Steam launch-options string into a catalog.Selection,
// the inverse of render. What the catalog does not recognize is kept verbatim, in
// position, in the selection's custom fields: nothing is dropped.
package parse

import (
	"regexp"
	"strings"

	"github.com/secato/slom/internal/catalog"
)

// Token is one shell word: Raw as written (quotes included), Value after unquoting.
type Token struct{ Raw, Value string }

// Tokenize splits a launch-options line like a POSIX shell: whitespace separates words;
// single quotes, double quotes and backslashes group and escape. An unterminated quote
// runs to the end of the line.
func Tokenize(line string) []Token {
	var toks []Token
	var raw, val strings.Builder
	in := false
	flush := func() {
		if in {
			toks = append(toks, Token{raw.String(), val.String()})
			raw.Reset()
			val.Reset()
			in = false
		}
	}
	rs := []rune(line)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			flush()
		case c == '\'':
			in = true
			raw.WriteRune(c)
			for i++; i < len(rs) && rs[i] != '\''; i++ {
				raw.WriteRune(rs[i])
				val.WriteRune(rs[i])
			}
			if i < len(rs) {
				raw.WriteRune('\'')
			}
		case c == '"':
			in = true
			raw.WriteRune(c)
			for i++; i < len(rs) && rs[i] != '"'; i++ {
				if rs[i] == '\\' && i+1 < len(rs) && strings.ContainsRune(`"\$`+"`", rs[i+1]) {
					raw.WriteRune(rs[i])
					i++
				}
				raw.WriteRune(rs[i])
				val.WriteRune(rs[i])
			}
			if i < len(rs) {
				raw.WriteRune('"')
			}
		case c == '\\' && i+1 < len(rs):
			in = true
			raw.WriteRune(c)
			i++
			raw.WriteRune(rs[i])
			val.WriteRune(rs[i])
		default:
			in = true
			raw.WriteRune(c)
			val.WriteRune(c)
		}
	}
	flush()
	return toks
}

var assignRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

// Parse reads a launch-options line into a Selection.
func Parse(c *catalog.Catalog, line string) catalog.Selection {
	p := &parser{c: c, sel: catalog.Selection{}}
	toks := Tokenize(line)
	cmd := -1
	for i, t := range toks {
		if t.Value == "%command%" {
			cmd = i
			break
		}
	}
	pre, post := toks, []Token(nil)
	if cmd >= 0 {
		pre, post = toks[:cmd], toks[cmd+1:]
	}
	for _, t := range post {
		p.sel.CustomArgs = append(p.sel.CustomArgs, t.Raw)
	}

	i := 0
	for i < len(pre) && assignRe.MatchString(pre[i].Value) {
		p.env(pre[i])
		i++
	}
	for i < len(pre) {
		i = p.wrapper(pre, i)
	}
	p.fixOrphans()
	if len(p.sel.Values) == 0 {
		p.sel.Values = nil
	}
	return p.sel
}

type parser struct {
	c   *catalog.Catalog
	sel catalog.Selection
}

func (p *parser) set(id, v string) {
	if p.sel.Values == nil {
		p.sel.Values = map[string]string{}
	}
	p.sel.Values[id] = v
}

func (p *parser) extra(host string, raws ...string) {
	if p.sel.ExtraArgs == nil {
		p.sel.ExtraArgs = map[string][]string{}
	}
	p.sel.ExtraArgs[host] = append(p.sel.ExtraArgs[host], raws...)
}

// env handles one leading NAME=value token.
func (p *parser) env(t Token) {
	m := assignRe.FindStringSubmatch(t.Value)
	name, val := m[1], m[2]
	for i := range p.c.Options {
		o := &p.c.Options[i]
		switch {
		case o.Slot == catalog.SlotEnv && o.Name == name:
			if o.Kind == catalog.Bool {
				if val == o.OnValueOrOne() {
					p.set(o.ID, "1")
					return
				}
			} else if o.Validate(val) == nil {
				p.set(o.ID, val)
				return
			}
		case o.Slot == catalog.SlotWrapper && o.ConfigEnv == name:
			p.configItems(o, val)
			return
		}
	}
	p.sel.CustomEnv = append(p.sel.CustomEnv, t.Raw)
}

// configItems splits a MANGOHUD_CONFIG-style value into known options and extras.
func (p *parser) configItems(host *catalog.Option, val string) {
	for _, item := range strings.Split(val, ",") {
		if item == "" {
			continue
		}
		key, v, hasV := strings.Cut(item, "=")
		var matched *catalog.Option
		for i := range p.c.Options {
			o := &p.c.Options[i]
			if o.Slot == catalog.SlotConfigItem && o.Host == host.ID && o.Name == key {
				matched = o
			}
		}
		switch {
		case matched == nil:
		case matched.Kind == catalog.Bool && !hasV:
			p.set(matched.ID, "1")
			continue
		case matched.Kind != catalog.Bool && hasV && matched.Validate(v) == nil:
			p.set(matched.ID, v)
			continue
		}
		p.extra(host.ID, item)
	}
}

// wrapper handles the token at pre[i] and returns the index of the next one.
func (p *parser) wrapper(pre []Token, i int) int {
	t := pre[i]
	for k := range p.c.Options {
		w := &p.c.Options[k]
		if w.Slot != catalog.SlotWrapper || w.Name != t.Value {
			continue
		}
		p.set(w.ID, "1")
		i++
		if w.ConfigEnv != "" || !p.hasArgs(w) {
			return i
		}
		for i < len(pre) {
			if w.Terminator != "" && pre[i].Value == w.Terminator {
				return i + 1
			}
			i = p.arg(w, pre, i)
		}
		return i
	}
	p.sel.CustomWrap = append(p.sel.CustomWrap, t.Raw)
	return i + 1
}

func (p *parser) hasArgs(w *catalog.Option) bool {
	for i := range p.c.Options {
		if p.c.Options[i].Slot == catalog.SlotWrapperArg && p.c.Options[i].Host == w.ID {
			return true
		}
	}
	return false
}

// arg reads one flag (and its value) of wrapper w.
func (p *parser) arg(w *catalog.Option, pre []Token, i int) int {
	t := pre[i]
	flag, inline, hasInline := t.Value, "", false
	if strings.HasPrefix(flag, "--") {
		flag, inline, hasInline = strings.Cut(flag, "=")
	}
	for k := range p.c.Options {
		o := &p.c.Options[k]
		if o.Slot != catalog.SlotWrapperArg || o.Host != w.ID || !o.Spelled(flag) {
			continue
		}
		if o.Kind == catalog.Bool {
			if !hasInline {
				p.set(o.ID, "1")
				return i + 1
			}
			break
		}
		if hasInline {
			if o.Validate(inline) == nil {
				p.set(o.ID, inline)
				return i + 1
			}
			break
		}
		if i+1 < len(pre) && o.Validate(pre[i+1].Value) == nil {
			p.set(o.ID, pre[i+1].Value)
			return i + 2
		}
		break
	}
	// Unknown (or invalid) flag: keep it, with its value if it seems to have one.
	p.extra(w.ID, t.Raw)
	if strings.HasPrefix(t.Value, "-") && !hasInline && i+1 < len(pre) &&
		!strings.HasPrefix(pre[i+1].Value, "-") && pre[i+1].Value != w.Terminator {
		p.extra(w.ID, pre[i+1].Raw)
		return i + 2
	}
	return i + 1
}

// fixOrphans handles config items (MANGOHUD_CONFIG) found without their wrapper: with no
// wrapper to host them they would vanish on render, so they go back to a custom env token.
func (p *parser) fixOrphans() {
	for i := range p.c.Options {
		w := &p.c.Options[i]
		if w.Slot != catalog.SlotWrapper || w.ConfigEnv == "" {
			continue
		}
		if _, on := p.sel.Values[w.ID]; on {
			continue
		}
		var items []string
		for k := range p.c.Options {
			o := &p.c.Options[k]
			if o.Slot == catalog.SlotConfigItem && o.Host == w.ID {
				if v, ok := p.sel.Values[o.ID]; ok {
					if o.Kind == catalog.Bool {
						items = append(items, o.Name)
					} else {
						items = append(items, o.Name+"="+v)
					}
					delete(p.sel.Values, o.ID)
				}
			}
		}
		items = append(items, p.sel.ExtraArgs[w.ID]...)
		delete(p.sel.ExtraArgs, w.ID)
		if len(p.sel.ExtraArgs) == 0 {
			p.sel.ExtraArgs = nil
		}
		if len(items) > 0 {
			p.sel.CustomEnv = append(p.sel.CustomEnv, w.ConfigEnv+"="+strings.Join(items, ","))
		}
	}
}
