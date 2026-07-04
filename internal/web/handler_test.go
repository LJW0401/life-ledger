package web

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestNewHandlerRequiresIndex(t *testing.T) {
	_, err := newHandler(fstest.MapFS{
		".keep": {Data: []byte("placeholder")},
	})
	if err == nil || !strings.Contains(err.Error(), "missing index.html") {
		t.Fatalf("expected missing index error, got %v", err)
	}
}
