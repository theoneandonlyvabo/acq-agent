// Package acqagent berisi logika penarikan file image antar node.
package acqagent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/audit"
)

// ChunkSize adalah ukuran tiap potongan stream: 1 MiB (kunci Fase 0).
const ChunkSize = 1 << 20

// Server melayani penarikan file image dari node ini.
type Server struct {
	acqagentv1.UnimplementedAcqAgentServer
	Log *audit.Logger
	// NodeID adalah CommonName sertifikat server; kosong bila tanpa TLS.
	NodeID string
	// Allow membatasi pasangan penarik; nil berarti tanpa pembatasan (dev).
	Allow *Allowlist
}

// rejectPhysicalPath menolak path device fisik; hanya file biasa yang boleh dibuka.
func rejectPhysicalPath(path string) error {
	lower := strings.ToLower(path)
	if strings.HasPrefix(path, "/dev/") ||
		strings.HasPrefix(lower, `\\.\`) ||
		strings.HasPrefix(lower, `\\?\`) {
		return fmt.Errorf("jalur device fisik dilarang: %s", path)
	}
	return nil
}

// Pull mengalirkan isi file ke penarik lalu menutup dengan hash SHA-256 sumber.
func (s *Server) Pull(req *acqagentv1.PullRequest, stream acqagentv1.AcqAgent_PullServer) error {
	peerAddr := "unknown"
	if p, ok := peer.FromContext(stream.Context()); ok && p.Addr != nil {
		peerAddr = p.Addr.String()
	}
	if err := rejectPhysicalPath(req.GetPath()); err != nil {
		s.Log.Error("pull.failed", "jalur ditolak", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return err
	}
	if s.Allow != nil {
		clientCN := ClientCNFromContext(stream.Context())
		if clientCN == "" {
			s.Log.Error("pull.failed", "identitas takdikenal", map[string]any{"path": req.GetPath(), "peer": peerAddr})
			return status.Error(codes.PermissionDenied, "allowlist butuh koneksi mTLS")
		}
		if !s.Allow.Allows(s.NodeID, clientCN) {
			s.Log.Error("pull.failed", "tidak diizinkan allowlist", map[string]any{"path": req.GetPath(), "peer": peerAddr, "client": clientCN})
			return status.Errorf(codes.PermissionDenied, "pasangan %s dan %s tidak diizinkan", s.NodeID, clientCN)
		}
	}
	// #nosec G304 -- path dari request sudah lolos tolak jalur device dan dibuka read-only.
	f, err := os.Open(req.GetPath()) // selalu read-only di sisi sumber
	if err != nil {
		s.Log.Error("pull.failed", "gagal membuka sumber", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return err
	}
	defer func() { _ = f.Close() }()
	s.Log.Info("pull.started", "penarikan dimulai", map[string]any{"path": req.GetPath(), "peer": peerAddr})

	sum := sha256.New()
	buf := make([]byte, ChunkSize)
	var offset uint64
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&acqagentv1.PullChunk{Data: buf[:n], Offset: offset}); sendErr != nil {
				s.Log.Error("pull.failed", "gagal mengirim chunk", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": sendErr.Error()})
				return sendErr
			}
			sum.Write(buf[:n])
			offset += uint64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			s.Log.Error("pull.failed", "gagal membaca sumber", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": readErr.Error()})
			return readErr
		}
	}
	hashHex := hex.EncodeToString(sum.Sum(nil))
	if sendErr := stream.Send(&acqagentv1.PullChunk{
		Offset:     offset,
		SourceHash: &acqagentv1.SourceHash{Sha256Hex: hashHex, TotalBytes: offset},
	}); sendErr != nil {
		s.Log.Error("pull.failed", "gagal mengirim hash", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": sendErr.Error()})
		return sendErr
	}
	s.Log.Info("pull.completed", "penarikan selesai", map[string]any{"path": req.GetPath(), "peer": peerAddr, "bytes": offset, "sha256": hashHex})
	return nil
}
