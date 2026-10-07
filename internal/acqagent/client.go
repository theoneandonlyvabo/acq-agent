package acqagent

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/audit"
)

// Preview mengambil sebagian isi file dari node sumber tanpa menarik
// seluruhnya. tlsCfg nil berarti plaintext (dev).
func Preview(ctx context.Context, addr, path string, offset uint64, limit uint32, tlsCfg *tls.Config) (*acqagentv1.PreviewResponse, error) {
	var conn *grpc.ClientConn
	var err error
	if tlsCfg != nil {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	return acqagentv1.NewAcqAgentClient(conn).Preview(ctx, &acqagentv1.PreviewRequest{Path: path, Offset: offset, Limit: limit})
}

// Result adalah ringkasan penarikan yang sudah terverifikasi hash-nya.
type Result struct {
	Bytes  uint64
	SHA256 string
}

// Pull menarik file dari node sumber, menyimpannya ke out, lalu memverifikasi
// hash SHA-256 sisi penarik terhadap hash sisi sumber.
// tlsCfg nil berarti plaintext (dev); isi untuk mTLS penuh.
// onProgress dipanggil tiap ada byte tertulis; nil berarti tanpa laporan.
func Pull(ctx context.Context, addr, src, out string, log *audit.Logger, tlsCfg *tls.Config, onProgress func(written uint64)) (Result, error) {
	log.Info("pull.started", "penarikan dimulai", map[string]any{"from": addr, "src": src, "out": out})
	var conn *grpc.ClientConn
	var err error
	if tlsCfg != nil {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	if err != nil {
		log.Error("pull.failed", "gagal terhubung", map[string]any{"from": addr, "err": err.Error()})
		return Result{}, err
	}
	defer func() { _ = conn.Close() }()

	stream, err := acqagentv1.NewAcqAgentClient(conn).Pull(ctx, &acqagentv1.PullRequest{Path: src})
	if err != nil {
		log.Error("pull.failed", "gagal meminta", map[string]any{"from": addr, "src": src, "err": err.Error()})
		return Result{}, err
	}
	// #nosec G304 -- path dari flag operator lokal, bukan dari jaringan.
	f, err := os.Create(out) // file hasil; sisi sumber selalu dibuka read-only
	if err != nil {
		log.Error("pull.failed", "gagal membuat file hasil", map[string]any{"out": out, "err": err.Error()})
		return Result{}, err
	}

	sum := sha256.New()
	var written uint64
	var wantHash string
	var wantTotal uint64
	hasHash := false
	for {
		m, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			_ = f.Close()
			_ = os.Remove(out)
			log.Error("pull.failed", "gagal menerima", map[string]any{"from": addr, "src": src, "err": recvErr.Error()})
			return Result{}, recvErr
		}
		if len(m.GetData()) > 0 {
			if _, writeErr := f.Write(m.GetData()); writeErr != nil {
				_ = f.Close()
				_ = os.Remove(out)
				log.Error("pull.failed", "gagal menulis hasil", map[string]any{"out": out, "err": writeErr.Error()})
				return Result{}, writeErr
			}
			sum.Write(m.GetData())
			written += uint64(len(m.GetData()))
			if onProgress != nil {
				onProgress(written)
			}
		}
		if m.GetSourceHash() != nil {
			wantHash = m.GetSourceHash().GetSha256Hex()
			wantTotal = m.GetSourceHash().GetTotalBytes()
			hasHash = true
		}
	}
	if closeErr := f.Close(); closeErr != nil {
		_ = os.Remove(out)
		log.Error("pull.failed", "gagal menutup hasil", map[string]any{"out": out, "err": closeErr.Error()})
		return Result{}, closeErr
	}
	if !hasHash {
		_ = os.Remove(out)
		err := errors.New("hash sumber tidak diterima")
		log.Error("pull.failed", "stream putus tanpa hash", map[string]any{"from": addr, "src": src})
		return Result{}, err
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if got != wantHash || written != wantTotal {
		_ = os.Remove(out)
		err := fmt.Errorf("hash tidak cocok: dapat %s (%d byte), sumber %s (%d byte)", got, written, wantHash, wantTotal)
		log.Error("pull.failed", "verifikasi hash gagal", map[string]any{"from": addr, "src": src, "err": err.Error()})
		return Result{}, err
	}
	res := Result{Bytes: written, SHA256: got}
	log.Info("pull.completed", "penarikan selesai dan terverifikasi", map[string]any{"from": addr, "src": src, "out": out, "bytes": written, "sha256": got})
	return res, nil
}
