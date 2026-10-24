package ssh

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func hostNames(hosts []*ConfigHost) []string {
	names := make([]string, len(hosts))
	for i, h := range hosts {
		names[i] = h.Name()
	}
	sort.Strings(names)
	return names
}

// Tools like OrbStack and Colima add `Include` lines to ~/.ssh/config, so
// hosts defined in included files must show up, and the directive itself
// must not be listed as a host (issue #11).
func TestConfigInclude(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	writeFile(t, filepath.Join(dir, "config"), `# Added by OrbStack
# This only works at the top of ssh_config

Include ~/.orbstack/ssh/config
include conf.d/*.conf
Include=nested

Host radon
  HostName 192.168.3.10
`)
	writeFile(t, filepath.Join(home, ".orbstack/ssh/config"), "Host orb\n  HostName 127.0.0.1\n  Port 32222\n")
	writeFile(t, filepath.Join(dir, "conf.d/a.conf"), "Host cobalt\n")
	writeFile(t, filepath.Join(dir, "conf.d/b.conf"), "Host seed seedbox\n")
	writeFile(t, filepath.Join(dir, "conf.d/ignored.txt"), "Host nope\n")
	// Relative paths in nested files resolve against the top-level config dir, as in OpenSSH.
	writeFile(t, filepath.Join(dir, "nested"), "Include conf.d/deep\n")
	writeFile(t, filepath.Join(dir, "conf.d/deep"), "Host deep\n")

	got := hostNames(parseConfigFile(filepath.Join(dir, "config")))
	want := []string{"cobalt", "deep", "orb", "radon", "seed", "seedbox"}
	if len(got) != len(want) {
		t.Fatalf("hosts = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hosts = %v, want %v", got, want)
		}
	}
}

// A config that includes itself must not hang the Script Filter.
func TestConfigIncludeCycle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	writeFile(t, path, "Include config\nHost loop\n")

	if hosts := parseConfigFile(path); len(hosts) == 0 {
		t.Fatal("expected hosts from a self-including config")
	}
}

func TestConfigIncludeMissingFileIsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	writeFile(t, path, "Include does-not-exist/*\nHost radon\n")

	if got := hostNames(parseConfigFile(path)); len(got) != 1 || got[0] != "radon" {
		t.Fatalf("hosts = %v, want [radon]", got)
	}
}
