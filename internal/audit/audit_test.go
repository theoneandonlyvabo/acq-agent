package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tiap panggilan harus menghasilkan tepat satu baris JSON dengan urutan kunci tetap.
func TestSingleLineJSON(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.log")
	l, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	l.Info("uji.info", "halo", map[string]any{"b": 1})
	l.Error("uji.gagal", "aduh", nil)

	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 2 {
		t.Fatalf("ingin 2 baris log, dapat %d", len(lines))
	}
	for _, line := range lines {
		idxTS := strings.Index(line, `"ts"`)
		idxLevel := strings.Index(line, `"level"`)
		idxEvent := strings.Index(line, `"event"`)
		idxMsg := strings.Index(line, `"msg"`)
		if !(0 <= idxTS && idxTS < idxLevel && idxLevel < idxEvent && idxEvent < idxMsg) {
			t.Errorf("urutan kunci tidak canonical: %s", line)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Errorf("baris bukan JSON valid: %v", err)
		}
	}
}
