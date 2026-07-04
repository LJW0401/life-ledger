// Package webdist embeds the React build output for the Go server binary.
package webdist

import "embed"

// Assets contains web/dist after npm run build.
//
//go:embed dist/*
var Assets embed.FS
