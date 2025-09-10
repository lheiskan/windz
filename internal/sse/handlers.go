package sse

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// RegisterHandlers registers the SSE HTTP handlers
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
	mux.HandleFunc("/events", handleSSE(mgr))
}

// handleSSE handles Server-Sent Events connections
func handleSSE(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("X-Accel-Buffering", "no") // Disable Nginx buffering

		// Get client ID from request (could be from session, header, or generated)
		clientID := r.Header.Get("X-Client-Id")
		if clientID == "" {
			// Generate a unique client ID based on remote addr and timestamp
			clientID = fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
		}

		// Register client with manager
		messageChan := mgr.AddClient(clientID)

		// Ensure cleanup on disconnect
		defer mgr.RemoveClient(clientID)

		// Create flusher for immediate sending
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send initial connection message
		initialMsg := Message{
			ID:   time.Now().Unix(),
			Type: "connected",
			Data: map[string]any{"client_id": clientID},
		}
		if err := writeSSEMessage(w, initialMsg); err != nil {
			log.Printf("Error sending initial SSE message: %v", err)
			return
		}
		flusher.Flush()

		// Notify manager about new client (will trigger initial data send)
		mgr.NotifyClientConnected(clientID)

		// Create a ticker for keepalive messages
		keepaliveTicker := time.NewTicker(30 * time.Second)
		defer keepaliveTicker.Stop()

		// Main event loop
		for {
			select {
			case <-r.Context().Done():
				// Client disconnected
				log.Printf("SSE client %s disconnected", clientID)
				return

			case msg, ok := <-messageChan:
				if !ok {
					// Channel closed, manager terminated the connection
					return
				}

				// Send the message
				if err := writeSSEMessage(w, msg); err != nil {
					log.Printf("Error sending SSE message to %s: %v", clientID, err)
					return
				}
				flusher.Flush()

			case <-keepaliveTicker.C:
				// Send keepalive comment to prevent timeout
				if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
					log.Printf("Error sending keepalive to %s: %v", clientID, err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

// writeSSEMessage writes a message in SSE format
func writeSSEMessage(w http.ResponseWriter, msg Message) error {
	// Set ID and timestamp if not set
	if msg.ID == 0 {
		msg.ID = time.Now().Unix()
	}

	// Write SSE formatted message
	if _, err := fmt.Fprintf(w, "id: %d\n", msg.ID); err != nil {
		return err
	}

	if msg.Type != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", msg.Type); err != nil {
			return err
		}
	}

	// Marshal data to JSON
	if msg.Data != nil {
		jsonData, err := json.Marshal(msg.Data)
		if err != nil {
			return fmt.Errorf("error marshaling SSE data: %w", err)
		}
		if _, err := fmt.Fprintf(w, "data: %s\n", jsonData); err != nil {
			return err
		}
	} else {
		// Even without data, send empty data field for valid SSE
		if _, err := fmt.Fprintf(w, "data: {}\n"); err != nil {
			return err
		}
	}

	// End of message
	if _, err := fmt.Fprintf(w, "\n"); err != nil {
		return err
	}

	return nil
}

// WriteSSEMessage is a helper function to write SSE messages (exported for external use)
func WriteSSEMessage(w http.ResponseWriter, msg Message) error {
	return writeSSEMessage(w, msg)
}

// @vibe: 🤖 -- ai
