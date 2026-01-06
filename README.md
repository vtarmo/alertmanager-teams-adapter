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

## Helm

```sh
helm upgrade --install alertmanager-teams-adapter \
  charts/alertmanager-teams-adapter \
  --set env.teamsWebhookUrl="https://outlook.office.com/webhook/..."
```

Helm repo:

```sh
helm repo add alertmanager-teams-adapter https://vtarmo.github.io/alertmanager-teams-adapter
helm repo update
```

Deploy from repo:

```sh
helm upgrade --install alertmanager-teams-adapter \
  alertmanager-teams-adapter/alertmanager-teams-adapter \
  --set env.teamsWebhookUrl="https://outlook.office.com/webhook/..."
```

Using an existing Secret or ConfigMap (key `TEAMS_WEBHOOK_URL`):

```sh
helm upgrade --install alertmanager-teams-adapter \
  charts/alertmanager-teams-adapter \
  --set env.existingSecret=alertmanager-teams-adapter
```

Using a Secret with a custom key name:

```sh
helm upgrade --install alertmanager-teams-adapter \
  charts/alertmanager-teams-adapter \
  --set env.existingSecret=teams-workflow-webhook \
  --set env.existingSecretKey=url
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

## Power Automate setup

Steps:

1. Create an automated flow with the trigger **When an HTTP request is received**.
2. Add **Parse JSON** and use the Alertmanager schema (from your payload or
   `alertmanager-json-schema.json` if you have it).
3. Add **Post adaptive card in a chat or channel** (or Teams workflow action).
4. In the Adaptive Card JSON field, paste the contents of `adaptive-card.json`.
5. Save and copy the generated webhook URL into Alertmanager.

## Power Automate Adaptive Card template

Use this JSON in the Adaptive Card action after a `Parse JSON` step that parses
the Alertmanager webhook payload:

```json
{
  "type": "message",
  "attachments": [
    {
      "contentType": "application/vnd.microsoft.card.adaptive",
      "contentUrl": null,
      "content": {
        "$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
        "type": "AdaptiveCard",
        "version": "1.5",
        "msteams": {
          "width": "Full"
        },
        "body": [
          {
            "type": "TextBlock",
            "text": "@{coalesce(first(body('Parse_JSON')?['alerts'])?['annotations']?['summary'], first(body('Parse_JSON')?['alerts'])?['labels']?['alertname'], 'Alert')}",
            "weight": "Bolder",
            "size": "Large",
            "wrap": true
          },
          {
            "type": "FactSet",
            "facts": [
              {
                "title": "Severity",
                "value": "@{first(body('Parse_JSON')?['alerts'])?['labels']?['severity']}"
              },
              {
                "title": "Rule",
                "value": "@{first(body('Parse_JSON')?['alerts'])?['labels']?['alertname']}"
              },
              {
                "title": "Namespace",
                "value": "@{first(body('Parse_JSON')?['alerts'])?['labels']?['namespace']}"
              },
              {
                "title": "Starts at",
                "value": "@{first(body('Parse_JSON')?['alerts'])?['startsAt']}"
              },
              {
                "title": "Value",
                "value": "@{first(body('Parse_JSON')?['alerts'])?['annotations']?['value']}"
              }
            ]
          },
          {
            "type": "TextBlock",
            "text": "Summary: @{first(body('Parse_JSON')?['alerts'])?['annotations']?['description']}",
            "wrap": true,
            "spacing": "Medium"
          }
        ],
        "actions": [
          {
            "type": "Action.OpenUrl",
            "title": "Open Grafana",
            "url": "@{coalesce(first(body('Parse_JSON')?['alerts'])?['annotations']?['dashboard_url'], first(body('Parse_JSON')?['alerts'])?['generatorURL'], body('Parse_JSON')?['externalURL'])}"
          }
        ]
      }
    }
  ]
}
```

The same template is available in `adaptive-card.json`.
