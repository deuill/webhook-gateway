package main

import (
	// Standard library.
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	// Internal packages.
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
	logLevel   = flag.String("log-level", "info", "The minimum log level to process logs under")
)

func logger() (*slog.Logger, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		return nil, err
	}

	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})), nil
}

func main() {
	// Ensure command-line flags are processed.
	flag.Parse()

	// Set up service-wide logging.
	log, err := logger()
	if err != nil {
		slog.Error("Failed initializing logger", "error", err.Error())
		os.Exit(1)
	}

	// Wait for and perform graceful shut-down on specific signals.
	var ctx, _ = signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)

	// Initialize gateway server from configuration.
	if srv, err := service.New(service.WithLogger(log)); err != nil {
		log.Error("Failed initializing service", "error", err.Error())
		os.Exit(1)
	} else if _, err := toml.DecodeFile(*configPath, &srv); err != nil {
		log.Error("Failed to load TOML configuration", "error", err.Error())
		os.Exit(1)
	} else if err = srv.Init(ctx); err != nil {
		log.Error("Failed to initialize service", "error", err.Error())
		os.Exit(1)
	}

	log.Info("Waiting for incoming messages...")
	<-ctx.Done()
}
