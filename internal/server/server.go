package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/tsukinoko-kun/netest/internal/db"
)

type Server struct {
	ln  net.Listener
	srv *http.Server
	mux *http.ServeMux
}

func New(addr string) (*Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api", apiHandler)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	server := &Server{}

	server.ln = ln
	server.srv = srv
	server.mux = mux

	go func() {
		_ = srv.Serve(ln)
		_ = ln.Close()
	}()

	return server, nil
}

//go:embed index.html
var indexHtml []byte

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHtml)
}

type apiResponse struct {
	TestResults []db.HistoryEntry `json:"test_results"`
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := db.Direct()
	entries, err := q.GetAllHistoryEntries(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve test results: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	je := json.NewEncoder(w)
	_ = je.Encode(apiResponse{TestResults: entries})
}

func (s *Server) ListeningAddr() string {
	return s.ln.Addr().String()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.srv == nil {
		return nil
	}
	if err := s.srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	} else {
		s.srv = nil
		if s.ln != nil {
			_ = s.ln.Close()
			s.ln = nil
		}
		s.mux = nil
		return nil
	}
}
