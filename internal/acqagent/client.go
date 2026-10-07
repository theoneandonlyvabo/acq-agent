package acqagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/audit"
)

// Result adalah ringkasan penarikan yang sudah terverifikasi hash-nya.
type Result struct {
	Bytes  uint64
	SHA256 string
}

// Pull menarik file dari node sumber, menyimpannya ke out, lalu memverifikasi
// hash SHA-256 sisi penarik terhadap hash sisi sumber.
// Koneksi masih tanpa TLS; mTLS dikerjakan di Fase 4.
func Pull(ctx context.Context, addr, src, out string, log *audit.Logger) (Result, error) {
	log.Info("pull.started", "penarikan dimulai", map[string]any{"from": addr, "src": src, "out": out})
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("pull.failed", "gagal terhubung", map[string]any{"from": addr, "err": err.Error()})
		return Result{}, err
	}
	defer conn.Close()

	stream, err := acqagentv1.NewAcqAgentClient(conn).Pull(ctx, &acqagentv1.PullRequest{Path: src})
	if err != nil {
		log.Error("pull.failed", "gagal meminta", map[string]any{"from": addr, "src": src, "err": err.Error()})
		return Result{}, err
	}
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
			f.Close()
			os.Remove(out)
			log.Error("pull.failed", "gagal menerima", map[string]any{"from": addr, "src": src, "err": recvErr.Error()})
			return Result{}, recvErr
		}
		if len(m.GetData()) > 0 {
			if _, writeErr := f.Write(m.GetData()); writeErr != nil {
				f.Close()
				os.Remove(out)
				log.Error("pull.failed", "gagal menulis hasil", map[string]any{"out": out, "err": writeErr.Error()})
				return Result{}, writeErr
			}
			sum.Write(m.GetData())
			written += uint64(len(m.GetData()))
		}
		if m.GetSourceHash() != nil {
			wantHash = m.GetSourceHash().GetSha256Hex()
			wantTotal = m.GetSourceHash().GetTotalBytes()
			hasHash = true
		}
	}
	if closeErr := f.Close(); closeErr != nil {
		os.Remove(out)
		log.Error("pull.failed", "gagal menutup hasil", map[string]any{"out": out, "err": closeErr.Error()})
		return Result{}, closeErr
	}
	if !hasHash {
		os.Remove(out)
		err := errors.New("hash sumber tidak diterima")
		log.Error("pull.failed", "stream putus tanpa hash", map[string]any{"from": addr, "src": src})
		return Result{}, err
	}
	got := hex.EncodeToString(sum.Sum(nil))
	if got != wantHash || written != wantTotal {
		os.Remove(out)
		err := fmt.Errorf("hash tidak cocok: dapat %s (%d byte), sumber %s (%d byte)", got, written, wantHash, wantTotal)
		log.Error("pull.failed", "verifikasi hash gagal", map[string]any{"from": addr, "src": src, "err": err.Error()})
		return Result{}, err
	}
	res := Result{Bytes: written, SHA256: got}
	log.Info("pull.completed", "penarikan selesai dan terverifikasi", map[string]any{"from": addr, "src": src, "out": out, "bytes": written, "sha256": got})
	return res, nil
}
