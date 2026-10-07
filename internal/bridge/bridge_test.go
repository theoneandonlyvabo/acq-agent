package bridge

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/acqagent"
	"acq-agent/internal/audit"
)

func TestProbe(t *testing.T) {
	dir := t.TempDir()
	log, auditPath := testLogger(t, dir)
	defer func() { _ = log.Close() }()
	b := New(log, "node-a", "127.0.0.1:50051", auditPath, "", "", "")
	ts := httptest.NewServer(b.Handler())
	defer ts.Close()

	open, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = open.Close() }()
	closed, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closedAddr := closed.Addr().String()
	_ = closed.Close()

	var up map[string]any
	getJSON(t, ts.URL+"/api/probe?addr="+open.Addr().String(), http.StatusOK, &up)
	if up["online"] != true {
		t.Fatalf("port terbuka harus online: %v", up)
	}
	var down map[string]any
	getJSON(t, ts.URL+"/api/probe?addr="+closedAddr, http.StatusOK, &down)
	if down["online"] != false {
		t.Fatalf("port tertutup harus offline: %v", down)
	}
	getJSON(t, ts.URL+"/api/probe", http.StatusBadRequest, nil)
}

func TestLoopbackGuard(t *testing.T) {
	for _, addr := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"} {
		if err := ensureLoopback(addr); err != nil {
			t.Errorf("%s harus diterima: %v", addr, err)
		}
	}
	for _, addr := range []string{"0.0.0.0:8080", "192.168.1.5:8080", "example.com:80", "asal"} {
		if err := ensureLoopback(addr); err == nil {
			t.Errorf("%s harus ditolak", addr)
		}
	}
}

func testLogger(t *testing.T, dir string, lines ...string) (*audit.Logger, string) {
	t.Helper()
	path := filepath.Join(dir, "audit.log")
	log, err := audit.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range lines {
		log.Info("uji", line, nil)
	}
	return log, path
}

func getJSON(t *testing.T, url string, code int, v any) {
	t.Helper()
	res, err := http.Get(url) // #nosec G107 -- URL test lokal.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != code {
		t.Fatalf("GET %s = %d, mau %d", url, res.StatusCode, code)
	}
	if v != nil {
		if err := json.NewDecoder(res.Body).Decode(v); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEndpoints(t *testing.T) {
	dir := t.TempDir()
	log, auditPath := testLogger(t, dir, "satu", "dua")
	defer func() { _ = log.Close() }()
	b := New(log, "node-a", "127.0.0.1:50051", auditPath, "", "", "")
	ts := httptest.NewServer(b.Handler())
	defer ts.Close()

	var status map[string]any
	getJSON(t, ts.URL+"/api/status", http.StatusOK, &status)
	if status["nodeID"] != "node-a" || status["tls"] != false {
		t.Fatalf("status salah: %v", status)
	}

	res, err := http.Post(ts.URL+"/api/pull", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("pull kosong harus 400, dapat %d", res.StatusCode)
	}

	getJSON(t, ts.URL+"/api/jobs/ngawur", http.StatusNotFound, nil)

	var auditRes struct {
		Lines []string `json:"lines"`
	}
	getJSON(t, ts.URL+"/api/audit?limit=1", http.StatusOK, &auditRes)
	if len(auditRes.Lines) != 1 {
		t.Fatalf("ingin 1 baris, dapat %d", len(auditRes.Lines))
	}
	getJSON(t, ts.URL+"/api/audit?limit=0", http.StatusBadRequest, nil)
}

// Pull async ujung-ke-ujung lewat bridge: polling sampai done, hash cocok.
func TestBridgePullE2E(t *testing.T) {
	dir := t.TempDir()
	data := make([]byte, 1<<20+7)
	for i := range data {
		data[i] = byte(i * 17 % 251)
	}
	wantSum := sha256.Sum256(data)
	wantHex := hex.EncodeToString(wantSum[:])
	src := filepath.Join(dir, "sumber.dd")
	if err := os.WriteFile(src, data, 0o600); err != nil {
		t.Fatal(err)
	}

	log, auditPath := testLogger(t, dir)
	defer func() { _ = log.Close() }()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	acqagentv1.RegisterAcqAgentServer(srv, &acqagent.Server{Log: log})
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	b := New(log, "node-a", lis.Addr().String(), auditPath, "", "", "")
	ts := httptest.NewServer(b.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]string{"from": lis.Addr().String(), "src": src, "out": filepath.Join(dir, "hasil.dd")})
	res, err := http.Post(ts.URL+"/api/pull", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var created map[string]string
	if err := json.NewDecoder(res.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusAccepted || created["jobID"] == "" {
		t.Fatalf("pull harus 202 + jobID, dapat %d %v", res.StatusCode, created)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		var job Job
		getJSON(t, ts.URL+"/api/jobs/"+created["jobID"], http.StatusOK, &job)
		if job.State == "done" {
			if job.Bytes != uint64(len(data)) || job.SHA256 != wantHex {
				t.Fatalf("job salah: %+v", job)
			}
			break
		}
		if job.State == "failed" {
			t.Fatalf("job gagal: %s", job.Error)
		}
		if time.Now().After(deadline) {
			t.Fatal("job tak kunjung selesai")
		}
		time.Sleep(100 * time.Millisecond)
	}
	got, err := os.ReadFile(filepath.Join(dir, "hasil.dd"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("isi file hasil tidak identik dengan sumber")
	}
	raw, err := os.ReadFile(auditPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"event":"pull.completed"`) {
		t.Fatal("audit log tidak memuat pull.completed")
	}
}

// Origin http loopback (port berapa pun) boleh; sisanya ditolak.
func TestCORSDevOrigins(t *testing.T) {
	dir := t.TempDir()
	log, auditPath := testLogger(t, dir)
	defer func() { _ = log.Close() }()
	b := New(log, "node-a", "127.0.0.1:50051", auditPath, "", "", "")
	ts := httptest.NewServer(b.Handler())
	defer ts.Close()

	allowed := []string{
		"http://localhost:5173",
		"http://127.0.0.1:5174",
		"http://localhost:9999",
		"http://127.0.0.1:8080",
		"http://[::1]:5173",
	}
	denied := []string{
		"https://localhost:5173",
		"https://127.0.0.1:5173",
		"http://example.com",
		"http://192.168.1.5:5173",
		"",
	}
	check := func(origin string, want string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/status", nil)
		if err != nil {
			t.Fatal(err)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		if got := res.Header.Get("Access-Control-Allow-Origin"); got != want {
			t.Errorf("origin %q: header = %q, mau %q", origin, got, want)
		}
	}
	for _, origin := range allowed {
		check(origin, origin)
	}
	for _, origin := range denied {
		check(origin, "")
	}
}
