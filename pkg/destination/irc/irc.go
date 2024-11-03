package irc

import (
	// Standard library.
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"

	// Third-party packages.
	"github.com/lrstanley/girc"
)

const (
	// The amount of time to wait before reconnecting a client session after connection failures.
	defaultReconnectDelay = 30 * time.Second
)

type IRC struct {
	// Connection options.
	server string // The hostname to connect to, required.
	port   int64  // The port number to connect to, defaults to 6667 or 6697 if not set.
	nick   string // The nickname to use for the IRC client, required.

	// Authentication.
	username string // The username to use when connecting, uses the nickname value if not set.
	password string // The password to use for SASL PLAIN authentication.

	// Extended connection options.
	noTLS       bool // Whether to disable TLS connection to the IRC server.
	noVerifyTLS bool // Whether or not TLS connections will be verified.

	// Destination options.
	recipients []string // The list of user or channel names to push notifications to.

	// Internal fields.
	client *girc.Client
}

// PushMessages writes the given messages to the destination users and channels configured for the
// IRC client session.
func (r *IRC) PushMessages(ctx context.Context, messages ...*gateway.Message) error {
	for _, msg := range messages {
		for _, name := range r.recipients {
			r.client.Cmd.Message(name, msg.Content)
		}
	}

	return nil
}

// Init ensures the [IRC] destination is configured correctly, and initializes a client connection
// to the IRC server pointed to by the host and port configured, authenticating if necessary.
func (r *IRC) Init(ctx context.Context) error {
	// Check for valid configuration.
	if r.server == "" {
		return fmt.Errorf("empty server name given")
	} else if !girc.IsValidNick(r.nick) {
		return fmt.Errorf("empty or invalid nickname '%s' given", r.nick)
	} else if len(r.recipients) == 0 {
		return fmt.Errorf("list of recipients given is empty")
	}

	// Set defaults for empty values, where possible.
	if r.port == 0 && r.noTLS {
		r.port = 6667
	} else if r.port == 0 {
		r.port = 6697
	}

	if r.username == "" {
		r.username = r.nick
	}

	// Initialze client connection according to configuration.
	var clientConfig = girc.Config{
		Server: r.server,
		Port:   int(r.port),
		Nick:   r.nick,
		User:   r.username,
		SSL:    !r.noTLS,
		TLSConfig: &tls.Config{
			ServerName:         r.server,
			InsecureSkipVerify: r.noVerifyTLS, //nolint:gosec // This is required for local development.
		},
	}

	if r.password != "" {
		clientConfig.SASL = &girc.SASLPlain{
			User: r.username,
			Pass: r.password,
		}
	}

	// Connect client to all recipient channels automatically.
	r.client = girc.New(clientConfig)
	r.client.DisableTracking()

	r.client.Handlers.Add(girc.CONNECTED, func(c *girc.Client, e girc.Event) {
		r.client.Cmd.Back()
		for _, name := range r.recipients {
			if strings.HasPrefix(name, "#") || strings.HasPrefix(name, "&") {
				c.Cmd.Join(name)
			}
		}
	})

	r.client.Handlers.Add(girc.DISCONNECTED, func(c *girc.Client, e girc.Event) {
		r.client.Cmd.Away("Disconnected")
	})

	wait := make(chan error, 1)
	r.client.Handlers.Add(girc.INITIALIZED, func(c *girc.Client, e girc.Event) {
		wait <- nil
	})

	go func() {
		if err := r.client.Connect(); err != nil {
			wait <- err
		}

		for {
			// TODO: Log reconnection errors here.
			if err := r.client.Connect(); err != nil {
				time.Sleep(defaultReconnectDelay)
			} else {
				return
			}
		}
	}()

	return <-wait
}

// UnmarshalTOML configures the [IRC] destination based on values sourced from TOML configuration.
func (r *IRC) UnmarshalTOML(data any) error {
	conf, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	if v, ok := conf["server"].(string); ok {
		r.server = v
	}
	if v, ok := conf["port"].(int64); ok {
		r.port = v
	}
	if v, ok := conf["nick"].(string); ok {
		r.nick = v
	}
	if v, ok := conf["username"].(string); ok {
		r.username = v
	} else {
		r.username = r.nick
	}
	if v, ok := conf["password"].(string); ok {
		r.password = v
	}

	if v, ok := conf["recipients"].(string); ok {
		for _, name := range strings.Fields(v) {
			if strings.HasPrefix(name, "#") || strings.HasPrefix(name, "&") {
				if !girc.IsValidChannel(name) {
					return fmt.Errorf("invalid channel name '%s' in recipient list", name)
				}
			} else if !girc.IsValidNick(name) {
				return fmt.Errorf("invalid user nickname '%s' in recipient list", name)
			}

			r.recipients = append(r.recipients, name)
		}
	}

	if v, ok := conf["no-tls"].(bool); ok {
		r.noTLS = v
	}
	if v, ok := conf["no-verify-tls"].(bool); ok {
		r.noVerifyTLS = v
	}

	return nil
}

func init() {
	initfn := func() gateway.Destination { return &IRC{} }
	gateway.RegisterDestination("irc", initfn)
}
