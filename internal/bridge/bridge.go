// Package bridge menyajikan HTTP API lokal untuk visual layer (dev only).
// Hanya bind loopback dan tanpa auth; jangan ekspos ke jaringan.
package bridge

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"acq-agent/internal/acqagent"
	"acq-agent/internal/audit"
)

// Job adalah satu penarikan async.
type Job struct {
	ID         string `json:"id"`
	From       string `json:"from"`
	Src        string `json:"src"`
	Out        string `json:"out"`
	State      string `json:"state"`
	Bytes      uint64 `json:"bytes"`
	SHA256     string `json:"sha256"`
	WantSHA256 string `json:"wantSHA256"`
	Error      string `json:"error,omitempty"`
	StartedAt  string `json:"startedAt"`
}

// Server adalah HTTP API bridge.
type Server struct {
	log       *audit.Logger
	nodeID    string
	serveAddr string
	auditPath string
	certFile  string
	keyFile   string
	caFile    string
	useTLS    bool

	mu   sync.RWMutex
	jobs map[string]*Job
	seq  atomic.Uint64
	mux  *http.ServeMux
}

// New membuat bridge; file cert/key/ca kosong berarti pull plaintext.
func New(log *audit.Logger, nodeID, serveAddr, auditPath, certFile, keyFile, caFile string) *Server {
	b := &Server{
		log:       log,
		nodeID:    nodeID,
		serveAddr: serveAddr,
		auditPath: auditPath,
		certFile:  certFile,
		keyFile:   keyFile,
		caFile:    caFile,
		useTLS:    certFile != "",
		jobs:      map[string]*Job{},
		mux:       http.NewServeMux(),
	}
	b.mux.HandleFunc("GET /api/status", b.handleStatus)
	b.mux.HandleFunc("POST /api/pull", b.handlePull)
	b.mux.HandleFunc("GET /api/jobs/{id}", b.handleJob)
	b.mux.HandleFunc("GET /api/audit", b.handleAudit)
	b.mux.HandleFunc("GET /api/probe", b.handleProbe)
	b.mux.HandleFunc("GET /api/preview", b.handlePreview)
	return b
}

// Handler mengekspos rute untuk testing.
func (b *Server) Handler() http.Handler {
	return withCORS(b.mux)
}

// Start menjalankan bridge sampai ctx batal; menolak alamat non-loopback.
func (b *Server) Start(ctx context.Context, addr string) error {
	if err := ensureLoopback(addr); err != nil {
		return err
	}
	srv := &http.Server{Addr: addr, Handler: withCORS(b.mux), ReadHeaderTimeout: 5 * time.Second}
	// #nosec G118 -- ctx lifecycle sudah batal saat shutdown; butuh konteks segar.
	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	b.log.Info("bridge.started", "bridge HTTP mendengarkan", map[string]any{"addr": addr})
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		b.log.Error("bridge.failed", "bridge HTTP berhenti", map[string]any{"addr": addr, "err": err.Error()})
		return err
	}
	return nil
}

// ensureLoopback menolak alamat yang bukan loopback.
func ensureLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if host == "localhost" {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("bridge hanya boleh bind loopback: %s", addr)
	}
	return nil
}

var devOrigins = map[string]bool{
	"http://localhost:5173": true,
	"http://127.0.0.1:5173": true,
	"http://localhost:4173": true,
	"http://127.0.0.1:4173": true,
}

// isDevOrigin menerima origin http loopback apa pun (port bebas).
// Dev-only: vite bisa geser port bila :5173 terpakai; tetap loopback-only.
func isDevOrigin(origin string) bool {
	if devOrigins[origin] {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// withCORS membuka akses untuk dev server Vite lokal saja.
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); isDevOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func (b *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"nodeID": b.nodeID, "tls": b.useTLS, "addr": b.serveAddr})
}

func (b *Server) handlePull(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From string `json:"from"`
		Src  string `json:"src"`
		Out  string `json:"out"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "body bukan JSON valid")
		return
	}
	if req.From == "" || req.Src == "" || req.Out == "" {
		writeErr(w, http.StatusBadRequest, "from, src, dan out wajib diisi")
		return
	}
	id := "job-" + strconv.FormatUint(b.seq.Add(1), 10)
	job := &Job{
		ID:        id,
		From:      req.From,
		Src:       req.Src,
		Out:       req.Out,
		State:     "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}
	b.mu.Lock()
	b.jobs[id] = job
	b.mu.Unlock()
	// #nosec G118 -- job sengaja lepas dari request (202 Accepted); hidup mengikuti proses.
	go b.run(context.Background(), job)
	writeJSON(w, http.StatusAccepted, map[string]string{"jobID": id})
}

func (b *Server) run(ctx context.Context, job *Job) {
	fail := func(msg string) {
		b.mu.Lock()
		job.State = "failed"
		job.Error = msg
		b.mu.Unlock()
	}
	tlsCfg, err := b.clientTLSFor(job.From)
	if err != nil {
		fail(err.Error())
		return
	}
	res, err := acqagent.Pull(ctx, job.From, job.Src, job.Out, b.log, tlsCfg, func(written uint64) {
		b.mu.Lock()
		job.Bytes = written
		b.mu.Unlock()
	})
	if err != nil {
		fail(err.Error())
		return
	}
	b.mu.Lock()
	job.State = "done"
	job.Bytes = res.Bytes
	job.SHA256 = res.SHA256
	job.WantSHA256 = res.SHA256
	b.mu.Unlock()
}

func (b *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	b.mu.RLock()
	job, ok := b.jobs[r.PathValue("id")]
	b.mu.RUnlock()
	if !ok {
		writeErr(w, http.StatusNotFound, "job tidak dikenal")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (b *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		n, err := strconv.Atoi(q)
		if err != nil || n < 1 || n > 1000 {
			writeErr(w, http.StatusBadRequest, "limit harus 1 sampai 1000")
			return
		}
		limit = n
	}
	lines, err := tailLines(b.auditPath, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal membaca audit log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

// clientTLSFor membangun kredensial client untuk alamat tujuan;
// nil berarti plaintext (dev).
func (b *Server) clientTLSFor(from string) (*tls.Config, error) {
	if b.certFile == "" {
		return nil, nil
	}
	return acqagent.ClientTLS(b.certFile, b.keyFile, b.caFile, acqagent.ServerNameOf(from))
}

func (b *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	addr := r.URL.Query().Get("addr")
	if addr == "" {
		writeErr(w, http.StatusBadRequest, "addr wajib diisi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"addr": addr, "online": probeAddr(addr)})
}

// probeAddr mengecek keterjangkauan TCP dengan timeout singkat.
func probeAddr(addr string) bool {
	// #nosec G704 -- bridge dev-only bind loopback tanpa auth; pemanggil
	// sudah bisa dial langsung, probe tidak menambah privilege.
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func (b *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, path := q.Get("from"), q.Get("path")
	if from == "" || path == "" {
		writeErr(w, http.StatusBadRequest, "from dan path wajib diisi")
		return
	}
	var offset uint64
	if s := q.Get("offset"); s != "" {
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "offset harus angka")
			return
		}
		offset = n
	}
	limit := uint32(4096)
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 65536 {
			writeErr(w, http.StatusBadRequest, "limit harus 1 sampai 65536")
			return
		}
		limit = uint32(n)
	}
	tlsCfg, err := b.clientTLSFor(from)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "gagal memuat TLS")
		return
	}
	resp, err := acqagent.Preview(r.Context(), from, path, offset, limit, tlsCfg)
	if err != nil {
		writeErr(w, grpcToHTTP(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"offset":      resp.GetOffset(),
		"total_bytes": resp.GetTotalBytes(),
		"data_base64": base64.StdEncoding.EncodeToString(resp.GetData()),
	})
}

// grpcToHTTP memetakan galat gRPC ke status HTTP untuk UI.
func grpcToHTTP(err error) int {
	switch status.Code(err) {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusBadGateway
	}
}

// tailLines membaca N baris terakhir file.
func tailLines(path string, n int) ([]string, error) {
	// #nosec G304 -- path dari flag operator lokal, bukan dari jaringan.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(string(raw), "\n")
	if text == "" {
		return []string{}, nil
	}
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}
