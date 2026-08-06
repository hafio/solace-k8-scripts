package broker

import (
	"strings"
	"testing"
)

func TestFixedScripts(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"show-redundancy", showRedundancyScript(), "show redundancy\n"},
		{"show-redundancy-detail", showRedundancyDetailScript(), "no paging\nshow redundancy detail\n"},
		{"revert-activity", revertActivityScript(), "home\nno paging\nenable\nadmin\nredundancy revert-activity\n"},
		{"release-activity", releaseActivityScript(), "home\nno paging\nenable\nconfigure\nredundancy release-activity\n"},
		{"no-release-activity", noReleaseActivityScript(), "home\nno paging\nenable\nconfigure\nno redundancy release-activity\n"},
		{"show-vpn", showVPNScript(), "home\nenable\nconfigure\nshow message-vpn *\n"},
		{"show-vpn-bare", showVPNBareScript(), "show message-vpn *\n"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}
}

// The revert during a redundancy test preserves a trailing space after the
// command (061); the leader-path revert does not (050). Guard both so a stray
// gofmt/whitespace change is caught.
func TestRevertActivityTrailingSpace(t *testing.T) {
	if got := revertActivityConfigureScript(); !strings.Contains(got, "redundancy revert-activity \n") {
		t.Errorf("revertActivityConfigureScript lost its trailing space: %q", got)
	}
	if strings.Contains(revertActivityScript(), "revert-activity \n") {
		t.Error("revertActivityScript should not have a trailing space after the command")
	}
}

func TestAssertLeaderScript(t *testing.T) {
	got := assertLeaderScript()
	for _, want := range []string{
		"config-sync assert-leader router\n",
		"config-sync assert-leader message-vpn *\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("assertLeaderScript missing %q", want)
		}
	}
	if !strings.HasSuffix(got, "show config-sync database\n") {
		t.Errorf("assertLeaderScript should end with the database show: %q", got)
	}
}

func TestServerCertScript(t *testing.T) {
	if got, want := serverCertFile("2026-07-31"), "tls-2026-07-31.crt.key"; got != want {
		t.Fatalf("serverCertFile = %q, want %q", got, want)
	}
	got := serverCertScript("2026-07-31")
	if !strings.HasPrefix(got, "enable\nconfigure\n") {
		t.Errorf("serverCertScript prefix: %q", got)
	}
	if !strings.Contains(got, "ssl server-certificate tls-2026-07-31.crt.key\n") {
		t.Errorf("serverCertScript missing the load line: %q", got)
	}
	if !strings.HasSuffix(got, "show ssl server-certificate detail\n") {
		t.Errorf("serverCertScript suffix: %q", got)
	}
}

func TestDomainCertsScriptSorted(t *testing.T) {
	// Unsorted map input must emit CAs in sorted order for deterministic output.
	got := domainCertsScript(map[string]string{"zeta": "z.pem", "alpha": "a.pem"})
	if !strings.HasPrefix(got, "no paging\nenable\nconfigure\nssl\n") {
		t.Errorf("domainCertsScript prefix: %q", got)
	}
	if !strings.HasSuffix(got, "end\nshow domain-certificate-authority ca-name *\n") {
		t.Errorf("domainCertsScript suffix: %q", got)
	}
	a := strings.Index(got, "create domain-certificate-authority alpha")
	z := strings.Index(got, "create domain-certificate-authority zeta")
	if a < 0 || z < 0 || a > z {
		t.Errorf("domainCertsScript not sorted (alpha=%d zeta=%d): %q", a, z, got)
	}
	if !strings.Contains(got, "create domain-certificate-authority alpha\ncertificate file a.pem\nexit\n") {
		t.Errorf("domainCertsScript missing alpha block: %q", got)
	}
}

func TestDisableDefaultUsersScriptQuoting(t *testing.T) {
	got := disableDefaultUsersScript([]string{"default", "my vpn"})
	if !strings.HasPrefix(got, "home\nenable\nconfigure\n") {
		t.Errorf("disableDefaultUsersScript prefix: %q", got)
	}
	// VPN names must be shell-safe quoted; a name with a space must stay one token.
	if !strings.Contains(got, `client-username default message-vpn "my vpn"`) {
		t.Errorf("disableDefaultUsersScript did not quote a spaced VPN name: %q", got)
	}
	if !strings.HasSuffix(got, "end\nshow client-username default message-vpn *\n") {
		t.Errorf("disableDefaultUsersScript suffix: %q", got)
	}
}

func TestProductKeysScript(t *testing.T) {
	got := productKeysScript([]string{"KEY-1", "KEY-2"})
	want := "enable\nadmin\nproduct-key KEY-1\nproduct-key KEY-2\nshow product-key\n"
	if got != want {
		t.Errorf("productKeysScript = %q, want %q", got, want)
	}
}

func TestDisableDefaultVPNScript(t *testing.T) {
	got := disableDefaultVPNScript()
	for _, want := range []string{
		`message-vpn "default"`,
		`client-username "default" message-vpn "default"`,
		"no ssl allow-downgrade-to-plain-text",
		"service smf plain-text shutdown",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("disableDefaultVPNScript missing %q", want)
		}
	}
}

func TestParseVPNNames(t *testing.T) {
	// The parser reads the VPN name from the first 32 columns, so each data row
	// must pad the name well past column 32 before the next column begins.
	row := func(name string) string { return name + strings.Repeat(" ", 40-len(name)) + "Yes" }
	out := strings.Join([]string{
		"Flags Legend:",
		"Message VPN" + strings.Repeat(" ", 29) + "Enabled",
		strings.Repeat("-", 40),
		row("default"),
		row("myvpn"),
		"# a comment row is skipped",
		"",
		row("another"),
	}, "\r\n")
	got := parseVPNNames(out)
	want := []string{"default", "myvpn", "another"}
	if len(got) != len(want) {
		t.Fatalf("parseVPNNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("parseVPNNames[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseVPNNamesNoSeparator(t *testing.T) {
	if got := parseVPNNames("no dashes here\njust text\n"); len(got) != 0 {
		t.Errorf("parseVPNNames without separator = %v, want empty", got)
	}
}

func TestGatherConfigsScript(t *testing.T) {
	got := gatherConfigsScript(7)
	if !strings.HasPrefix(got, "home\nno paging\n") {
		t.Errorf("gatherConfigsScript prefix: %q", got[:40])
	}
	if !strings.Contains(got, "show acl-profile * > configs/cliout/show-aclprofiles.out\n") {
		t.Error("gatherConfigsScript missing the first show command")
	}
	if !strings.Contains(got, "gather-diagnostics days-of-history '7' no-encrypt\n") {
		t.Error("gatherConfigsScript missing gather-diagnostics with days substituted")
	}
	if n := strings.Count(got, "> configs/cliout/show-"); n != len(gatherShowCommands) {
		t.Errorf("gatherConfigsScript emitted %d show lines, want %d", n, len(gatherShowCommands))
	}
}

func TestZipConfigsScript(t *testing.T) {
	got := zipConfigsScript()
	if !strings.Contains(got, "zip gather-configs.zip -q -r cli-out/*") {
		t.Errorf("zipConfigsScript missing the zip command: %q", got)
	}
}

func TestSortedKeys(t *testing.T) {
	got := sortedKeys(map[string]string{"c": "", "a": "", "b": ""})
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedKeys = %v, want %v", got, want)
		}
	}
}
