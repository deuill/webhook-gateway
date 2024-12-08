package main

import (
	// Standard library.
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	// Internal packages.
	_ "go.deuill.org/webhook-gateway/pkg/destination/irc"
	_ "go.deuill.org/webhook-gateway/pkg/destination/xmpp"
	"go.deuill.org/webhook-gateway/pkg/service"
	_ "go.deuill.org/webhook-gateway/pkg/source/cloudflare-notifications"
	_ "go.deuill.org/webhook-gateway/pkg/source/grafana"

	// Third-party packages.
	"github.com/BurntSushi/toml"
)

// Global configuration.
var (
	configPath = flag.String("config", "config.toml", "Path to main configuration file, in TOML format.")
)

func main() {
	// Ensure command-line flags are processed.
	flag.Parse()

	// Wait for and perform graceful shut-down on specific signals.
	var ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	// Initialize gateway server from configuration.
	if srv, err := service.New(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed initializing service: %s\n", err)
		os.Exit(1)
	} else if _, err := toml.DecodeFile(*configPath, &srv); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load TOML configuration: %s\n", err)
		os.Exit(1)
	} else if err = srv.Init(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize service: %s\n", err)
		os.Exit(1)
	}

	<-ctx.Done()
}
