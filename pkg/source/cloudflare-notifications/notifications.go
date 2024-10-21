package cloudflare_notifications

import (
	// Standard library.
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"
)

// A Payload represents the full request payload for Cloudflare Notifications. By default,
// notification payloads contain only a simple text field, with not much configurability.
type Payload struct {
	Text string `json:"text"`
}

// Grafana represents a message source for Cloudflare Notifications. For information on how incoming
// requests are parsed, check the documentation for [Notifications.ParseHTTP].
type Notifications struct{}

// New instantiates an instance of a Cloudflare [Notifications] source.
func New() (*Notifications, error) {
	return &Notifications{}, nil
}

// ParseHTTP processes the given HTTP request, parsing a standard Cloudflare Notifications payload.
//
// Incoming requests will have the 'cf-webhook-auth' header checked for a correct token
// corresponding secret configured at the gateway level.
func (n *Notifications) ParseHTTP(r *http.Request) ([]*gateway.Message, error) {
	// Validate secret in HTTP headers.
	if secret := gateway.GetSecret(r.Context()); secret != "" {
		if h := r.Header.Get("cf-webhook-auth"); h == "" {
			return nil, fmt.Errorf("cf-webhook-auth header not found")
		} else if h != secret {
			return nil, fmt.Errorf("invalid authentication token")
		}
	}

	// Try to read payload from incoming request.
	buf, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading request body: %w", err)
	}

	defer r.Body.Close()
	var payload Payload

	if err := json.Unmarshal(buf, &payload); err != nil {
		return nil, fmt.Errorf("failed parsing request: %w", err)
	}

	var msg gateway.Message
	if payload.Text != "" {
		msg.Content = payload.Text
	} else {
		return nil, fmt.Errorf("no message content found")
	}

	return []*gateway.Message{&msg}, nil
}

// Init ensures the Cloudflare [Notifications] source is configured correctly, and initializes any
// sub-resources necessary for its operation.
func (n *Notifications) Init(_ context.Context) error {
	return nil
}

// Register Grafana source for gateway configuration.
func init() {
	initfn := func() gateway.Source { return &Notifications{} }
	gateway.RegisterSource("cloudflare-notifications", initfn)
}
