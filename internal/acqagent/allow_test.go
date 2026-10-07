package acqagent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAllows(t *testing.T) {
	a := &Allowlist{Pairs: [][2]string{{"node-a", "node-b"}}}
	cases := []struct {
		server, client string
		want           bool
	}{
		{"node-a", "node-b", true},
		{"node-b", "node-a", true},
		{"node-a", "node-c", false},
		{"", "node-b", false},
	}
	for _, c := range cases {
		if got := a.Allows(c.server, c.client); got != c.want {
			t.Errorf("Allows(%q, %q) = %v, mau %v", c.server, c.client, got, c.want)
		}
	}
}

func TestLoadAllowlist(t *testing.T) {
	baik := filepath.Join(t.TempDir(), "allow.json")
	if err := os.WriteFile(baik, []byte(`{"pairs":[["node-a","node-b"]]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAllowlist(baik)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Allows("node-a", "node-b") {
		t.Fatal("pasangan valid harus diizinkan")
	}
	buruk := filepath.Join(t.TempDir(), "buruk.json")
	if err := os.WriteFile(buruk, []byte(`{"pairs":[["node-a",""]]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAllowlist(buruk); err == nil {
		t.Fatal("identitas kosong harus ditolak")
	}
	if _, err := LoadAllowlist(filepath.Join(t.TempDir(), "hilang.json")); err == nil {
		t.Fatal("file hilang harus error")
	}
}
