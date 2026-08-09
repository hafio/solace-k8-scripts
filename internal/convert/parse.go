// Package convert turns a legacy bash env file -- the pre-Go `bash/env/<name>`
// format sourced by 000-env.sh -- into the unified YAML env file this CLI reads.
//
// It is a one-way migration aid, not a shell interpreter: it understands the
// assignment forms the env files actually use (scalars, indexed arrays,
// associative arrays, `declare`/`export` prefixes, `${VAR}` references to
// earlier assignments) and ignores everything else. Variables it cannot map are
// reported as warnings rather than dropped silently, so a hand-edited env file
// never loses a setting without saying so.
package convert

import (
	"fmt"
	"regexp"
	"strings"
)

// vars is a parsed bash env file: scalar assignments, indexed arrays, and
// associative arrays, keyed by variable name. seen preserves assignment order so
// the unmapped-variable warnings come out in file order.
type vars struct {
	scalar map[string]string
	array  map[string][]string
	assoc  map[string]map[string]string
	seen   []string
	used   map[string]bool
}

func newVars() *vars {
	return &vars{
		scalar: map[string]string{},
		array:  map[string][]string{},
		assoc:  map[string]map[string]string{},
		used:   map[string]bool{},
	}
}

func (v *vars) record(name string) {
	if _, dup := v.scalar[name]; dup {
		return
	}
	if _, dup := v.array[name]; dup {
		return
	}
	if _, dup := v.assoc[name]; dup {
		return
	}
	v.seen = append(v.seen, name)
}

// s returns a scalar and marks the variable as mapped.
func (v *vars) s(name string) string {
	v.used[name] = true
	return v.scalar[name]
}

// l returns an indexed array and marks the variable as mapped. A scalar
// assignment is accepted as a one-element list, which is how a single-entry
// bash array is sometimes written.
func (v *vars) l(name string) []string {
	v.used[name] = true
	if a, ok := v.array[name]; ok {
		return a
	}
	if s, ok := v.scalar[name]; ok && s != "" {
		return []string{s}
	}
	return nil
}

// m returns an associative array and marks the variable as mapped.
func (v *vars) m(name string) map[string]string {
	v.used[name] = true
	return v.assoc[name]
}

// has reports whether the variable was assigned a non-empty value. It does not
// mark it as mapped -- callers use it only for platform detection.
func (v *vars) has(name string) bool {
	if s, ok := v.scalar[name]; ok && s != "" {
		return true
	}
	if a, ok := v.array[name]; ok && len(a) > 0 {
		return true
	}
	_, ok := v.assoc[name]
	return ok
}

// ignore marks a variable as mapped without reading it, for bash-only plumbing
// that has no YAML equivalent.
func (v *vars) ignore(names ...string) {
	for _, n := range names {
		v.used[n] = true
	}
}

// unmapped lists, in file order, every assigned variable no mapping consumed.
func (v *vars) unmapped() []string {
	var out []string
	for _, n := range v.seen {
		if !v.used[n] {
			out = append(out, n)
		}
	}
	return out
}

var (
	assignRE = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)
	assocRE  = regexp.MustCompile(`^\[(.+)\]=(.*)$`)
	refRE    = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}|\$([A-Za-z_][A-Za-z0-9_]*)`)
)

// parse reads a bash env file into vars. Lines that are not assignments
// (functions, conditionals, blank lines, comments) are skipped: an env file is
// declarations, and anything else is bash the YAML schema has no place for.
func parse(src string) (*vars, error) {
	v := newVars()
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")

	for i := 0; i < len(lines); i++ {
		line, isAssoc := stripDecl(strings.TrimSpace(lines[i]))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := assignRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name, rhs := m[1], m[2]

		if !strings.HasPrefix(rhs, "(") {
			v.record(name)
			v.scalar[name] = v.expand(firstToken(rhs))
			continue
		}

		// An array body may span lines; accumulate until the closing paren.
		body := rhs[1:]
		for closeIdx(body) < 0 {
			i++
			if i >= len(lines) {
				return nil, fmt.Errorf("unterminated array assignment for %s: no closing ')'", name)
			}
			body += "\n" + lines[i]
		}
		body = body[:closeIdx(body)]

		toks := tokenize(body)
		for j, t := range toks {
			toks[j] = v.expand(t)
		}
		v.record(name)
		if isAssoc || allAssocEntries(toks) {
			entries := map[string]string{}
			for _, t := range toks {
				if e := assocRE.FindStringSubmatch(t); e != nil {
					entries[e[1]] = e[2]
				}
			}
			v.assoc[name] = entries
			continue
		}
		v.array[name] = toks
	}
	return v, nil
}

// stripDecl removes any `declare`/`export`/`local` prefixes, reporting whether
// the declaration was an associative array (`declare -A`).
func stripDecl(line string) (string, bool) {
	assoc := false
	for {
		switch {
		case strings.HasPrefix(line, "declare -A "):
			line, assoc = strings.TrimSpace(line[len("declare -A "):]), true
		case strings.HasPrefix(line, "declare -a "):
			line = strings.TrimSpace(line[len("declare -a "):])
		case strings.HasPrefix(line, "declare "):
			line = strings.TrimSpace(line[len("declare "):])
		case strings.HasPrefix(line, "export "):
			line = strings.TrimSpace(line[len("export "):])
		case strings.HasPrefix(line, "local "):
			line = strings.TrimSpace(line[len("local "):])
		default:
			return line, assoc
		}
	}
}

// allAssocEntries reports whether every token has the `[key]=value` shape, which
// is how an associative array reads even without the `declare -A` prefix.
func allAssocEntries(toks []string) bool {
	if len(toks) == 0 {
		return false
	}
	for _, t := range toks {
		if !assocRE.MatchString(t) {
			return false
		}
	}
	return true
}

// closeIdx returns the index of the first `)` outside quotes and outside a
// comment, or -1 when the body is not yet terminated.
func closeIdx(s string) int {
	var q rune
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case q != 0:
			if r == q {
				q = 0
			}
		case r == '\'' || r == '"':
			q = r
		case r == '#':
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case r == ')':
			return i
		}
	}
	return -1
}

// tokenize splits a bash word list into words, honoring single and double
// quotes and dropping `#` comments. Adjacent quoted and bare chunks join into
// one word, so `[CA-NAME]="cert.pem"` stays a single token.
func tokenize(s string) []string {
	var (
		out    []string
		cur    strings.Builder
		inWord bool
		q      rune
	)
	flush := func() {
		if inWord {
			out = append(out, cur.String())
			cur.Reset()
			inWord = false
		}
	}
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case q != 0:
			if r == q {
				q = 0
				continue
			}
			if q == '"' && r == '\\' && i+1 < len(rs) {
				i++
				r = rs[i]
			}
			cur.WriteRune(r)
			inWord = true
		case r == '\'' || r == '"':
			q = r
			inWord = true // an empty "" is still a word
		case r == '#':
			flush()
			for i < len(rs) && rs[i] != '\n' {
				i++
			}
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
			inWord = true
		}
	}
	flush()
	return out
}

// firstToken unquotes a scalar right-hand side, dropping any trailing comment.
func firstToken(rhs string) string {
	toks := tokenize(rhs)
	if len(toks) == 0 {
		return ""
	}
	return toks[0]
}

// expand substitutes `$VAR` / `${VAR}` with an earlier scalar assignment, which
// is how the bash env files reference e.g. ${SOLBK_NS}. An unknown name expands
// to empty, exactly as bash would.
func (v *vars) expand(s string) string {
	if !strings.ContainsRune(s, '$') {
		return s
	}
	return refRE.ReplaceAllStringFunc(s, func(ref string) string {
		return v.scalar[strings.Trim(ref, "${}")]
	})
}
