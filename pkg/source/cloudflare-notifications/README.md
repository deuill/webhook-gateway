# Cloudflare Notifications WebHook Source

This directory contains a source for WebHook events emitted by [Cloudflare Notifications][cloudflare-notifications].

## Configuration

This source does not accept any configuration options, and simply forwards the contents of the
`text` field into emitted messages.

[cloudflare-notifications]: https://developers.cloudflare.com/notifications/get-started/configure-webhooks/
