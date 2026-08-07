package main

import (
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
