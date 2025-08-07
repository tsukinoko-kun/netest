package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/tsukinoko-kun/netest/internal/db"
	"github.com/tsukinoko-kun/netest/internal/networktest"
	"net"
	"net/http"
)

type Server struct {
	ln       net.Listener
	srv      *http.Server
	mux      *http.ServeMux
	database *db.DB
}

func New(addr string, database *db.DB) (*Server, error) {
	server := &Server{database: database}

	mux := http.NewServeMux()
	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/api", server.apiHandler)
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		_ = srv.Serve(ln)
		_ = ln.Close()
	}()

	server.ln = ln
	server.srv = srv
	server.mux = mux

	return server, nil
}

//go:embed index.html
var indexHtml []byte

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(indexHtml)
}

type apiResponse struct {
	TestResults []db.HistoryEntry[networktest.TestResults] `json:"test_results"`
}

func (s *Server) apiHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := db.RetrieveAll[networktest.TestResults](s.database)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to retrieve test results: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	je := json.NewEncoder(w)
	je.SetIndent("", "  ")
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
