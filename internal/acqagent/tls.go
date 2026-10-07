package acqagent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// ServerTLS membangun opsi server gRPC dengan mTLS: client wajib menyerahkan
// sertifikat yang ditandatangani CA yang sama. Mengembalikan opsi server dan
// CommonName sertifikat server sebagai identitas node ini.
func ServerTLS(certFile, keyFile, caFile string) (grpc.ServerOption, string, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, "", err
	}
	if len(cert.Certificate) == 0 {
		return nil, "", fmt.Errorf("sertifikat kosong: %s", certFile)
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, "", err
	}
	pool, err := loadCAPool(caFile)
	if err != nil {
		return nil, "", err
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}
	return grpc.Creds(credentials.NewTLS(cfg)), leaf.Subject.CommonName, nil
}

// ClientTLS membangun kredensial TLS client dengan verifikasi server penuh.
func ClientTLS(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	pool, err := loadCAPool(caFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		ServerName:   serverName,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

func loadCAPool(caFile string) (*x509.CertPool, error) {
	// #nosec G304 -- path dari flag operator lokal, bukan dari jaringan.
	raw, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(raw) {
		return nil, fmt.Errorf("CA tidak valid: %s", caFile)
	}
	return pool, nil
}

// ClientCNFromContext mengambil CommonName client dari koneksi mTLS.
// String kosong berarti koneksi plaintext tanpa identitas.
func ClientCNFromContext(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	info, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok || len(info.State.VerifiedChains) == 0 {
		return ""
	}
	return info.State.VerifiedChains[0][0].Subject.CommonName
}
