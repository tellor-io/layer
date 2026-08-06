package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/metrics"
)

// Status is a snapshot of monitor readiness.
type Status struct {
	Ready               bool      `json:"ready"`
	NodeName            string    `json:"node_name"`
	LastProcessedHeight uint64    `json:"last_processed_height"`
	TipHeight           uint64    `json:"tip_height,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	LastError           string    `json:"last_error,omitempty"`
	CursorLag           uint64    `json:"cursor_lag,omitempty"`
}

// Tracker holds live health signals updated by the poller.
type Tracker struct {
	nodeName string

	mu                  sync.RWMutex
	lastProcessedHeight uint64
	tipHeight           uint64
	lastSuccessAt       time.Time
	lastError           string

	ready atomic.Bool
}

// NewTracker creates a health tracker.
func NewTracker(nodeName string) *Tracker {
	return &Tracker{nodeName: nodeName}
}

// SetReady marks the process as ready (initial cursor resolved, poller started).
func (t *Tracker) SetReady(ready bool) {
	t.ready.Store(ready)
}

// RecordSuccess updates state after a successful block process.
func (t *Tracker) RecordSuccess(processed, tip uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastProcessedHeight = processed
	t.tipHeight = tip
	t.lastSuccessAt = time.Now().UTC()
	t.lastError = ""
}

// RecordError records a non-fatal ingest error.
func (t *Tracker) RecordError(err error) {
	if err == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastError = err.Error()
}

// Snapshot returns the current status.
func (t *Tracker) Snapshot() Status {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var lag uint64
	if t.tipHeight > t.lastProcessedHeight {
		lag = t.tipHeight - t.lastProcessedHeight
	}

	return Status{
		Ready:               t.ready.Load(),
		NodeName:            t.nodeName,
		LastProcessedHeight: t.lastProcessedHeight,
		TipHeight:           t.tipHeight,
		LastSuccessAt:       t.lastSuccessAt,
		LastError:           t.lastError,
		CursorLag:           lag,
	}
}

// Server serves /healthz, /readyz, /status, and optionally /metrics.
type Server struct {
	addr     string
	tracker  *Tracker
	metrics  *metrics.Registry
	log      *slog.Logger
	server   *http.Server
}

// NewServer creates a health HTTP server. metrics may be nil to omit /metrics.
func NewServer(addr string, tracker *Tracker, reg *metrics.Registry, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		addr:    addr,
		tracker: tracker,
		metrics: reg,
		log:     log,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)
	mux.HandleFunc("/status", s.handleStatus)
	if reg != nil {
		mux.Handle("/metrics", reg.Handler())
	}

	s.server = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s
}

// Start begins listening. It returns when the server stops.
func (s *Server) Start() error {
	s.log.Info("health server listening", "addr", s.addr)
	err := s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	st := s.tracker.Snapshot()
	if !st.Ready || st.LastProcessedHeight == 0 {
		http.Error(w, "not ready\n", http.StatusServiceUnavailable)
		return
	}
	// Stale if no successful process in 2 minutes once ready.
	if !st.LastSuccessAt.IsZero() && time.Since(st.LastSuccessAt) > 2*time.Minute {
		http.Error(w, "stale\n", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ready\n"))
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.tracker.Snapshot())
}
