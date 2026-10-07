package acqagent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/audit"
)

func TestRejectPhysicalPath(t *testing.T) {
	for _, p := range []string{"/dev/sda", "/dev/loop0", `\\.\PhysicalDrive0`, `\\?\C:\x`} {
		if err := rejectPhysicalPath(p); err == nil {
			t.Errorf("path %q harus ditolak", p)
		}
	}
	for _, p := range []string{"testdata/disk01.dd", filepath.Join(t.TempDir(), "x.dd")} {
		if err := rejectPhysicalPath(p); err != nil {
			t.Errorf("path %q harus diterima: %v", p, err)
		}
	}
}

// Roundtrip penuh lewat TCP localhost: serve lalu pull, byte dan hash harus identik.
func TestPullRoundtrip(t *testing.T) {
	dir := t.TempDir()
	// Isi deterministik 2 MiB + 12345 byte, sengaja tidak kelipatan chunk.
	data := make([]byte, 2*(1<<20)+12345)
	for i := range data {
		data[i] = byte(i * 31 % 251)
	}
	wantSum := sha256.Sum256(data)
	wantHex := hex.EncodeToString(wantSum[:])
	src := filepath.Join(dir, "sumber.dd")
	if err := os.WriteFile(src, data, 0o600); err != nil {
		t.Fatal(err)
	}

	log, err := audit.Open(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	acqagentv1.RegisterAcqAgentServer(srv, &Server{Log: log})
	go srv.Serve(lis)
	defer srv.GracefulStop()

	dst := filepath.Join(dir, "hasil.dd")
	res, err := Pull(context.Background(), lis.Addr().String(), src, dst, log)
	if err != nil {
		t.Fatal(err)
	}
	if res.Bytes != uint64(len(data)) || res.SHA256 != wantHex {
		t.Fatalf("ringkasan salah: %+v, mau bytes=%d sha256=%s", res, len(data), wantHex)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("isi file hasil tidak identik dengan sumber")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{`"event":"pull.started"`, `"event":"pull.completed"`} {
		if !strings.Contains(string(raw), event) {
			t.Errorf("audit log tidak memuat %s", event)
		}
	}
}
