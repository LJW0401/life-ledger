// cmd/server is the executable entry point. It parses the minimal CLI surface
// and delegates application assembly and runtime behavior to internal packages.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"life-ledger/internal/app"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "life-ledger: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("life-ledger", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	configPath := flags.String("config", "config.toml", "path to config.toml")
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
