package gateway

import (
	// Standard library.
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

// A Message represents a notification, as parsed in by a [Source], and provided to a [Destination].
type Message struct {
	Content string
}

// A Source represents any method of parsing a concrete [Message] from an incoming [http.Request].
// Sources typically have additional internal requirements for authentication and other metadata or
// configuration.
type Source interface {
	ParseHTTP(*http.Request) ([]*Message, error)
	Init(context.Context) error
}

// A Destination represents any method of pushing [Message] content to a (potentially) remote
// endpoint. Destinations typically require ways of interfacing with their remote endpoints, and
// thus require additional, source-specific configuration.
type Destination interface {
	PushMessages(context.Context, ...*Message) error
	Init(context.Context) error
}

// A Gateway represents a [Source]-to-[Destination] mapping, with some additional metadata related
// to authentication and HTTP pathing. Though most of the heavy lifting is done by downstream
// dependencies, [Gateway] instances do, at least, require that they have a unique path and/or secret
// configured for their correct operation.
type Gateway struct {
	// Configurable fields.
	path        string
	secret      string
	source      Source
	destination Destination

	// Internal fields.
	logger *slog.Logger
}

// New instantiates an instance of a [Gateway] type, for the options given.
func New(options ...Option) (*Gateway, error) {
	var g = Gateway{
		logger: slog.Default(),
	}

	for _, fn := range options {
		if err := fn(&g); err != nil {
			return nil, err
		}
	}

	return &g, nil
}

// A Option represents any configuration provided to new instances of [Gateway] types.
type Option func(*Gateway) error

// WithPath sets the HTTP request path (and optional HTTP method prefix) used for serving requests
// to the corresponding [Gateway]. If no explicit path is given, requests generally fall back to an
// implicit path based on the configured secret; see documentation for [WithSecret] and [Gateway.Init]
// for more.
func WithPath(path string) Option {
	return func(w *Gateway) error {
		w.path = path
		return nil
	}
}

// WithSecret sets the secret used for authenticating incoming requests to this [Gateway]. Noted that
// processing of authentication credentials against the given secret is generally the domain of
// [Source] instances, typically in [Source.ParseHTTP] calls.
func WithSecret(secret string) Option {
	return func(w *Gateway) error {
		w.secret = secret
		return nil
	}
}

// WithSource sets the given [Source] instance as the default source for the corresponding [Gateway].
func WithSource(src Source) Option {
	return func(w *Gateway) error {
		w.source = src
		return nil
	}
}

// WithDestination sets the given [Destination] instance as the default destination for the
// corresponding [Gateway].
func WithDestination(dest Destination) Option {
	return func(w *Gateway) error {
		w.destination = dest
		return nil
	}
}

// WithLogger sets the given [slog.Logger] as the log handler for the service and other downstream
// dependencies.
func WithLogger(l *slog.Logger) Option {
	return func(g *Gateway) error {
		g.logger = l
		return nil
	}
}

// Init ensures the [Service] is configured correctly, and initializes any sub-resources necessary
// for its operation. Specifically, any attached [Source] and [Destination] instances will have
// their 'Init' functions called, with any errors being returned immediately.
func (g *Gateway) Init(ctx context.Context) error {
	if g.path == "" && g.secret == "" {
		return fmt.Errorf("no path or secret found in gateway configuration")
	} else if g.path == "" {
		g.logger.Info("no path defined in gateway configuration, using gateway secret for path")
		g.path = "/" + g.secret
	}

	if g.source == nil {
		return fmt.Errorf("no source configuration found")
	} else if err := g.source.Init(ctx); err != nil {
		return fmt.Errorf("failed initializing source: %w", err)
	}

	if g.destination == nil {
		return fmt.Errorf("no destination configuration found")
	} else if err := g.destination.Init(ctx); err != nil {
		return fmt.Errorf("failed initializing destination: %w", err)
	}

	return nil
}

// HandleHTTP returns a HTTP path and corresponding [http.HandlerFunc] for the [Gateway], as
// configured. Most processing for requests happens as part of [Source.ParseHTTP] and
// [Destination.PushMessages], see the documentation for those functions for more information.
func (g *Gateway) HandleHTTP() (string, http.HandlerFunc) {
	h := func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(SetSecret(r.Context(), g.secret))
		if msg, err := g.source.ParseHTTP(r); err != nil || len(msg) == 0 {
			msg := fmt.Sprintf("failed processing incoming request: %s", err)
			http.Error(w, msg, http.StatusBadRequest)
			g.logger.Debug(msg)
			return
		} else if err = g.destination.PushMessages(r.Context(), msg...); err != nil {
			msg := fmt.Sprintf("failed pushing notification messages: %s", err)
			http.Error(w, msg, http.StatusBadRequest)
			g.logger.Debug(msg)
			return
		}
	}

	return g.path, h
}

// TomlUmarshaler is defined here to avoid having to import the `toml` package if we don't need to.
type tomlUnmarshaler interface {
	UnmarshalTOML(any) error
}

// UnmarshalTOML configures the [Gateway] based on values sourced from TOML configuration.
func (g *Gateway) UnmarshalTOML(data any) error {
	conf, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("no valid configuration keys found")
	}

	if v, ok := conf["secret"].(string); ok {
		g.secret = v
	}

	if v, ok := conf["path"].(string); ok {
		g.path = v
	}

	// Parse source and destination configuration.
	if v, ok := conf["source"].(map[string]any); ok {
		name, ok := v["type"].(string)
		if !ok || name == "" {
			return fmt.Errorf("empty or missing source type in gateway configuration")
		} else if _, ok = knownSources[name]; !ok {
			return fmt.Errorf("unknown source type '%s' given in gateway configuration", name)
		}

		g.source = knownSources[name]()
		if m, ok := g.source.(tomlUnmarshaler); ok {
			if v, ok = v[name].(map[string]any); ok {
				if err := m.UnmarshalTOML(v); err != nil {
					return fmt.Errorf("failed parsing configuration for source '%s': %w", name, err)
				}
			}
		}
	}

	if v, ok := conf["destination"].(map[string]any); ok {
		name, ok := v["type"].(string)
		if !ok || name == "" {
			return fmt.Errorf("empty or missing destination type in gateway configuration")
		} else if _, ok = knownDestinations[name]; !ok {
			return fmt.Errorf("unknown destination type '%s' given in gateway configuration", name)
		}

		g.destination = knownDestinations[name]()
		if m, ok := g.destination.(tomlUnmarshaler); ok {
			if v, ok = v[name].(map[string]any); ok {
				if err := m.UnmarshalTOML(v); err != nil {
					return fmt.Errorf("failed parsing configuration for destination '%s': %w", name, err)
				}
			}
		}
	}

	return nil
}

// ContextKey is a unique type for values stored in contexts.
type contextKey int

const (
	// SecretKey is a context key used for storing the gateway secret for use in downstream callers.
	secretKey contextKey = iota
)

// SetSecret returns the given [context.Context] with a secret value stored, as expected by future
// invocations of [GetSecret].
func SetSecret(ctx context.Context, secret string) context.Context {
	return context.WithValue(ctx, secretKey, secret)
}

// GetSecret returns the gateway secret stored in the request context, against which incoming
// requests should be checked (typically in [Source] implementations).
func GetSecret(ctx context.Context) string {
	if v, ok := ctx.Value(secretKey).(string); ok {
		return v
	}
	return ""
}

// List of registered sources and destinations, by name.
var (
	knownSources      = make(map[string]func() Source)
	knownDestinations = make(map[string]func() Destination)
)

// RegisterSource assigns the given name to the given [Source] instantiation function, allowing for
// this to be used as a message source in [Gateway] instances.
func RegisterSource(name string, src func() Source) {
	knownSources[name] = src
}

// RegisterDestination assigns the given name to the given [Destination] instantiation function,
// allowing for this to be used as a message destination in [Gateway] instances.
func RegisterDestination(name string, dest func() Destination) {
	knownDestinations[name] = dest
}
