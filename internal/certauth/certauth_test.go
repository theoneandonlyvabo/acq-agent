package certauth

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
)

func TestCAAndNode(t *testing.T) {
	caPEM, _, caCert, caKey, err := CreateCA("test-ca", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !caCert.IsCA {
		t.Fatal("sertifikat CA harus bertanda IsCA")
	}
	leafPEM, _, err := SignNode(caCert, caKey, "node-a", []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, 2)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(leafPEM)
	if block == nil {
		t.Fatal("gagal decode PEM sertifikat node")
		return
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if leaf.Subject.CommonName != "node-a" {
		t.Fatalf("CN salah: %s", leaf.Subject.CommonName)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("gagal memuat CA ke pool")
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Fatalf("rantai sertifikat tidak valid: %v", err)
	}
}
