package xmpp

import (
	// Standard library.
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	// Internal packages.
	"go.deuill.org/webhook-gateway/pkg/gateway"

	// Third-party packages.
	"mellium.im/sasl"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// DefaultAuthMechanisms represents the list of SASL authentication mechanisms this client is allowed
// to use in server authentication.
var defaultAuthMechanisms = []sasl.Mechanism{
	sasl.ScramSha256Plus,
	sasl.ScramSha256,
	sasl.ScramSha1Plus,
	sasl.ScramSha1,
	sasl.Plain,
}

// Message is an XMPP message containing simple body content.
type Message struct {
	stanza.Message
	Body string `xml:"body"`
}

// XMPP represents a client connection to an XMPP server, used for pushing notification messages as
// an authenticated user.
type XMPP struct {
	// Client credentials.
	clientJID      jid.JID // The JID to authenticate the XMPP client as.
	clientPassword string  // The password to use in client authentication.

	// Connection options.
	noTLS       bool // Whether to disable TLS connection to the XMPP server.
	noVerifyTLS bool // Whether or not TLS connections will be verified.
	useStartTLS bool // Whether or not connection will be allowed to be made over StartTLS.

	// Destination options.
	recipientJIDs []jid.JID // The list of JIDs to push notifications to.

	// Internal fields.
	session *xmpp.Session
}

// PushMessages writes the given messages to the destination JID configured for the XMPP session.
func (x *XMPP) PushMessages(ctx context.Context, messages ...*gateway.Message) error {
	for _, msg := range messages {
		for _, jid := range x.recipientJIDs {
			// Determine whether this is a direct or group-chat message from the resource part of
			// the JID, which is only set if the message was destined for a group-chat.
			var kind = stanza.ChatMessage
			if jid.Resourcepart() != "" {
				jid, kind = jid.Bare(), stanza.GroupChatMessage
			}

			var m = Message{
				Message: stanza.Message{To: jid, Type: kind},
				Body:    msg.Content,
			}

			// TODO: Log rather than return error here.
			if err := x.session.Encode(ctx, m); err != nil {
				return err
			}
		}
	}

	return nil
}

// Init ensures the [XMPP] destination is configured correctly, and initializes a client connection
// to the XMPP server pointed to by the client JID configured, authenticating if necessary.
func (x *XMPP) Init(ctx context.Context) error {
	if x.clientJID.Equal(jid.JID{}) {
		return fmt.Errorf("empty client JID given in configuration")
	} else if len(x.recipientJIDs) == 0 {
		return fmt.Errorf("no recipient JIDs given in configuration")
	}

	// Initialze connection according to configuration.
	var tlsConfig = &tls.Config{
		ServerName:         x.clientJID.Domain().String(),
		InsecureSkipVerify: x.noVerifyTLS, //nolint:gosec // This is required for local development.
	}

	var dialer = &dial.Dialer{NoTLS: x.noTLS}
	if x.noVerifyTLS {
		dialer.TLSConfig = tlsConfig
	}

	conn, err := dialer.Dial(ctx, "tcp", x.clientJID)
	if err != nil {
		return fmt.Errorf("connection to XMPP server failed: %w", err)
	}

	// Enable optional features and initialize client session, according to configuration.
	features := []xmpp.StreamFeature{xmpp.BindResource()}
	if x.useStartTLS {
		features = append(features, xmpp.StartTLS(tlsConfig))
	}
	if x.clientPassword != "" {
		features = append(features, xmpp.SASL("", x.clientPassword, defaultAuthMechanisms...))
	}

	session, err := xmpp.NewClientSession(ctx, x.clientJID, conn, features...)
	if err != nil {
		return fmt.Errorf("connection to XMPP server failed: %w", err)
	}

	x.session = session

	// Send initial presence to let the server know we want to send messages.
	err = x.session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return fmt.Errorf("setting initial XMPP presence failed: %w", err)
	}

	// Send available presences to recipients.
	for _, jid := range x.recipientJIDs {
		err = x.session.Send(ctx, stanza.Presence{Type: stanza.AvailablePresence, To: jid}.Wrap(nil))
		if err != nil {
			return fmt.Errorf("sending XMPP presence to %s failed: %w", jid, err)
		}
	}

	return nil
}

// UnmarshalTOML configures the [XMPP] destination based on values sourced from TOML configuration.
func (x *XMPP) UnmarshalTOML(data any) error {
	conf, ok := data.(map[string]any)
	if !ok {
		return nil
	}

	if v, ok := conf["jid"].(string); ok {
		id, err := jid.Parse(v)
		if err != nil {
			return fmt.Errorf("failed parsing client JID: %w", err)
		}

		x.clientJID = id
	}

	if v, ok := conf["password"].(string); ok {
		x.clientPassword = v
	}

	if v, ok := conf["recipients"].(string); ok {
		for _, r := range strings.Fields(v) {
			id, err := jid.Parse(r)
			if err != nil {
				return fmt.Errorf("failed parsing recipient JID: %w", err)
			}

			x.recipientJIDs = append(x.recipientJIDs, id)
		}
	}

	if v, ok := conf["no-tls"].(bool); ok {
		x.noTLS = v
	}
	if v, ok := conf["no-verify-tls"].(bool); ok {
		x.noVerifyTLS = v
	}
	if v, ok := conf["use-starttls"].(bool); ok {
		x.useStartTLS = v
	}

	return nil
}

func init() {
	initfn := func() gateway.Destination { return &XMPP{} }
	gateway.RegisterDestination("xmpp", initfn)
}
