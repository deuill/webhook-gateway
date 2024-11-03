# IRC Message Destination

This directory contains a destination for events received from any source defined for a gateway, and
pushed to one or more users or channels on a specific IRC network.

## Configuration

```toml
[gateway.destination.irc]
server = "irc.example.chat"
port = 6697
nick = "example"
username = "exampleuser"
password = "password"
recipients = "otheruser #somechannel #some-other-channel"
no-tls = false
no-verify-tls = false
```

The `server` option defines the server to connect to, and is required to be a valid hostname or
IP address.

The `port` option defines the port number to use in connecting to the IRC server, and will
default to 6667 if `no-tls` is set to `true`, or 6697 otherwise.

The `nick` option defines the primary nickname to use when connecting to the IRC server; if taken, a
number of fallbacks (typically, the nickname chosen plus a number of trailing underscores) will be chosen.

The `username` option defines who to authenticate as with the IRC; by default, this will be the same
value as the primary username, but some IRC servers require an email address or similar.

The `password` option defines the credentials used when authenticating as the given `username`; it is
not required that this is set, and most IRC servers will allow for unauthenticated connections,
assuming the nickname chosen hasn't been reserved.

The `recipients` option defines a space-separated list of user or user nicknames or channel names to
distribute messages to. Both regular channels (prefixed with `#`) and local channels (prefixed with
`&`) are supported; any other value will assumed to be a user nickname.

The `no-tls` option disables TLS and attempts to connect via a plain-text socket, if set to `true`.

The `no-verify-tls` option will allow *any* certificate to be accepted as valid for outgoing TLS
connections if set to `true`; it obviously doesn't affect anything if `no-tls` is also set to
`true`. Setting this can be dangerous, and is mainly used for local development.
