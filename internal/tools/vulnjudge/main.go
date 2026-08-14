// Command vulnjudge turns a govulncheck JSON stream into the gate decision the
// dev-script contract asks for: fatal on a fixable vulnerability this module
// actually calls, a warning on one with no released fix.
//
// govulncheck cannot express that split itself. Its text mode exits non-zero for
// any finding at all, fixable or not, which would freeze releases on someone
// else's patch schedule; `-format json` exits 0 no matter what it finds. So the
// scripts capture the JSON and this judges it.
//
// Usage (see the `scan` task in scripts/dev.sh and scripts/dev.ps1):
//
//	go tool govulncheck -format json ./... > scan.json
//	go run ./internal/tools/vulnjudge scan.json
//
// Exit codes: 0 nothing to act on, 1 a fixable called vulnerability, 2 bad input.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// How govulncheck names standard library findings, whose fix is a toolchain
// bump rather than a `go get`.
const stdlibModule = "stdlib"

type position struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

type frame struct {
	Module   string    `json:"module"`
	Package  string    `json:"package"`
	Function string    `json:"function"`
	Position *position `json:"position"`
}

// message is the subset of govulncheck's stream messages the decision needs;
// `config` and `progress` entries decode to a zero value and are skipped.
type message struct {
	OSV *struct {
		ID      string `json:"id"`
		Summary string `json:"summary"`
	} `json:"osv"`
	Finding *struct {
		OSV          string  `json:"osv"`
		FixedVersion string  `json:"fixed_version"`
		Trace        []frame `json:"trace"`
	} `json:"finding"`
}

type vuln struct {
	id      string
	summary string
	fixed   string
	module  string
	site    string // file:line of the call in this module
	symbol  string // the calling function there
	called  bool
}

// fix returns the actionable next step for a fixable vulnerability.
func (v *vuln) fix() string {
	if v.module == stdlibModule {
		return fmt.Sprintf("raise the toolchain line in go.mod to %s or later", v.fixed)
	}
	return fmt.Sprintf("go get %s@%s", v.module, v.fixed)
}

// main is the thinnest possible wrapper: everything it would otherwise do lives in
// run, which returns an exit code instead of calling os.Exit, so the argument
// handling and both failure paths are testable in-process.
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is main's body. args excludes the program name; out takes the report and
// errOut the usage/read failures. It returns the process exit code: 2 for a usage
// or read error, otherwise whatever judge decided.
func run(args []string, out, errOut io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "usage: vulnjudge <govulncheck-json-file>")
		return 2
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(errOut, "vulnjudge: %v\n", err)
		return 2
	}
	code, report := judge(data)
	fmt.Fprint(out, report)
	return code
}

// judge parses a govulncheck JSON stream and returns the exit code plus the
// report to print. Findings are grouped by OSV id, since govulncheck emits a
// separate module-, package- and symbol-level finding for the same
// vulnerability.
func judge(data []byte) (int, string) {
	// PowerShell's UTF-8 writers prepend a BOM, which json.Decoder rejects.
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))))

	seen := map[string]*vuln{}
	entry := func(id string) *vuln {
		if v, ok := seen[id]; ok {
			return v
		}
		v := &vuln{id: id}
		seen[id] = v
		return v
	}

	for {
		var m message
		err := dec.Decode(&m)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 2, fmt.Sprintf("vulnjudge: malformed govulncheck JSON: %v\n", err)
		}

		switch {
		case m.OSV != nil:
			entry(m.OSV.ID).summary = m.OSV.Summary
		case m.Finding != nil:
			v := entry(m.Finding.OSV)
			if m.Finding.FixedVersion != "" {
				v.fixed = m.Finding.FixedVersion
			}
			if len(m.Finding.Trace) == 0 {
				continue
			}
			if m.Finding.Trace[0].Module != "" {
				v.module = m.Finding.Trace[0].Module
			}
			// Trace[0] is the vulnerable symbol. A function name there means
			// govulncheck traced a real call into it -- the same thing its text
			// mode counts as "your code is affected". Module- and package-level
			// findings carry no function and are reachable-but-not-called.
			if m.Finding.Trace[0].Function == "" {
				continue
			}
			v.called = true
			// The far end of the trace is the entry point in this module, so
			// the last frame carrying a position is the call site to report.
			for _, f := range m.Finding.Trace {
				if f.Position != nil {
					v.site = fmt.Sprintf("%s:%d", f.Position.Filename, f.Position.Line)
					v.symbol = f.Function
				}
			}
		}
	}

	var fixable, nofix []*vuln
	uncalled := 0
	for _, v := range seen {
		switch {
		case !v.called:
			uncalled++
		case v.fixed != "":
			fixable = append(fixable, v)
		default:
			nofix = append(nofix, v)
		}
	}
	byID := func(s []*vuln) { sort.Slice(s, func(i, j int) bool { return s[i].id < s[j].id }) }
	byID(fixable)
	byID(nofix)

	var b strings.Builder
	for _, v := range fixable {
		describe(&b, v, "fixed in "+v.fixed)
		fmt.Fprintf(&b, "  fix: %s\n", v.fix())
	}
	for _, v := range nofix {
		describe(&b, v, "no released fix")
	}

	ignored := ""
	if uncalled > 0 {
		ignored = fmt.Sprintf(" (%s not called by this module)", plural(uncalled, "vulnerability", "vulnerabilities"))
	}

	switch {
	case len(fixable) > 0:
		fmt.Fprintf(&b, "\n%s called by this module with a fix available%s\n",
			plural(len(fixable), "vulnerability", "vulnerabilities"), ignored)
		return 1, b.String()
	case len(nofix) > 0:
		fmt.Fprintf(&b, "\n%s called by this module, none with a released fix -- warning only%s\n",
			plural(len(nofix), "vulnerability", "vulnerabilities"), ignored)
		return 0, b.String()
	default:
		fmt.Fprintf(&b, "no vulnerabilities called by this module%s\n", ignored)
		return 0, b.String()
	}
}

// describe writes one vulnerability block: id, module, status, then whatever
// detail govulncheck gave us.
func describe(b *strings.Builder, v *vuln, status string) {
	module := v.module
	if module == "" {
		module = "unknown module"
	}
	fmt.Fprintf(b, "%s  %s  %s\n", v.id, module, status)
	if v.summary != "" {
		fmt.Fprintf(b, "  %s\n", v.summary)
	}
	if v.site != "" {
		fmt.Fprintf(b, "  called from %s (%s)\n", v.site, v.symbol)
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
