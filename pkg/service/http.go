package service

import (
	// Standard library.
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// HTTP represents a basic HTTP server, currently only able to serve plain HTTP requests.
type HTTP struct {
	// Configuration options.
	host string
	port string

	// Internal fields.
	server *http.Server
}

// NewHTTP instantiates a new HTTP server for the given options.
func NewHTTP(options ...HTTPOption) (*HTTP, error) {
	var h = HTTP{
		server: &http.Server{
			Handler:           http.NewServeMux(),
			ReadHeaderTimeout: time.Second * 1,
		},
	}

	for _, fn := range options {
		if err := fn(&h); err != nil {
			return nil, err
		}
	}

	return &h, nil
}

// A HTTPOption represents any configuration option available to the HTTP server.
type HTTPOption func(*HTTP) error

// WithHTTPHost sets the given hostname for an HTTP server.
func WithHTTPHost(host string) HTTPOption {
	return func(h *HTTP) error {
		h.host = host
		return nil
	}
}

// WithHTTPPort sets the given port number for an HTTP server.
func WithHTTPPort(port string) HTTPOption {
	return func(h *HTTP) error {
		h.port = port
		return nil
	}
}

// Handle registers the given [http.HandlerFunc] for the given HTTP method and path pattern. Any
// errors caught will be returned verbatim; check documentation for [http.ServeMux] for more
// information.
func (h *HTTP) Handle(pattern string, handler http.HandlerFunc) (err error) {
	defer func() {
		if v := recover(); v == nil {
			return
		} else if s, ok := v.(string); ok {
			err = errors.New(s)
		} else if e, ok := v.(error); ok {
			err = e
		} else {
			err = errors.New("unknown error in setting up HTTP handler")
		}
	}()

	h.server.Handler.(*http.ServeMux).HandleFunc(pattern, handler)
	return err
}

// Init ensures the HTTP server is configured correctly, and listens on the configured hostname and
// port, ensuring that the listener is correctly set up before returning.
// TODO: Ensure context cancellation causes graceful shutdown.
func (h *HTTP) Init(ctx context.Context) error {
	// Start internal TCP socket listener.
	ln, err := net.Listen("tcp", net.JoinHostPort(h.host, h.port))
	if err != nil {
		return err
	}

	// Wait for HTTP server to begin listening for connections before returning, in order to ensure
	// that subsequent calls to receiver functions can complete successfully.
	wait := make(chan error, 1)
	go func() {
		h.server.BaseContext = func(net.Listener) context.Context {
			// Assume no error happened if we're at the point where we're spawning a context for the
			// underlying listener.
			wait <- nil
			return ctx
		}
		wait <- h.server.Serve(ln)
	}()

	return <-wait
}
