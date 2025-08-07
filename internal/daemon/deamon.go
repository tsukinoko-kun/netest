package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/tsukinoko-kun/netest/internal/history"
	mymath "github.com/tsukinoko-kun/netest/internal/math"
	"github.com/tsukinoko-kun/netest/internal/networktest"
	"github.com/tsukinoko-kun/netest/internal/server"
	myslices "github.com/tsukinoko-kun/netest/internal/slices"

	"github.com/kardianos/service"
)

var (
	svc    service.Service
	logger service.Logger
)

type (
	program struct {
		running atomic.Bool
		srv     *server.Server
	}
)

var Addr string

func (p *program) Start(s service.Service) error {
	_ = logger.Info("netest daemon starting")
	p.running.Store(true)
	go p.loop()
	if Addr != "" {
		srv, err := server.New(Addr)
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
		if err := networktest.Run(); err != nil {
			_ = logger.Error(err)
		}
		history.Summarize(joinHistoryEntry)
	}
}

func joinHistoryEntry(entries []history.HistoryEntry[networktest.TestResults]) history.HistoryEntry[networktest.TestResults] {
	if len(entries) == 0 {
		return history.HistoryEntry[networktest.TestResults]{
			networktest.TestResults{},
			time.Now(),
		}
	}
	medianTestResult := networktest.Median(myslices.Map(entries, history.ExtractValue))
	medianTime := mymath.MedianTime(myslices.Map(entries, history.ExtractTime))

	return history.HistoryEntry[networktest.TestResults]{
		medianTestResult,
		medianTime,
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
	return nil
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
