// cmd/server is the executable entry point. It parses the minimal CLI surface
// and delegates application assembly and runtime behavior to internal packages.
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"life-ledger/internal/app"
	"life-ledger/internal/backup"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "life-ledger: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "hash-password":
			return hashPassword(os.Stdin, os.Stdout)
		case "generate-secret":
			return generateSecret(os.Stdout)
		case "backup":
			flags := flag.NewFlagSet("life-ledger backup", flag.ContinueOnError)
			flags.SetOutput(os.Stderr)
			defaultPath, err := defaultConfigPath()
			if err != nil {
				return err
			}
			configPath := flags.String("config", defaultPath, "path to config.toml")
			if err := flags.Parse(args[1:]); err != nil {
				return err
			}
			return backup.Run(*configPath)
		}
	}

	flags := flag.NewFlagSet("life-ledger", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	defaultPath, err := defaultConfigPath()
	if err != nil {
		return err
	}
	configPath := flags.String("config", defaultPath, "path to config.toml")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unknown command %q", flags.Arg(0))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	application, err := app.New(*configPath)
	if err != nil {
		return err
	}
	return application.Run(ctx)
}

func defaultConfigPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return filepath.Join(filepath.Dir(executable), "config.toml"), nil
}

func hashPassword(in io.Reader, out io.Writer) error {
	fmt.Fprint(out, "Password: ")
	reader := bufio.NewReader(in)
	password, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	password = strings.TrimRight(password, "\r\n")
	if password == "" {
		return fmt.Errorf("password must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, string(hash))
	return nil
}

func generateSecret(out io.Writer) error {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Errorf("generate secret: %w", err)
	}
	fmt.Fprintln(out, base64.RawURLEncoding.EncodeToString(buf))
	return nil
}
