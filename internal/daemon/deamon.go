package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/tsukinoko-kun/netest/internal/db"
	"github.com/tsukinoko-kun/netest/internal/networktest"
	"github.com/tsukinoko-kun/netest/internal/server"

	"github.com/kardianos/service"
)

var (
	svc    service.Service
	logger service.Logger
)

type (
	program struct {
		running  atomic.Bool
		srv      *server.Server
		database *db.DB
	}
)

var Addr string

func (p *program) Start(s service.Service) error {
	_ = logger.Info("netest daemon starting")

	// Initialize database
	database, err := db.New()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	p.database = database

	p.running.Store(true)
	go p.loop()
	if Addr != "" {
		srv, err := server.New(Addr, database)
		if err != nil {
			p.running.Store(false)
			return err
		}
		p.srv = srv
		_ = logger.Infof("Listening on %s", Addr)
	}
	return nil
}

func (p *program) loop() {
	for p.running.Load() {
		time.Sleep(30 * time.Minute)
		if err := networktest.Run(p.database); err != nil {
			_ = logger.Error(err)
		}
		if err := db.Summarize(p.database, joinNetworkTestResults); err != nil {
			_ = logger.Error(fmt.Errorf("failed to summarize results: %w", err))
		}
	}
}

func (p *program) Stop(s service.Service) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = logger.Info("netest daemon stopping")
	p.running.Store(false)
	if p.srv != nil {
		_ = p.srv.Stop(ctx)
		p.srv = nil
	}
	if p.database != nil {
		_ = p.database.Close()
		p.database = nil
	}
	return nil
}

// joinNetworkTestResults combines multiple network test results into one
func joinNetworkTestResults(entries []db.HistoryEntry[networktest.TestResults]) db.HistoryEntry[networktest.TestResults] {
	if len(entries) == 0 {
		return db.HistoryEntry[networktest.TestResults]{}
	}
	if len(entries) == 1 {
		return entries[0]
	}

	// Extract just the values for median calculation
	results := make([]networktest.TestResults, len(entries))
	for i, entry := range entries {
		results[i] = entry.Value
	}

	// Use the median function from networktest package
	medianResult := networktest.Median(results)

	// Use the median time from the entries
	return db.HistoryEntry[networktest.TestResults]{
		Value: medianResult,
		Time:  entries[len(entries)/2].Time,
	}
}

func initService() {
	var args []string
	if Addr != "" {
		args = []string{"daemon", "run", "--addr", Addr}
	} else {
		args = []string{"daemon", "run"}
	}
	cfg := &service.Config{
		Name:        "netestd",
		DisplayName: "NeTest Daemon",
		Description: "Network Test Daemon",
		Arguments:   args,
	}
	prg := &program{}
	s, err := service.New(prg, cfg)
	if err != nil {
		log.Fatal(err)
	}
	svc = s
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
}

func Install() {
	initService()
	if err := svc.Install(); err != nil {
		_ = logger.Error(err)
	}
}

func Uninstall() {
	initService()
	if err := svc.Uninstall(); err != nil {
		_ = logger.Error(err)
	}
}

func Start() {
	initService()
	err := svc.Start()
	if err != nil {
		_ = logger.Error(err)
	}
}

func Stop() {
	initService()
	err := svc.Stop()
	if err != nil {
		_ = logger.Error(err)
	}
}

func StatusString() (string, error) {
	initService()
	status, err := svc.Status()
	if err != nil {
		if errors.Is(err, service.ErrNotInstalled) {
			return "not installed", nil
		}
		return "", err
	}

	switch status {
	case service.StatusUnknown:
		return "unknown", nil
	case service.StatusRunning:
		return "running", nil
	case service.StatusStopped:
		return "stopped", nil
	default:
		return "", fmt.Errorf("unknown status: %d", status)
	}
}

func IsRunning() bool {
	initService()
	status, err := svc.Status()
	if err != nil {
		if errors.Is(err, service.ErrNotInstalled) {
			return false
		}
		return false
	}
	return status == service.StatusRunning
}

func Run() {
	initService()
	err := svc.Run()
	if err != nil {
		_ = logger.Error(err)
	}
}
