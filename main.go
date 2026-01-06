package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type alertPayload struct {
	ExternalURL string  `json:"externalURL"`
	Alerts      []alert `json:"alerts"`
}

type alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
	GeneratorURL string           `json:"generatorURL"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func coalesce(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func buildAdaptiveCard(payload alertPayload) map[string]any {
	var current alert
	if len(payload.Alerts) > 0 {
		current = payload.Alerts[0]
	}

	summary := coalesce(current.Annotations["summary"], current.Labels["alertname"], "Alert")
	description := current.Annotations["description"]
	severity := current.Labels["severity"]
	rule := current.Labels["alertname"]
	namespace := current.Labels["namespace"]
	startsAt := current.StartsAt
	value := current.Annotations["value"]
	dashboardURL := coalesce(current.Annotations["dashboard_url"], current.GeneratorURL, payload.ExternalURL)

	return map[string]any{
		"type": "message",
		"attachments": []any{
			map[string]any{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"contentUrl":  nil,
				"content": map[string]any{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.5",
					"msteams": map[string]any{
						"width": "Full",
					},
					"body": []any{
						map[string]any{
							"type":   "TextBlock",
							"text":   summary,
							"weight": "Bolder",
							"size":   "Large",
							"wrap":   true,
						},
						map[string]any{
							"type": "FactSet",
							"facts": []any{
								map[string]any{"title": "Severity", "value": severity},
								map[string]any{"title": "Rule", "value": rule},
								map[string]any{"title": "Namespace", "value": namespace},
								map[string]any{"title": "Starts at", "value": startsAt},
								map[string]any{"title": "Value", "value": value},
							},
						},
						map[string]any{
							"type":    "TextBlock",
							"text":    "Summary: " + description,
							"wrap":    true,
							"spacing": "Medium",
						},
					},
					"actions": []any{
						map[string]any{
							"type":  "Action.OpenUrl",
							"title": "Open Grafana",
							"url":   dashboardURL,
						},
					},
				},
			},
		},
	}
}

func main() {
	teamsWebhookURL := os.Getenv("TEAMS_WEBHOOK_URL")
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = "0.0.0.0"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	timeout := 10
	if rawTimeout := os.Getenv("REQUEST_TIMEOUT"); rawTimeout != "" {
		if parsed, err := strconv.Atoi(rawTimeout); err == nil {
			timeout = parsed
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})

	http.HandleFunc("/alertmanager", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if teamsWebhookURL == "" {
			http.Error(w, "TEAMS_WEBHOOK_URL is not set", http.StatusInternalServerError)
			return
		}

		var payload alertPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		card := buildAdaptiveCard(payload)
		body, err := json.Marshal(card)
		if err != nil {
			http.Error(w, "failed to build payload", http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequest(http.MethodPost, teamsWebhookURL, bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to create request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "teams webhook error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			http.Error(w, "teams webhook error", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})

	address := listenAddr + ":" + port
	log.Printf("starting adapter on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
