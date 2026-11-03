// Package steam finds Steam installs, libraries, users and installed games, reads and
// surgically edits localconfig.vdf, and controls the Steam process.
package steam

import (
	"fmt"
	"strings"
)

// Node is one entry of a Valve text KeyValues file. It is either a string value or a
// block of children. Byte offsets point into the parsed source so callers can edit the
// source in place without re-serializing it.
type Node struct {
	Key      string
	Value    string // unescaped; empty for blocks
	IsBlock  bool
	Children []*Node

	KeyStart, KeyEnd int // span of the key token, quotes included
	ValStart, ValEnd int // span of the value token, quotes included (strings only)
	Open, Close      int // offsets of '{' and '}' (blocks only)
}

// Child returns the first child with the given key, compared case-insensitively
// (Steam treats keys that way).
func (n *Node) Child(key string) *Node {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			return c
		}
	}
	return nil
}

// Path follows keys from n, returning nil if any is missing.
func (n *Node) Path(keys ...string) *Node {
	for _, k := range keys {
		n = n.Child(k)
		if n == nil {
			return nil
		}
	}
	return n
}

// Str returns the string value of the child key, or "".
func (n *Node) Str(key string) string {
	if c := n.Child(key); c != nil && !c.IsBlock {
		return c.Value
	}
	return ""
}

// ParseVDF parses src into a synthetic root whose children are the top-level keys.
func ParseVDF(src []byte) (*Node, error) {
	p := &vdfParser{src: src}
	root := &Node{IsBlock: true, Open: -1, Close: len(src)}
	kids, err := p.block(false)
	if err != nil {
		return nil, err
	}
	root.Children = kids
	return root, nil
}

type vdfToken struct {
	kind       byte // 's' string, '{', '}', 0 at EOF
	text       string
	start, end int
}

type vdfParser struct {
	src []byte
	pos int
}

func (p *vdfParser) next() (vdfToken, error) {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			p.pos++
		case c == '/' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '/':
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
		case c == '{' || c == '}':
			t := vdfToken{kind: c, start: p.pos, end: p.pos + 1}
			p.pos++
			return t, nil
		case c == '"':
			return p.quoted()
		default:
			start := p.pos
			for p.pos < len(p.src) {
				c := p.src[p.pos]
				if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '{' || c == '}' || c == '"' {
					break
				}
				p.pos++
			}
			return vdfToken{kind: 's', text: string(p.src[start:p.pos]), start: start, end: p.pos}, nil
		}
	}
	return vdfToken{start: p.pos, end: p.pos}, nil
}

func (p *vdfParser) quoted() (vdfToken, error) {
	start := p.pos
	p.pos++
	var b strings.Builder
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		switch c {
		case '"':
			p.pos++
			return vdfToken{kind: 's', text: b.String(), start: start, end: p.pos}, nil
		case '\\':
			if p.pos+1 >= len(p.src) {
				return vdfToken{}, fmt.Errorf("vdf: dangling escape at byte %d", p.pos)
			}
			p.pos++
			switch e := p.src[p.pos]; e {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case '\\', '"':
				b.WriteByte(e)
			default: // unknown escape: keep both characters as Steam does
				b.WriteByte('\\')
				b.WriteByte(e)
			}
			p.pos++
		default:
			b.WriteByte(c)
			p.pos++
		}
	}
	return vdfToken{}, fmt.Errorf("vdf: unterminated string at byte %d", start)
}

// block parses entries until the closing brace (or EOF for the top level).
func (p *vdfParser) block(nested bool) ([]*Node, error) {
	var out []*Node
	for {
		k, err := p.next()
		if err != nil {
			return nil, err
		}
		switch k.kind {
		case 0:
			if nested {
				return nil, fmt.Errorf("vdf: unexpected end of file, missing '}'")
			}
			return out, nil
		case '}':
			if !nested {
				return nil, fmt.Errorf("vdf: unexpected '}' at byte %d", k.start)
			}
			return out, nil
		case '{':
			return nil, fmt.Errorf("vdf: unexpected '{' at byte %d", k.start)
		}
		n := &Node{Key: k.text, KeyStart: k.start, KeyEnd: k.end}
		v, err := p.next()
		if err != nil {
			return nil, err
		}
		switch v.kind {
		case 's':
			n.Value, n.ValStart, n.ValEnd = v.text, v.start, v.end
		case '{':
			n.IsBlock, n.Open = true, v.start
			if n.Children, err = p.block(true); err != nil {
				return nil, err
			}
			n.Close = p.pos - 1
		default:
			return nil, fmt.Errorf("vdf: key %q has no value", k.text)
		}
		out = append(out, n)
	}
}

// EscapeVDF escapes s for use inside a quoted VDF string.
func EscapeVDF(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`)
	return r.Replace(s)
}
