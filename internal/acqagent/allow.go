package acqagent

import (
	"encoding/json"
	"fmt"
	"os"
)

// Allowlist adalah daftar pasangan identitas node yang boleh saling menarik.
// Format file JSON: {"pairs": [["node-a", "node-b"]]}. Urutan dalam pasangan bebas.
type Allowlist struct {
	Pairs [][2]string `json:"pairs"`
}

// LoadAllowlist membaca file allowlist JSON.
func LoadAllowlist(path string) (*Allowlist, error) {
	// #nosec G304 -- path dari flag operator lokal, bukan dari jaringan.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var a Allowlist
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, err
	}
	for _, p := range a.Pairs {
		if p[0] == "" || p[1] == "" {
			return nil, fmt.Errorf("allowlist punya identitas kosong: %q", p)
		}
	}
	return &a, nil
}

// Allows melaporkan apakah server dan client adalah pasangan yang diizinkan.
func (a *Allowlist) Allows(server, client string) bool {
	for _, p := range a.Pairs {
		if (p[0] == server && p[1] == client) || (p[0] == client && p[1] == server) {
			return true
		}
	}
	return false
}
