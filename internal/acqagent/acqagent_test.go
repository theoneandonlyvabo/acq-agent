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
	"acq-agent/internal/certauth"
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
	defer func() { _ = log.Close() }()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	acqagentv1.RegisterAcqAgentServer(srv, &Server{Log: log})
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	dst := filepath.Join(dir, "hasil.dd")
	res, err := Pull(context.Background(), lis.Addr().String(), src, dst, log, nil, nil)
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

// mTLS + allowlist ujung-ke-ujung: pasangan izin lolos, pasangan asing ditolak.
func TestTLSAllowlist(t *testing.T) {
	dir := t.TempDir()
	caPEM, _, caCert, caKey, err := certauth.CreateCA("test-ca", 2)
	if err != nil {
		t.Fatal(err)
	}
	save := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	caFile := save("ca.pem", caPEM)
	serverCert, serverKey, err := certauth.SignNode(caCert, caKey, "node-a", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, 2)
	if err != nil {
		t.Fatal(err)
	}
	clientCert, clientKey, err := certauth.SignNode(caCert, caKey, "node-b", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, 2)
	if err != nil {
		t.Fatal(err)
	}
	serverCertFile := save("node-a.pem", serverCert)
	serverKeyFile := save("node-a-key.pem", serverKey)
	clientCertFile := save("node-b.pem", clientCert)
	clientKeyFile := save("node-b-key.pem", clientKey)
	allowFile := save("allow.json", []byte(`{"pairs":[["node-a","node-b"]]}`))

	data := make([]byte, 1<<20+7)
	for i := range data {
		data[i] = byte(i * 17 % 251)
	}
	src := save("sumber.dd", data)

	log, err := audit.Open(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = log.Close() }()

	opt, nodeID, err := ServerTLS(serverCertFile, serverKeyFile, caFile)
	if err != nil {
		t.Fatal(err)
	}
	if nodeID != "node-a" {
		t.Fatalf("identitas server salah: %s", nodeID)
	}
	allow, err := LoadAllowlist(allowFile)
	if err != nil {
		t.Fatal(err)
	}
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(opt)
	handler := &Server{Log: log, NodeID: nodeID, Allow: allow}
	acqagentv1.RegisterAcqAgentServer(srv, handler)
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	tlsCfg, err := ClientTLS(clientCertFile, clientKeyFile, caFile, "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "hasil.dd")
	res, err := Pull(context.Background(), lis.Addr().String(), src, dst, log, tlsCfg, nil)
	if err != nil {
		t.Fatalf("pasangan izin harus lolos: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) || res.Bytes != uint64(len(data)) {
		t.Fatal("isi file hasil tidak identik dengan sumber")
	}

	handler.Allow = &Allowlist{Pairs: [][2]string{{"node-a", "node-c"}}}
	if _, err := Pull(context.Background(), lis.Addr().String(), src, dst, log, tlsCfg, nil); err == nil {
		t.Fatal("pasangan asing harus ditolak")
	} else if !strings.Contains(err.Error(), "tidak diizinkan") {
		t.Fatalf("pesan error salah: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"event":"pull.failed"`) {
		t.Fatal("audit log tidak memuat pull.failed")
	}
}
