// Program utama node peer-to-peer acq-agent.
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/acqagent"
	"acq-agent/internal/audit"
	"acq-agent/internal/bridge"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "serve":
		err = runServe(os.Args[2:])
	case "pull":
		err = runPull(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "perintah takdikenal: %s\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gagal:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "pakai: agent <serve|pull> [flag]")
	fmt.Fprintln(os.Stderr, "  serve --addr :50051 --audit-log audit.log [--tls-cert c.pem --tls-key k.pem --tls-ca ca.pem --allowlist allow.json] [--http 127.0.0.1:8080]")
	fmt.Fprintln(os.Stderr, "  pull --from 127.0.0.1:50051 --src file.dd --out hasil.dd --audit-log audit.log [--tls-cert c.pem --tls-key k.pem --tls-ca ca.pem]")
}

// runServe menjalankan node ini sebagai sumber yang melayani penarikan.
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":50051", "alamat listen, mis. :50051")
	logPath := fs.String("audit-log", "audit.log", "path file audit log")
	tlsCert := fs.String("tls-cert", "", "sertifikat node (wajib bersama tls-key dan tls-ca untuk mTLS)")
	tlsKey := fs.String("tls-key", "", "kunci sertifikat node")
	tlsCA := fs.String("tls-ca", "", "CA untuk verifikasi lawan")
	allowPath := fs.String("allowlist", "", "file allowlist JSON (butuh mTLS)")
	httpAddr := fs.String("http", "", "alamat HTTP bridge lokal, mis. 127.0.0.1:8080 (kosong = mati, dev only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var opts []grpc.ServerOption
	nodeID := ""
	var allow *acqagent.Allowlist
	if *tlsCert != "" || *tlsKey != "" || *tlsCA != "" {
		if *tlsCert == "" || *tlsKey == "" || *tlsCA == "" {
			return fmt.Errorf("flag --tls-cert, --tls-key, dan --tls-ca harus diisi bersamaan")
		}
		var cn string
		var err error
		var opt grpc.ServerOption
		if opt, cn, err = acqagent.ServerTLS(*tlsCert, *tlsKey, *tlsCA); err != nil {
			return err
		}
		opts = append(opts, opt)
		nodeID = cn
	}
	if *allowPath != "" {
		if nodeID == "" {
			return fmt.Errorf("flag --allowlist butuh mTLS (--tls-cert, --tls-key, --tls-ca)")
		}
		var err error
		if allow, err = acqagent.LoadAllowlist(*allowPath); err != nil {
			return err
		}
	}
	log, err := audit.Open(*logPath)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if *httpAddr != "" {
		httpSrv := bridge.New(log, nodeID, listener.Addr().String(), *logPath, *tlsCert, *tlsKey, *tlsCA)
		go func() {
			if err := httpSrv.Start(ctx, *httpAddr); err != nil {
				fmt.Fprintln(os.Stderr, "bridge gagal:", err)
			}
		}()
	}
	server := grpc.NewServer(opts...)
	acqagentv1.RegisterAcqAgentServer(server, &acqagent.Server{Log: log, NodeID: nodeID, Allow: allow})
	log.Info("serve.started", "node mendengarkan", map[string]any{"addr": listener.Addr().String(), "tls": nodeID != "", "http": *httpAddr})
	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()
	return server.Serve(listener)
}

// runPull menarik satu file image dari node sumber ke file lokal.
func runPull(args []string) error {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	from := fs.String("from", "", "alamat node sumber, mis. 127.0.0.1:50051")
	src := fs.String("src", "", "path file image di sisi sumber")
	out := fs.String("out", "", "path file hasil di sisi penarik")
	logPath := fs.String("audit-log", "audit.log", "path file audit log")
	tlsCert := fs.String("tls-cert", "", "sertifikat node (wajib bersama tls-key dan tls-ca untuk mTLS)")
	tlsKey := fs.String("tls-key", "", "kunci sertifikat node")
	tlsCA := fs.String("tls-ca", "", "CA untuk verifikasi server")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *src == "" || *out == "" {
		fs.Usage()
		return fmt.Errorf("flag --from, --src, dan --out wajib diisi")
	}
	var tlsCfg *tls.Config
	if *tlsCert != "" || *tlsKey != "" || *tlsCA != "" {
		if *tlsCert == "" || *tlsKey == "" || *tlsCA == "" {
			return fmt.Errorf("flag --tls-cert, --tls-key, dan --tls-ca harus diisi bersamaan")
		}
		var err error
		if tlsCfg, err = acqagent.ClientTLS(*tlsCert, *tlsKey, *tlsCA, acqagent.ServerNameOf(*from)); err != nil {
			return err
		}
	}
	log, err := audit.Open(*logPath)
	if err != nil {
		return err
	}
	defer func() { _ = log.Close() }()
	res, err := acqagent.Pull(context.Background(), *from, *src, *out, log, tlsCfg, nil)
	if err != nil {
		return err
	}
	fmt.Printf("OK bytes=%d sha256=%s\n", res.Bytes, res.SHA256)
	return nil
}
