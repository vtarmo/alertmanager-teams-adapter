# Alertmanager Teams Adapter

Small HTTP service that receives Prometheus Alertmanager webhooks and forwards a
single alert as a Microsoft Teams Adaptive Card.

## Features

- Accepts Alertmanager webhook payloads on `/alertmanager`.
- Forwards a card to a Teams Incoming Webhook.
- Simple health endpoint at `/healthz`.

## Configuration

Environment variables:

- `TEAMS_WEBHOOK_URL` (required): Teams incoming webhook URL.
- `LISTEN_ADDR` (optional, default `0.0.0.0`): Bind address.
- `PORT` (optional, default `8080`): Bind port.
- `REQUEST_TIMEOUT` (optional, default `10`): Outbound webhook timeout in seconds.

## Run locally

```sh
export TEAMS_WEBHOOK_URL="https://outlook.office.com/webhook/..."
export PORT=8080
./alertmanager-teams-adapter
```

## Docker

```sh
docker build -t alertmanager-teams-adapter .
docker run --rm -p 8080:8080 \
  -e TEAMS_WEBHOOK_URL="https://outlook.office.com/webhook/..." \
  alertmanager-teams-adapter
```

## Alertmanager config example

```yaml
receivers:
  - name: teams
    webhook_configs:
      - url: http://alertmanager-teams-adapter:8080/alertmanager
        send_resolved: true
```

## Payload details

The adapter forwards the first alert in the Alertmanager payload. It uses:

- `annotations.summary` for the title (fallback: `labels.alertname`).
- `annotations.description` for the summary text.
- `labels.severity`, `labels.alertname`, `labels.namespace`, `startsAt`,
  `annotations.value` as facts.
- `annotations.dashboard_url`, then `generatorURL`, then `externalURL` as the
  "Open Grafana" link.
