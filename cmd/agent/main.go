// Program utama node peer-to-peer acq-agent.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/acqagent"
	"acq-agent/internal/audit"
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
	fmt.Fprintln(os.Stderr, "  serve --addr :50051 --audit-log audit.log")
	fmt.Fprintln(os.Stderr, "  pull --from 127.0.0.1:50051 --src file.dd --out hasil.dd --audit-log audit.log")
}

// runServe menjalankan node ini sebagai sumber yang melayani penarikan.
func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":50051", "alamat listen, mis. :50051")
	logPath := fs.String("audit-log", "audit.log", "path file audit log")
	if err := fs.Parse(args); err != nil {
		return err
	}
	log, err := audit.Open(*logPath)
	if err != nil {
		return err
	}
	defer log.Close()
	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	server := grpc.NewServer() // tanpa TLS; mTLS dikerjakan di Fase 4
	acqagentv1.RegisterAcqAgentServer(server, &acqagent.Server{Log: log})
	log.Info("serve.started", "node mendengarkan", map[string]any{"addr": listener.Addr().String()})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || *src == "" || *out == "" {
		fs.Usage()
		return fmt.Errorf("flag --from, --src, dan --out wajib diisi")
	}
	log, err := audit.Open(*logPath)
	if err != nil {
		return err
	}
	defer log.Close()
	res, err := acqagent.Pull(context.Background(), *from, *src, *out, log)
	if err != nil {
		return err
	}
	fmt.Printf("OK bytes=%d sha256=%s\n", res.Bytes, res.SHA256)
	return nil
}
