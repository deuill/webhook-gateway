# Grafana WebHook Source

This directory contains a source for WebHook events emitted by [Grafana AlertManager][grafana-alertmanager].

## Configuration

```toml
[gateway.source.grafana]
template = "{{.Status}}: {{.Title}} is alerting!"
```

By default, no specific configuration is required, and messages will be emitted according to the
[server-defined template][grafana-notification-template]. However, gateways can override the
notification template, using Go's [`text/template` syntax][template-syntax].

For a list of available fields in templates, check the [reference][template-reference] documentation
and the `Payload` definition in the [`grafana.go`](grafana.go).

[grafana-alertmanager]: https://grafana.com/docs/grafana/latest/alerting/configure-notifications/manage-contact-points/integrations/webhook-notifier/
[grafana-notification-template]: https://grafana.com/docs/grafana/latest/alerting/configure-notifications/template-notifications/
[template-syntax]: https://pkg.go.dev/text/template
[template-reference]: https://grafana.com/docs/grafana/latest/alerting/configure-notifications/template-notifications/reference/
