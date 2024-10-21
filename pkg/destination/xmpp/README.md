# XMPP Message Destination

This directory contains a destination for events received from any source defined for a gateway, and
pushed to one or more 1-on-1 or MUC XMPP addresses.

## Configuration

```toml
[gateway.destination.xmpp]
jid = "test@example.com"
password = "password"
recipients = "foobar@example.com somegroup@chat.example.com/alerts"
no-tls = false
no-verify-tls = false
use-starttls = false
```

The `jid` option determines the server to connect to, as auto-discovered (typically using DNS), as
well as the user JID the service will connect as; it is required that this option is a non-empty, valid JID.

The `password` option determines the credentials used when authenticating as the given `jid`; it is
not required that this is set, but few XMPP servers will allow for connections without some form of
authentication.

The `recipients` option defines a space-separated list of user or MUC JIDs to distribute messages
to; MUC JIDs in particular can have a resource part set, which helps differentiate between different
gateways on the same service. It is required that this option contains at least one valid JID.

The `no-tls` option disables TLS and attempts to connect via a plain-text socket, if set to `true`.
Noted that most XMPP servers will not allow clients to authenticate if encryption is completely
turned off; try setting `use-starttls = true` if TLS is turned off and authenticated connections are
failing.

The `no-verify-tls` option will allow *any* certificate to be accepted as valid for outgoing TLS
connections if set to `true`; it obviously doesn't affect anything if `no-tls` is also set to
`true`. Setting this can be dangerous, and is mainly used for local development.

The `use-starttls` option will, if set to `true`, attempt to use StartTLS in making an encrypted
connection to the XMPP server. Some servers don't provide explicit TLS ports, but still expect
encrypted connections to be made; set this to `true` if you're having trouble connecting to or
authenticating with an XMPP server.
