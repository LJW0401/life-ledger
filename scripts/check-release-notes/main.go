// check-release-notes validates release.md metadata before release builds start.
// A mismatched tag fails before CI spends time building binaries.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type config struct {
	filePath string
	tag      string
}

var releaseVersionRE = regexp.MustCompile(`(?m)^<!--\s*release-version:\s*([^\s]+)\s*-->$`)

func main() {
	cfg := config{}
	flag.StringVar(&cfg.filePath, "file", "release.md", "release notes file to validate")
	flag.StringVar(&cfg.tag, "tag", os.Getenv("GITHUB_REF_NAME"), "git tag expected in release notes")
	flag.Parse()

	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	if strings.TrimSpace(cfg.tag) == "" {
		return errors.New("release tag is empty; pass -tag or set GITHUB_REF_NAME")
	}

	body, err := os.ReadFile(cfg.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s is missing; create it from release.template.md before pushing a tag", cfg.filePath)
		}
		return fmt.Errorf("read %s: %w", cfg.filePath, err)
	}
	if strings.TrimSpace(string(body)) == "" {
		return fmt.Errorf("%s is empty; fill release notes before pushing tag %s", cfg.filePath, cfg.tag)
	}

	version, err := releaseVersion(body)
	if err != nil {
		return fmt.Errorf("validate %s: %w", cfg.filePath, err)
	}
	if version != cfg.tag {
		return fmt.Errorf("%s release-version %q does not match git tag %q", cfg.filePath, version, cfg.tag)
	}

	fmt.Printf("%s release-version matches git tag %s.\n", cfg.filePath, cfg.tag)
	return nil
}

func releaseVersion(body []byte) (string, error) {
	matches := releaseVersionRE.FindAllSubmatch(body, -1)
	switch len(matches) {
	case 0:
		return "", errors.New("missing metadata comment: <!-- release-version: vX.Y.Z -->")
	case 1:
		version := string(matches[0][1])
		if strings.TrimSpace(version) == "" {
			return "", errors.New("release-version metadata is empty")
		}
		return version, nil
	default:
		return "", errors.New("multiple release-version metadata comments found")
	}
}
