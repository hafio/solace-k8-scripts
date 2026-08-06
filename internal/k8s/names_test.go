package k8s

import (
	"testing"

	"solace/internal/config"
)

func haCfg() *config.Config {
	return &config.Config{Redundancy: "yes", K8s: config.K8sConfig{Name: "dev-broker", Namespace: "solace"}}
}

func saCfg() *config.Config {
	return &config.Config{Redundancy: "no", K8s: config.K8sConfig{Name: "dev-broker", Namespace: "solace"}}
}

func TestResourceNames(t *testing.T) {
	cfg := haCfg()
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"pod primary", podName(cfg, config.Primary), "dev-broker-pubsubplus-p-0"},
		{"pod backup", podName(cfg, config.Backup), "dev-broker-pubsubplus-b-0"},
		{"pod monitor", podName(cfg, config.Monitor), "dev-broker-pubsubplus-m-0"},
		{"pvc primary", pvcName(cfg, config.Primary), "data-dev-broker-pubsubplus-p-0"},
		{"pvc monitor", pvcName(cfg, config.Monitor), "data-dev-broker-pubsubplus-m-0"},
		{"sts primary", stsName(cfg, config.Primary), "dev-broker-pubsubplus-p"},
		{"sts backup", stsName(cfg, config.Backup), "dev-broker-pubsubplus-b"},
		{"lb service", lbServiceName(cfg), "dev-broker-pubsubplus"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestHARoles(t *testing.T) {
	if got := HARoles(haCfg()); len(got) != 3 || got[0] != config.Primary || got[1] != config.Backup || got[2] != config.Monitor {
		t.Errorf("HARoles(HA) = %v, want [p b m]", got)
	}
	if got := HARoles(saCfg()); len(got) != 1 || got[0] != config.Primary {
		t.Errorf("HARoles(standalone) = %v, want [p]", got)
	}
}

func TestProductKeyRoles(t *testing.T) {
	if got := ProductKeyRoles(haCfg()); len(got) != 2 || got[0] != config.Primary || got[1] != config.Backup {
		t.Errorf("ProductKeyRoles(HA) = %v, want [p b]", got)
	}
	if got := ProductKeyRoles(saCfg()); len(got) != 1 || got[0] != config.Primary {
		t.Errorf("ProductKeyRoles(standalone) = %v, want [p]", got)
	}
}
