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
		case "init-config":
			flags := flag.NewFlagSet("life-ledger init-config", flag.ContinueOnError)
			flags.SetOutput(os.Stderr)
			defaultPath, err := defaultConfigPath()
			if err != nil {
				return err
			}
			configPath := flags.String("config", defaultPath, "path to create config.toml")
			cookieSecure := flags.Bool("cookie-secure", false, "set auth cookie Secure attribute")
			if err := flags.Parse(args[1:]); err != nil {
				return err
			}
			return initConfig(*configPath, *cookieSecure, os.Stdout)
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
	secret, err := randomBase64(32)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, secret)
	return nil
}

func initConfig(path string, cookieSecure bool, out io.Writer) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config already exists: %s", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect config: %w", err)
	}
	password, err := randomBase64(18)
	if err != nil {
		return err
	}
	secret, err := randomBase64(32)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	content := fmt.Sprintf(`[server]
host = "127.0.0.1"
port = 8080

[data]
dir = "./data"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = %q
session_secret = %q
session_days = 7

[security]
trusted_proxies = ["127.0.0.1"]
login_failure_window_minutes = 10
login_failure_limit = 5
login_lock_minutes = 15
cookie_secure = %t

[export]
timezone = "Asia/Shanghai"
max_upload_mb = 5
max_import_rows = 5000

[backup]
dir = "./backups"
`, string(hash), secret, cookieSecure)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	configDir := filepath.Dir(path)
	for _, dir := range []string{"data", "backups"} {
		if err := os.MkdirAll(filepath.Join(configDir, dir), 0o700); err != nil {
			return fmt.Errorf("create %s dir: %w", dir, err)
		}
	}
	fmt.Fprintf(out, "config created: %s\n", path)
	fmt.Fprintln(out, "username: admin")
	fmt.Fprintf(out, "password: %s\n", password)
	fmt.Fprintln(out, "save this password; it is not stored in plaintext")
	return nil
}

func randomBase64(size int) (string, error) {
	buf := make([]byte, 32)
	if size > 0 {
		buf = make([]byte, size)
	}
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random value: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
