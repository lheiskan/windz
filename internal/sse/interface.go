package sse

import (
	"time"
)

// Manager defines the interface for SSE client management
type Manager interface {
	// AddClient registers a new SSE client and returns a channel for messages
	AddClient(clientID string) <-chan Message

	// RemoveClient unregisters an SSE client
	RemoveClient(clientID string)

	// HasClients returns true if there are any connected clients
	HasClients() bool

	// ClientCount returns the number of connected clients
	ClientCount() int

	// Broadcast sends a message to all connected clients
	Broadcast(message Message)
}

// Message represents a Server-Sent Event message
type Message struct {
	ID        int64       `json:"id"`   // Unix timestamp for SSE message ID
	Type      string      `json:"type"` // Message type (data, status, etc.)
	StationID string      `json:"station_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// @vibe: 🤖 -- ai
