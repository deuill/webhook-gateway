# WebHook Gateway

WebHook Gateway processes events emitted by sources such as Grafana, Cloudflare Notifications, etc.
into destinations such as XMPP; it is generally intended as a way of forwarding alerts into chat for
platforms with no direct support.

Currently, the following sources are supported:

  - [Grafana AlertManager][grafana-alertmanager]
  - [Cloudflare Notifications][cloudflare-notifications]

The currently supported destinations include:

- [XMPP][xmpp]
- [IRC][irc]

These can be used interchangeably and matched as needed.

## Building and Installing

Installing `webhook-gateway` locally requires that you have Go installed, at a minimum. To install,
simply run the following command:

```sh
$ go install go.deuill.org/webhook-gateway/cmd/webhook-gateway@latest
```

The `webhook-gateway` binary should be placed in your `$GOBIN` path.

## Deployment

### Containers

Local deployments can be made by manually building the `webhook-gateway` binary with a
locally installed Go toolchain; in most cases, you'll likely want to use the pre-built container
images with Podman or Docker.

Doing so requires that you first create a valid `config.toml` configuration file, which you can copy
over from the `config.example.toml` template, and mounting it into
`/var/lib/webhook-gateway/config.toml`:

```sh
$ cp config.example.toml config.toml # Probably need to edit the file as well.
$ docker run --rm -v ./config.toml:/var/lib/webhook-gateway/config.toml docker.io/deuill/webhook-gateway:latest
```

### Google Cloud

Google Cloud Run provides a platform for running arbitrary containers with a generous [free
tier][gcloud-run-free-tier]; assuming you've already set up your Google Cloud account and have the
`gcloud` command-line tools installed locally, you can deploy into Google Cloud Run via service
definitions shipped here. First, set up a secret containing your entire `config.toml` file:

```sh
$ gcloud secrets create webhook-gateway-config --data-file=config.toml
```

Then, create the service from the included `gcloud-service.yaml` file:

```sh
$ gcloud run services replace gcloud-service.yaml --region=europe-west1
```

This should return an un-authenticated URL you can use from within your event sources as-is.

## Configuration

Configuration is made entirely via a single TOML file, a full example of which can be found
[here][toml-config]. In general, providing a configuration file is mandatory as options
don't (generally) have defaults set; only a number of options are required, though. The following
sections are available:

### `http`

```toml
[http]
host = "localhost"
port = "8080"
```

The `host` option determines which hostname/IP address the service will listen for HTTP requests on.
Set this to `0.0.0.0` if you want to listen on *all* interfaces (including those potentially
connected to the public Internet).

The `port` option determines which port number will be used to listen for HTTP requests on.

### `gateway`

```toml
[[gateway]]
path = "POST /alerts"
secret = "foobar"
```

This section can be defined multiple times, for as many gateways as we want to set up, and has a
number of sub-sections, described below.

The `secret` option defines a secret value to use for authenticating incoming gateway requests,
typically checked against source-specific methods (e.g. the `Authorization` HTTP header). This value
is not required, and will have request be processed without explicit authentication if left empty --
it is *highly recommended* that you at least set the `path` to a sufficiently secure value instead,
in these cases.

The `path` option defines an absolute path, with an optional HTTP method prefix, to register for
processing incoming requests. Though this option isn't required -- leaving it empty will have the
gateway listen on `/<gateway-secret>` instead -- setting it is highly recommended. The value of this
option *must* be unique across gateway definitions.

### `gateway.source` and `gateway.destination`

```toml
[[gateway]]
secret = "foobar"

[gateway.source]
type = "grafana"

[gateway.destination]
type = "xmpp"
```

As per TOML syntax, these sections can also be defined inline with the `gateway` section, e.g.:

```toml
[[gateway]]
secret = "foobar"
source.type = "grafana"
destination.type = "xmpp"
```

Both ways of definition lead to the same result. These sections define the source and destination
for messages, as processed from incoming WebHook requests.

The `type` option defines which source or destination type will be used, and therefore what
additional configuration may need to be provided as a sub-section; for type names, check the
`source` and `destination` folders -- each sub-folder is a valid source and destination type,
respectively.

### `gateway.source.<type>` and `gateway.destination.<type>`

```toml
[[gateway]]
secret = "foobar"
source.type = "grafana"
destination.type = "xmpp"

[gateway.source.grafana]
template = "{{.Status}}: {{.Title}} is alerting!"

[gateway.destination.xmpp]
jid = "test@example.com"
password = "foobar"
```

These sections define source- and destination-specific configuration, with destination configuration
typically containing a number of required options. For more information on these options, check
README files in the respective source and destination directories.

## License

All code in this repository is covered by the terms of the MIT License, the full text of which can be found in the LICENSE file.

[grafana-alertmanager]: pkg/source/grafana/README.md
[cloudflare-notifications]: pkg/source/cloudflare-notifications/README.md
[xmpp]: pkg/destination/xmpp/README.md
[irc]: pkg/destination/irc/README.md
[toml-config]: https://github.com/deuill/webhook-gateway/blob/trunk/config.example.toml
