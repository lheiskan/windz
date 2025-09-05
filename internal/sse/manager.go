package sse

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// manager implements the SSE Manager interface
type manager struct {
	clients map[string]chan Message
	mu      sync.RWMutex
}

// NewManager creates a new SSE manager instance
func NewManager() Manager {
	return &manager{
		clients: make(map[string]chan Message),
	}
}

// AddClient registers a new SSE client and returns a channel for messages
func (m *manager) AddClient(clientID string) <-chan Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing client if present
	if existingChan, exists := m.clients[clientID]; exists {
		close(existingChan)
		delete(m.clients, clientID)
	}

	// Create new channel with buffer to prevent blocking
	clientChan := make(chan Message, 100)
	m.clients[clientID] = clientChan

	log.Printf("SSE client connected: %s (total: %d)", clientID, len(m.clients))

	return clientChan
}

// RemoveClient unregisters an SSE client
func (m *manager) RemoveClient(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if clientChan, exists := m.clients[clientID]; exists {
		close(clientChan)
		delete(m.clients, clientID)
		log.Printf("SSE client disconnected: %s (remaining: %d)", clientID, len(m.clients))
	}
}

// HasClients returns true if there are any connected clients
func (m *manager) HasClients() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.clients) > 0
}

// ClientCount returns the number of connected clients
func (m *manager) ClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.clients)
}

// Broadcast sends a message to all connected clients
func (m *manager) Broadcast(message Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Set timestamp and ID if not already set
	if message.Timestamp.IsZero() {
		message.Timestamp = time.Now()
	}
	if message.ID == 0 {
		message.ID = message.Timestamp.Unix()
	}

	// Send to all clients
	for clientID, clientChan := range m.clients {
		select {
		case clientChan <- message:
			// Message sent successfully
		default:
			// Channel is full, client is slow or disconnected
			log.Printf("SSE client %s channel full, skipping message", clientID)
		}
	}

	if len(m.clients) > 0 {
		log.Printf("Broadcasted SSE message type=%s to %d clients", message.Type, len(m.clients))
	}
}

// formatSSEMessage formats a message for SSE protocol
func formatSSEMessage(msg Message) string {
	// Format SSE message according to protocol
	result := fmt.Sprintf("id: %d\n", msg.ID)
	result += fmt.Sprintf("event: %s\n", msg.Type)

	// Simple JSON encoding for data field
	// In production, you might want to use json.Marshal
	if msg.Data != nil {
		result += fmt.Sprintf("data: %v\n", msg.Data)
	}

	result += "\n" // End of message
	return result
}

// @vibe: 🤖 -- ai
