package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unifiedcommunicationsprotocol/server/internal/admin"
)

// handleAdminSubscribe upgrades the connection to WebSocket and streams admin events.
func handleAdminSubscribe(hub *admin.AdminHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, use server-sent events (SSE) instead of WebSocket for simplicity
		// This avoids gorilla/websocket dependency while still providing real-time updates

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Subscribe to events
		subscriber := hub.Subscribe(r.RemoteAddr + "-" + time.Now().String())
		defer hub.Unsubscribe(subscriber.ID)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send initial heartbeat
		fmt.Fprintf(w, "data: {\"type\": \"connected\"}\n\n")
		flusher.Flush()

		// Stream events
		for {
			select {
			case event := <-subscriber.Events:
				// Marshal event to JSON
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}

				// Send as SSE
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()

			case <-r.Context().Done():
				// Client disconnected
				return
			}
		}
	}
}
