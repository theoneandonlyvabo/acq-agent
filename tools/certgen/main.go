// Program kecil pembuat CA dan sertifikat node untuk latihan mTLS lokal.
// Bukan untuk produksi. Hasilnya masuk certs/ yang diabaikan git.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"acq-agent/internal/certauth"
)

func main() {
	out := flag.String("out", "certs", "direktori output")
	nodes := flag.String("nodes", "", "daftar CN node dipisah koma, mis. node-a,node-b")
	days := flag.Int("days", 365, "masa berlaku hari")
	flag.Parse()
	if *nodes == "" || *days <= 0 {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(*out, strings.Split(*nodes, ","), *days); err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func run(out string, nodes []string, days int) error {
	for _, n := range nodes {
		if strings.TrimSpace(n) == "" {
			return fmt.Errorf("nama node tidak boleh kosong")
		}
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		return err
	}
	save := func(name string, data []byte) error {
		p := filepath.Join(out, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			return err
		}
		fmt.Println("wrote", p)
		return nil
	}
	caPEM, caKeyPEM, caCert, caKey, err := certauth.CreateCA("acqagent-ca", days)
	if err != nil {
		return err
	}
	if err := save("ca.pem", caPEM); err != nil {
		return err
	}
	if err := save("ca-key.pem", caKeyPEM); err != nil {
		return err
	}
	for _, n := range nodes {
		name := strings.TrimSpace(n)
		certPEM, keyPEM, err := certauth.SignNode(caCert, caKey, name, []string{"localhost"}, []net.IP{net.ParseIP("127.0.0.1")}, days)
		if err != nil {
			return err
		}
		if err := save(name+".pem", certPEM); err != nil {
			return err
		}
		if err := save(name+"-key.pem", keyPEM); err != nil {
			return err
		}
	}
	return nil
}
