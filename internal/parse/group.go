package parse

import "strings"

func isFlag(value string) bool { return len(value) > 1 && value[0] == '-' }

// Group joins verbatim tokens into chips, one per thing the user would call one setting:
// a program with its flags and their values ("taskset -c 0-3"), or a flag with its value
// ("--weird-flag 12"). A "--" ends a group. Each chip is its tokens' raw text joined by
// spaces, so Ungroup restores the exact tokens.
func Group(raws []string) []string {
	var chips []string
	var cur []string
	var first, last string // values of the group's first and last tokens
	flush := func() {
		if len(cur) > 0 {
			chips = append(chips, strings.Join(cur, " "))
			cur = nil
		}
	}
	for _, raw := range raws {
		toks := Tokenize(raw)
		value := raw
		if len(toks) == 1 {
			value = toks[0].Value
		}
		start := len(cur) == 0 || last == "--"
		if !start {
			prevFlag := isFlag(last)
			if isFlag(first) { // a run of flags: a word belongs only right after its flag
				start = isFlag(value) || !prevFlag
			} else { // a program: its flags and their values belong to it, another word starts a new one
				start = !isFlag(value) && !prevFlag && !assignRe.MatchString(value) // NAME=value belongs to "env"
			}
		}
		if start {
			flush()
			first = value
		}
		cur = append(cur, raw)
		last = value
	}
	flush()
	return chips
}

// Ungroup is the inverse of Group: it splits each chip back into verbatim tokens. A chip
// the user edited may hold several tokens, or none (it then disappears).
func Ungroup(chips []string) []string {
	var out []string
	for _, chip := range chips {
		for _, t := range Tokenize(chip) {
			out = append(out, t.Raw)
		}
	}
	return out
}

// Units splits a launch line into the pieces a person would call one change: an
// environment assignment, a program, a flag together with its value, "--", %command%. It is
// for comparing two lines (see Diff), not for rebuilding one.
func Units(line string) []string {
	var out []string
	toks := Tokenize(line)
	for i := 0; i < len(toks); i++ {
		v := toks[i].Raw
		if strings.HasPrefix(v, "-") && v != "--" && i+1 < len(toks) {
			if n := toks[i+1].Raw; !strings.HasPrefix(n, "-") && n != "%command%" {
				v += " " + n
				i++
			}
		}
		out = append(out, v)
	}
	return out
}

// Diff says what going from the old line to the new one adds and removes, by unit: order
// does not count, a unit present in both is not mentioned, and a repeated unit counts as
// many times as it is repeated.
func Diff(oldLine, newLine string) (added, removed []string) {
	have := map[string]int{}
	for _, u := range Units(oldLine) {
		have[u]++
	}
	for _, u := range Units(newLine) {
		if have[u] > 0 {
			have[u]--
		} else {
			added = append(added, u)
		}
	}
	for _, u := range Units(oldLine) {
		if have[u] > 0 {
			have[u]--
			removed = append(removed, u)
		}
	}
	return added, removed
}
