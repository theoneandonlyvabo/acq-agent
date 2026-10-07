// Package acqagent berisi logika penarikan file image antar node.
package acqagent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	acqagentv1 "acq-agent/gen/proto/acqagent/v1"
	"acq-agent/internal/audit"
)

// ChunkSize adalah ukuran tiap potongan stream: 1 MiB (kunci Fase 0).
const ChunkSize = 1 << 20

// Batas pratinjau per panggilan: default 4 KiB, maks 64 KiB.
const (
	DefaultPreviewBytes = 4 << 10
	MaxPreviewBytes     = 64 << 10
)

// Server melayani penarikan file image dari node ini.
type Server struct {
	acqagentv1.UnimplementedAcqAgentServer
	Log *audit.Logger
	// NodeID adalah CommonName sertifikat server; kosong bila tanpa TLS.
	NodeID string
	// Allow membatasi pasangan penarik; nil berarti tanpa pembatasan (dev).
	Allow *Allowlist
	// ServeDir membatasi file yang boleh disajikan; kosong = tanpa batas (dev).
	ServeDir string
}

// checkServeDir memastikan path berada di bawah ServeDir.
// ServeDir kosong berarti tanpa pembatasan (dev).
func (s *Server) checkServeDir(path string) error {
	if s.ServeDir == "" {
		return nil
	}
	dir, err := filepath.EvalSymlinks(s.ServeDir)
	if err != nil {
		return fmt.Errorf("direktori serve tidak valid: %s", s.ServeDir)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	target, err := filepath.EvalSymlinks(abs)
	if err != nil {
		target = abs // file belum tentu ada; cek secara leksikal
	}
	rel, err := filepath.Rel(dir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return status.Errorf(codes.PermissionDenied, "path di luar direktori serve: %s", path)
	}
	return nil
}

// rejectPhysicalPath menolak path device fisik; hanya file biasa yang boleh dibuka.
func rejectPhysicalPath(path string) error {
	lower := strings.ToLower(path)
	if strings.HasPrefix(path, "/dev/") ||
		strings.HasPrefix(lower, `\\.\`) ||
		strings.HasPrefix(lower, `\\?\`) {
		return status.Errorf(codes.PermissionDenied, "jalur device fisik dilarang: %s", path)
	}
	return nil
}

// authorize menjalankan semua guard baca (device, direktori, allowlist)
// dan mengembalikan alamat peer untuk audit.
func (s *Server) authorize(ctx context.Context, path string) (string, error) {
	peerAddr := "unknown"
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		peerAddr = p.Addr.String()
	}
	if err := rejectPhysicalPath(path); err != nil {
		return peerAddr, err
	}
	if err := s.checkServeDir(path); err != nil {
		return peerAddr, err
	}
	if s.Allow != nil {
		clientCN := ClientCNFromContext(ctx)
		if clientCN == "" {
			return peerAddr, status.Error(codes.PermissionDenied, "allowlist butuh koneksi mTLS")
		}
		if !s.Allow.Allows(s.NodeID, clientCN) {
			return peerAddr, status.Errorf(codes.PermissionDenied, "pasangan %s dan %s tidak diizinkan", s.NodeID, clientCN)
		}
	}
	return peerAddr, nil
}

// Pull mengalirkan isi file ke penarik lalu menutup dengan hash SHA-256 sumber.
func (s *Server) Pull(req *acqagentv1.PullRequest, stream acqagentv1.AcqAgent_PullServer) error {
	peerAddr, err := s.authorize(stream.Context(), req.GetPath())
	if err != nil {
		s.Log.Error("pull.failed", "ditolak", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return err
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

// Preview mengembalikan sebagian isi file untuk pratinjau (read-only).
// Sukses tidak diaudit (berisik per halaman); penolakan tetap diaudit.
func (s *Server) Preview(ctx context.Context, req *acqagentv1.PreviewRequest) (*acqagentv1.PreviewResponse, error) {
	peerAddr, err := s.authorize(ctx, req.GetPath())
	if err != nil {
		s.Log.Error("preview.failed", "ditolak", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return nil, err
	}
	limit := req.GetLimit()
	if limit == 0 {
		limit = DefaultPreviewBytes
	}
	if limit > MaxPreviewBytes {
		limit = MaxPreviewBytes
	}
	// #nosec G304 -- path sudah lolos guard device dan direktori, dibuka read-only.
	f, err := os.Open(req.GetPath())
	if err != nil {
		s.Log.Error("preview.failed", "gagal membuka sumber", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return nil, err
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		s.Log.Error("preview.failed", "gagal stat sumber", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
		return nil, err
	}
	if fi.Size() < 0 {
		s.Log.Error("preview.failed", "ukuran tidak valid", map[string]any{"path": req.GetPath(), "peer": peerAddr})
		return nil, fmt.Errorf("ukuran file tidak valid: %s", req.GetPath())
	}
	// #nosec G115 -- size >= 0 dicek di atas, aman ke uint64.
	total := uint64(fi.Size())
	offset := req.GetOffset()
	var data []byte
	if offset < total {
		n := min(uint64(limit), total-offset)
		data = make([]byte, n)
		// #nosec G115 -- offset < total <= MaxInt64 by construction, aman ke int64.
		if _, err := f.ReadAt(data, int64(offset)); err != nil && err != io.EOF {
			s.Log.Error("preview.failed", "gagal membaca sumber", map[string]any{"path": req.GetPath(), "peer": peerAddr, "err": err.Error()})
			return nil, err
		}
	}
	return &acqagentv1.PreviewResponse{Offset: offset, Data: data, TotalBytes: total}, nil
}
