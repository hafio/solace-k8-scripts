package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return data
}

// The whole point of the task: a fix exists for something we call, so fail; no
// fix exists, so warn and pass. Anything we merely depend on is neither.
func TestJudge(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		wantCode int
		contains []string
	}{
		{
			name:     "called and fixable fails",
			fixture:  "called_fixable.json",
			wantCode: 1,
			contains: []string{
				"GO-2026-4971  stdlib  fixed in go1.26.3",
				"called from internal/container/manager.go:59 (NewManager)",
				"fix: raise the toolchain line in go.mod to go1.26.3 or later",
				"1 vulnerability called by this module with a fix available",
				"(1 vulnerability not called by this module)",
			},
		},
		{
			name:     "called with no released fix warns and passes",
			fixture:  "called_unfixable.json",
			wantCode: 0,
			contains: []string{
				"GO-2026-9001  gopkg.in/yaml.v3  no released fix",
				"called from internal/config/load.go:88 (Load)",
				"none with a released fix -- warning only",
			},
		},
		{
			name:     "uncalled findings are ignored",
			fixture:  "uncalled_only.json",
			wantCode: 0,
			contains: []string{"no vulnerabilities called by this module (1 vulnerability not called"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, report := judge(load(t, tc.fixture))
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d\nreport:\n%s", code, tc.wantCode, report)
			}
			for _, want := range tc.contains {
				if !strings.Contains(report, want) {
					t.Errorf("report missing %q\ngot:\n%s", want, report)
				}
			}
		})
	}
}

// A fixable vulnerability in a module (not the stdlib) gets a `go get` hint
// rather than a toolchain bump, so the advice matches what actually fixes it.
func TestJudgeModuleFixHint(t *testing.T) {
	stream := `{"osv":{"id":"GO-2026-7777","summary":"boom"}}
{"finding":{"osv":"GO-2026-7777","fixed_version":"v1.9.1","trace":[
  {"module":"github.com/spf13/cobra","package":"github.com/spf13/cobra","function":"Execute"},
  {"module":"solace","package":"solace/internal/cli","function":"Run","position":{"filename":"internal/cli/root.go","line":31}}]}}`

	code, report := judge([]byte(stream))
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if want := "fix: go get github.com/spf13/cobra@v1.9.1"; !strings.Contains(report, want) {
		t.Errorf("report missing %q\ngot:\n%s", want, report)
	}
}

// An empty scan is the everyday case and must not be mistaken for a failure.
func TestJudgeNoFindings(t *testing.T) {
	code, report := judge([]byte(`{"config":{"scanner_name":"govulncheck"}}`))
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if want := "no vulnerabilities called by this module\n"; report != want {
		t.Errorf("report = %q, want %q", report, want)
	}
}

// A UTF-8 BOM is what PowerShell's encoders emit; json.Decoder rejects it, so
// the scan task would fail on Windows for no real reason.
func TestJudgeTolerantOfBOM(t *testing.T) {
	data := append([]byte("\xef\xbb\xbf"), load(t, "uncalled_only.json")...)
	if code, report := judge(data); code != 0 {
		t.Errorf("exit code = %d, want 0\nreport:\n%s", code, report)
	}
}

// Truncated or non-JSON input is a systemic problem, not a clean scan: it must
// fail loudly rather than report success.
func TestJudgeMalformedInput(t *testing.T) {
	for _, in := range []string{`{"finding":{`, `not json at all`, `{"osv":{"id":1}}`} {
		code, report := judge([]byte(in))
		if code != 2 {
			t.Errorf("judge(%q) exit code = %d, want 2", in, code)
		}
		if !strings.Contains(report, "malformed govulncheck JSON") {
			t.Errorf("judge(%q) report = %q, want a malformed-JSON message", in, report)
		}
	}
}

// A finding with an empty trace is a legitimately malformed-but-valid-JSON
// record (govulncheck emitted the OSV/finding pair but traced no frames). The
// empty-trace guard must fold it into "uncalled" instead of indexing Trace[0]
// a few lines later and panicking.
func TestJudgeEmptyTrace(t *testing.T) {
	code, report := judge([]byte(`{"finding":{"osv":"GO-2026-5000","trace":[]}}`))
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nreport:\n%s", code, report)
	}
	if want := "1 vulnerability not called by this module"; !strings.Contains(report, want) {
		t.Errorf("report missing %q\ngot:\n%s", want, report)
	}
}

// With only one vuln per bucket in every other fixture, sort.Slice never calls
// byID's comparator, so a broken or deleted Less would go uncaught by CI (the
// "seen" map randomizes iteration order every run). Two fixable OSVs given out
// of ID order forces the comparator to actually run.
func TestJudgeSortsFixableByID(t *testing.T) {
	stream := `{"osv":{"id":"GO-2026-9000","summary":"boom high"}}
{"finding":{"osv":"GO-2026-9000","fixed_version":"v2.0.0","trace":[
  {"module":"example.com/foo","package":"example.com/foo","function":"Bar","position":{"filename":"foo.go","line":10}}]}}
{"osv":{"id":"GO-2026-1000","summary":"boom low"}}
{"finding":{"osv":"GO-2026-1000","fixed_version":"v1.0.0","trace":[
  {"module":"example.com/baz","package":"example.com/baz","function":"Qux","position":{"filename":"baz.go","line":20}}]}}`

	code, report := judge([]byte(stream))
	if code != 1 {
		t.Errorf("exit code = %d, want 1\nreport:\n%s", code, report)
	}
	idxLow := strings.Index(report, "GO-2026-1000")
	idxHigh := strings.Index(report, "GO-2026-9000")
	if idxLow == -1 || idxHigh == -1 {
		t.Fatalf("report missing one of the two OSV ids\ngot:\n%s", report)
	}
	if idxLow > idxHigh {
		t.Errorf("expected GO-2026-1000's block before GO-2026-9000's\ngot:\n%s", report)
	}
	if want := "2 vulnerabilities called by this module"; !strings.Contains(report, want) {
		t.Errorf("report missing %q\ngot:\n%s", want, report)
	}
}

// A called finding whose trace[0] carries a function but no module is a
// plausible partial govulncheck record. describe must fall back to the
// "unknown module" placeholder rather than print a blank field.
func TestJudgeUnknownModule(t *testing.T) {
	code, report := judge([]byte(`{"finding":{"osv":"GO-2026-6000","trace":[{"function":"SomeFunc"}]}}`))
	if code != 0 {
		t.Errorf("exit code = %d, want 0\nreport:\n%s", code, report)
	}
	if want := "GO-2026-6000  unknown module  no released fix"; !strings.Contains(report, want) {
		t.Errorf("report missing %q\ngot:\n%s", want, report)
	}
}

// run's usage branch must fire for any argument count other than exactly one,
// print the message on errOut (not out), and return exit code 2.
func TestRunUsageError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"noArgs", nil},
		{"tooManyArgs", []string{"a", "b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := run(tc.args, &out, &errOut)
			if code != 2 {
				t.Errorf("run(%v) code = %d, want 2", tc.args, code)
			}
			if !strings.Contains(errOut.String(), "usage: vulnjudge") {
				t.Errorf("run(%v) errOut = %q, want a usage message", tc.args, errOut.String())
			}
			if out.Len() != 0 {
				t.Errorf("run(%v) out = %q, want empty on a usage error", tc.args, out.String())
			}
		})
	}
}

// An unreadable path (missing file, bad permissions) is a setup failure, not a
// scan result: run must name the file in the error and return code 2 rather
// than handing a zero-value byte slice to judge.
func TestRunUnreadableFile(t *testing.T) {
	var out, errOut bytes.Buffer
	path := filepath.Join("testdata", "does-not-exist.json")
	code := run([]string{path}, &out, &errOut)
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), path) {
		t.Errorf("errOut = %q, want it to name the unreadable file %q", errOut.String(), path)
	}
	if out.Len() != 0 {
		t.Errorf("out = %q, want empty on a read error", out.String())
	}
}

// The happy path: run reads the file, hands it to judge, prints judge's
// report to out, and returns judge's code -- proving the wiring between run
// and judge, not just judge in isolation.
func TestRunHappyPath(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{filepath.Join("testdata", "called_fixable.json")}, &out, &errOut)
	if code != 1 {
		t.Errorf("code = %d, want 1\nout:\n%s", code, out.String())
	}
	if !strings.Contains(out.String(), "GO-2026-4971") {
		t.Errorf("out missing report content, got:\n%s", out.String())
	}
	if errOut.Len() != 0 {
		t.Errorf("errOut = %q, want empty on success", errOut.String())
	}
}
