package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlatforms pins the enumerator's contents and order: the order is what a
// prompt lists and what an error names, so it is part of the contract rather
// than an accident of the literal.
func TestPlatforms(t *testing.T) {
	got := Platforms()
	want := []Platform{K8s, Docker, Podman}
	if len(got) != len(want) {
		t.Fatalf("Platforms() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Platforms()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestParsePlatform covers every accepted spelling plus the rejections. The
// retired k8s spelling is listed explicitly: it used to be accepted, and a test
// that merely omitted it would not notice it being quietly reinstated.
func TestParsePlatform(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    Platform
		wantErr bool
	}{
		{in: "", want: ""},
		{in: "kubernetes", want: K8s},
		{in: "docker", want: Docker},
		{in: "podman", want: Podman},
		{in: "kube", want: K8s},
		{in: "dk", want: Docker},
		{in: "pm", want: Podman},
		{in: "k8s", wantErr: true},
		{in: "k8", wantErr: true},
		{in: "KUBERNETES", wantErr: true},
		{in: "KUBE", wantErr: true},
		{in: " docker", wantErr: true},
	} {
		got, err := ParsePlatform(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParsePlatform(%q) = %q, want an error", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePlatform(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParsePlatform(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestParsePlatformErrorTeachesCanonical pins that a rejection names the words
// the env file's own sections use. An error that only listed the abbreviations
// would teach the short form as the real name.
func TestParsePlatformErrorTeachesCanonical(t *testing.T) {
	_, err := ParsePlatform("kubernets")
	if err == nil {
		t.Fatal("ParsePlatform(\"kubernets\") should fail")
	}
	for _, want := range []string{"kubernetes", "docker", "podman", "kube", "dk", "pm"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should name %q", err, want)
		}
	}
}

// writeEnv writes body to a temp file and returns its path.
func writeEnv(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "env.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write env: %v", err)
	}
	return path
}

// TestDetectPlatforms covers the whole resolution input space: none, one, and
// several sections, plus the empty-section marker form. The marker case is the
// load-bearing one -- the container schema has no mandatory field, so `docker: {}`
// is the only way such a file can say which platform it is for.
func TestDetectPlatforms(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		want []Platform
	}{
		{name: "none", body: "image:\n  repo: solace/solace-pubsub-standard\n", want: nil},
		{name: "kubernetes only", body: "kubernetes:\n  name: solace\n", want: []Platform{K8s}},
		{name: "docker only", body: "docker:\n  container:\n    dataDir: /opt/solace/data\n", want: []Platform{Docker}},
		{name: "podman only", body: "podman:\n  rootless: true\n", want: []Platform{Podman}},
		{name: "empty marker counts", body: "docker: {}\n", want: []Platform{Docker}},
		{name: "null section counts", body: "podman:\n", want: []Platform{Podman}},
		{name: "two sections", body: "kubernetes:\n  name: solace\ndocker: {}\n", want: []Platform{K8s, Docker}},
		{
			name: "all three, reported in Platforms order",
			body: "podman: {}\ndocker: {}\nkubernetes: {}\n",
			want: []Platform{K8s, Docker, Podman},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DetectPlatforms(writeEnv(t, tc.body))
			if err != nil {
				t.Fatalf("DetectPlatforms error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("DetectPlatforms = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("DetectPlatforms[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestDetectPlatformsMissingFile pins that a bad path fails here rather than
// reaching Load with an empty platform list, which would report the wrong thing.
func TestDetectPlatformsMissingFile(t *testing.T) {
	_, err := DetectPlatforms(filepath.Join(t.TempDir(), "absent.yaml"))
	if err == nil {
		t.Fatal("DetectPlatforms should fail on a missing file")
	}
}

// TestDetectPlatformsBashFileHint pins that the legacy-bash-env mistake keeps
// its actionable message here. DetectPlatforms is now the first thing to read
// the file, so it is where that mistake surfaces: without this it would degrade
// to a bare YAML decode error and lose the `solace-util convert` hint.
func TestDetectPlatformsBashFileHint(t *testing.T) {
	_, err := DetectPlatforms(writeEnv(t, "SOLBK_NAME=\"solace\"\nSOLBK_NS=solace-ns\n\tbad\n"))
	if err == nil {
		t.Fatal("DetectPlatforms should reject a legacy bash env file")
	}
	if !strings.Contains(err.Error(), "convert") {
		t.Errorf("error %q should point at solace-util convert", err)
	}
}
