package app

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestServeReportsPortConflict(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	server := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           http.NewServeMux(),
		ReadHeaderTimeout: time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Serve(ctx, server); err == nil {
		t.Fatal("expected listen error")
	}
}
