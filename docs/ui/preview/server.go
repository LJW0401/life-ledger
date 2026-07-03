// Command server starts a local-only UI preview server with history fallback for the static prototype.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:4173", "preview server address")
	dir := flag.String("dir", "docs/ui/preview", "preview asset directory")
	flag.Parse()

	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("resolve preview directory: %v", err)
	}

	indexPath := filepath.Join(absDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		log.Fatalf("preview index not found at %s: %v", indexPath, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(r.URL.Path)
		assetPath := filepath.Join(absDir, strings.TrimPrefix(cleanPath, "/"))

		if hasFileExtension(cleanPath) {
			http.ServeFile(w, r, assetPath)
			return
		}

		http.ServeFile(w, r, indexPath)
	})

	fmt.Printf("life-ledger UI preview: http://%s/important-dates\n", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatalf("start preview server: %v", err)
	}
}

func hasFileExtension(path string) bool {
	base := filepath.Base(path)
	return strings.Contains(base, ".")
}
