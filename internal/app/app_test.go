package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
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

func TestRoutesSetSecurityHeaders(t *testing.T) {
	handler := routes(http.NotFoundHandler(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	for key, value := range map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"Referrer-Policy":        "same-origin",
	} {
		if got := rec.Header().Get(key); got != value {
			t.Fatalf("%s = %q, want %q", key, got, value)
		}
	}
}
