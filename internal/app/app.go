// Package app assembles configuration, database, HTTP handlers, and lifecycle
// management. Business rules stay in lower-level packages.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"life-ledger/internal/config"
	"life-ledger/internal/db"
	appweb "life-ledger/internal/web"
)

// App owns process-level dependencies for the HTTP service.
type App struct {
	Config config.Config
	DB     *sql.DB
	Server *http.Server
}

// New loads configuration, opens SQLite, runs migrations, and prepares routes.
func New(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	conn, err := db.Open(cfg)
	if err != nil {
		return nil, err
	}

	webHandler, err := appweb.NewHandler()
	if err != nil {
		conn.Close()
		return nil, err
	}

	server := &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           routes(webHandler),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{Config: cfg, DB: conn, Server: server}, nil
}

// Run starts the HTTP server and shuts it down when ctx is canceled.
func (a *App) Run(ctx context.Context) error {
	defer a.DB.Close()
	log.Printf("life-ledger listening on http://%s", a.Config.Server.Address())
	return Serve(ctx, a.Server)
}

// Serve listens for HTTP traffic. It is separated for lifecycle tests.
func Serve(ctx context.Context, server *http.Server) error {
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", server.Addr, err)
	}

	errc := make(chan error, 1)
	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errc
	case err := <-errc:
		return err
	}
}

func routes(webHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"接口不存在","details":[]}}`))
	})
	mux.Handle("/", webHandler)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}
