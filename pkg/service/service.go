package service

import (
	// Standard library.
	"context"
	"fmt"
	"log/slog"
	"net/http"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"
)

// A Handler represents any type that's capable of attaching a given [http.HandlerFunc] against a
// specific path to server-wide request processing.
type Handler interface {
	Handle(string, http.HandlerFunc) error
	Init(context.Context) error
}

// A Service represents an abstract collection of [gateway.Gateway] configurations, against a request
// [Handler] used for fulfilling incoming requests.
type Service struct {
	gateway []*gateway.Gateway
	handler Handler
	logger  *slog.Logger
}

// New instantiates an instance of a [Service], for the options given.
func New(options ...Option) (*Service, error) {
	var s = Service{
		logger: slog.Default(),
	}

	for _, fn := range options {
		if err := fn(&s); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// A Option represents any configuration provided to new instances of [Service] types.
type Option func(*Service) error

// WithHandler sets the server-wide request handler, to be used for processing incoming requests.
func WithHandler(h Handler) Option {
	return func(s *Service) error {
		s.handler = h
		return nil
	}
}

// WithGateway adds the given [gateway.Gateway] to the list considered for request processing and
// [gateway.Message] forwarding.
func WithGateway(w *gateway.Gateway) Option {
	return func(s *Service) error {
		s.gateway = append(s.gateway, w)
		return nil
	}
}

// WithLogger sets the given [slog.Logger] as the log handler for the service and other downstream
// dependencies.
func WithLogger(l *slog.Logger) Option {
	return func(s *Service) error {
		s.logger = l
		return nil
	}
}

// Init ensures the [Service] is configured correctly, and initializes any sub-resources necessary
// for its operation. Specifically, any attached [gateway.Gateway] and [Handler] instances will have
// their 'Init' functions called, with any errors being returned immediately.
func (s *Service) Init(ctx context.Context) error {
	if s.handler == nil {
		return fmt.Errorf("no request handler configuration found")
	} else if len(s.gateway) == 0 {
		return fmt.Errorf("no gateway configuration found")
	}

	// Set up request handlers.
	if err := s.handler.Handle(s.handleHealth()); err != nil {
		return fmt.Errorf("failed setting up request handler for health-checks: %w", err)
	}

	for _, g := range s.gateway {
		if err := g.Init(ctx); err != nil {
			return fmt.Errorf("failed initializing gateway: %w", err)
		} else if err = s.handler.Handle(g.HandleHTTP()); err != nil {
			return fmt.Errorf("failed setting up request handler for gateway: %w", err)
		}
	}

	if err := s.handler.Init(ctx); err != nil {
		return fmt.Errorf("failed initializing request handler: %w", err)
	}

	return nil
}

// UnmarshalTOML configures the [Service] based on values sourced from TOML configuration.
func (s *Service) UnmarshalTOML(data any) error {
	conf, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("no valid configuration keys found")
	}

	// Process configuration for HTTP server.
	if v, ok := conf["http"].(map[string]any); ok {
		var options []HTTPOption
		if host, ok := v["host"].(string); ok {
			options = append(options, WithHTTPHost(host))
		}
		if port, ok := v["port"].(string); ok {
			options = append(options, WithHTTPPort(port))
		}

		h, err := NewHTTP(options...)
		if err != nil {
			return fmt.Errorf("failed initializing HTTP server: %w", err)
		}

		s.handler = h
	}

	// Process configuration for gateways.
	if v, ok := conf["gateway"].([]map[string]any); ok {
		for i := range v {
			g, err := gateway.New(gateway.WithLogger(s.logger))
			if err != nil {
				return fmt.Errorf("failed initializing gateway: %w", err)
			} else if err := g.UnmarshalTOML(v[i]); err != nil {
				return fmt.Errorf("failed parsing gateway configuration: %w", err)
			}

			s.gateway = append(s.gateway, g)
		}
	}

	return nil
}

// HandleHealth is an HTTP handler for health-checks.
func (s *Service) handleHealth() (string, http.HandlerFunc) {
	return "/_health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
}
