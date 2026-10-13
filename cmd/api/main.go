package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdulmuti12/api-point-go/internal/application"
	"github.com/abdulmuti12/api-point-go/internal/platform"
	"github.com/abdulmuti12/api-point-go/internal/repository/gormrepo"
	"github.com/abdulmuti12/api-point-go/internal/transport/httpapi"
)

func main() {
	if e := run(); e != nil {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	c, e := platform.LoadConfig()
	if e != nil {
		return e
	}
	db, e := platform.OpenDatabase(c)
	if e != nil {
		return e
	}
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	defer sqlDB.Close()
	issues, e := gormrepo.CheckSchema(db)
	if e != nil {
		return e
	}
	if len(issues) > 0 {
		return fmt.Errorf("business schema needs attention; run cmd/check-schema (%d issues)", len(issues))
	}
	// Security tables are an explicit prerequisite. Business tables are never
	// auto-migrated during startup, preventing accidental production schema edits.
	for _, table := range []string{"point_go_tokens", "point_go_revoked_tokens", "point_go_locks"} {
		if !db.Migrator().HasTable(table) {
			return errors.New("run cmd/migrate to create Go security tables before starting the API")
		}
	}
	if e = os.MkdirAll(c.Storage, 0750); e != nil {
		return e
	}
	service := &application.Service{Store: gormrepo.New(db), Tokens: platform.Tokens{Secret: c.Secret, Issuer: c.AppURL, TTL: c.TokenTTL, AcceptLegacy: c.AcceptLegacyTokens}, Passwords: platform.Passwords{}, Mail: platform.SMTPMailer{Config: c}, Google: platform.Google{ClientID: c.GoogleClientID, ClientSecret: c.GoogleClientSecret}, AppURL: c.AppURL, FrontendURL: c.FrontendURL}
	server := &http.Server{Addr: c.Address, Handler: httpapi.New(service, c), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() { slog.Info("API listening", "address", c.Address); errCh <- server.ListenAndServe() }()
	select {
	case e := <-errCh:
		if !errors.Is(e, http.ErrServerClosed) {
			return e
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
	return nil
}
