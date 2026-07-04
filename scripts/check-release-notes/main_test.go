package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPassesWhenVersionMatchesTag(t *testing.T) {
	path := writeReleaseNotes(t, "<!-- release-version: v1.2.3 -->\n\n# v1.2.3\n")

	if err := run(config{filePath: path, tag: "v1.2.3"}); err != nil {
		t.Fatalf("run() unexpected error: %v", err)
	}
}

func TestRunFailsWhenReleaseNotesMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release.md")

	err := run(config{filePath: path, tag: "v1.2.3"})
	if err == nil {
		t.Fatal("run() expected missing file error")
	}
	if !strings.Contains(err.Error(), "create it from release.template.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFailsWhenVersionDoesNotMatchTag(t *testing.T) {
	path := writeReleaseNotes(t, "<!-- release-version: v1.2.2 -->\n\n# v1.2.2\n")

	err := run(config{filePath: path, tag: "v1.2.3"})
	if err == nil {
		t.Fatal("run() expected mismatch error")
	}
	if !strings.Contains(err.Error(), `does not match git tag "v1.2.3"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFailsWhenVersionMetadataMissing(t *testing.T) {
	path := writeReleaseNotes(t, "# v1.2.3\n")

	err := run(config{filePath: path, tag: "v1.2.3"})
	if err == nil {
		t.Fatal("run() expected missing metadata error")
	}
	if !strings.Contains(err.Error(), "missing metadata comment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunFailsWhenMultipleVersionMarkersExist(t *testing.T) {
	path := writeReleaseNotes(t, "<!-- release-version: v1.2.3 -->\n<!-- release-version: v1.2.3 -->\n")

	err := run(config{filePath: path, tag: "v1.2.3"})
	if err == nil {
		t.Fatal("run() expected duplicate metadata error")
	}
	if !strings.Contains(err.Error(), "multiple release-version") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeReleaseNotes(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release.md")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write release notes fixture: %v", err)
	}
	return path
}
