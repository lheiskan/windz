package sse

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	if mgr.HasClients() {
		t.Error("New manager should have no clients")
	}

	if mgr.ClientCount() != 0 {
		t.Errorf("New manager should have 0 clients, got %d", mgr.ClientCount())
	}
}

func TestAddClient(t *testing.T) {
	mgr := NewManager()

	// Add first client
	chan1 := mgr.AddClient("client1")
	if chan1 == nil {
		t.Fatal("AddClient returned nil channel")
	}

	if !mgr.HasClients() {
		t.Error("Manager should have clients after adding one")
	}

	if mgr.ClientCount() != 1 {
		t.Errorf("Manager should have 1 client, got %d", mgr.ClientCount())
	}

	// Add second client
	chan2 := mgr.AddClient("client2")
	if chan2 == nil {
		t.Fatal("AddClient returned nil channel for second client")
	}

	if mgr.ClientCount() != 2 {
		t.Errorf("Manager should have 2 clients, got %d", mgr.ClientCount())
	}

	// Adding same client ID should replace the old one
	chan3 := mgr.AddClient("client1")
	if chan3 == nil {
		t.Fatal("AddClient returned nil channel for replaced client")
	}

	// Should still have 2 clients
	if mgr.ClientCount() != 2 {
		t.Errorf("Manager should still have 2 clients after replacement, got %d", mgr.ClientCount())
	}
}

func TestRemoveClient(t *testing.T) {
	mgr := NewManager()

	// Add clients
	mgr.AddClient("client1")
	mgr.AddClient("client2")
	mgr.AddClient("client3")

	if mgr.ClientCount() != 3 {
		t.Errorf("Manager should have 3 clients, got %d", mgr.ClientCount())
	}

	// Remove one client
	mgr.RemoveClient("client2")
	if mgr.ClientCount() != 2 {
		t.Errorf("Manager should have 2 clients after removal, got %d", mgr.ClientCount())
	}

	// Remove non-existent client (should not panic)
	mgr.RemoveClient("nonexistent")
	if mgr.ClientCount() != 2 {
		t.Errorf("Manager should still have 2 clients, got %d", mgr.ClientCount())
	}

	// Remove remaining clients
	mgr.RemoveClient("client1")
	mgr.RemoveClient("client3")

	if mgr.HasClients() {
		t.Error("Manager should have no clients after removing all")
	}
}

func TestBroadcast(t *testing.T) {
	mgr := NewManager()

	// Add clients
	chan1 := mgr.AddClient("client1")
	chan2 := mgr.AddClient("client2")
	chan3 := mgr.AddClient("client3")

	// Broadcast message
	testMsg := Message{
		Type:      "test",
		StationID: "station1",
		Data:      map[string]string{"test": "data"},
	}

	// Use goroutines to receive messages
	var wg sync.WaitGroup
	wg.Add(3)

	received := make([]Message, 3)

	go func() {
		defer wg.Done()
		select {
		case msg := <-chan1:
			received[0] = msg
		case <-time.After(1 * time.Second):
			t.Error("Client 1 timeout waiting for message")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case msg := <-chan2:
			received[1] = msg
		case <-time.After(1 * time.Second):
			t.Error("Client 2 timeout waiting for message")
		}
	}()

	go func() {
		defer wg.Done()
		select {
		case msg := <-chan3:
			received[2] = msg
		case <-time.After(1 * time.Second):
			t.Error("Client 3 timeout waiting for message")
		}
	}()

	// Broadcast the message
	mgr.Broadcast(testMsg)

	// Wait for all clients to receive
	wg.Wait()

	// Verify all clients received the message
	for i, msg := range received {
		if msg.Type != "test" {
			t.Errorf("Client %d received wrong message type: %s", i+1, msg.Type)
		}
		if msg.StationID != "station1" {
			t.Errorf("Client %d received wrong station ID: %s", i+1, msg.StationID)
		}
		if msg.ID == 0 {
			t.Errorf("Client %d received message with no ID", i+1)
		}
		if msg.Timestamp.IsZero() {
			t.Errorf("Client %d received message with no timestamp", i+1)
		}
	}
}

func TestBroadcastToNoClients(t *testing.T) {
	mgr := NewManager()

	// Broadcast to no clients (should not panic)
	testMsg := Message{
		Type: "test",
		Data: "test data",
	}

	mgr.Broadcast(testMsg) // Should complete without error
}

func TestConcurrentOperations(t *testing.T) {
	mgr := NewManager()
	var wg sync.WaitGroup

	// Concurrently add clients
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client%d", id)
			mgr.AddClient(clientID)
		}(i)
	}

	// Concurrently broadcast messages
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			msg := Message{
				Type: fmt.Sprintf("test%d", id),
				Data: id,
			}
			mgr.Broadcast(msg)
		}(i)
	}

	// Concurrently remove clients
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			clientID := fmt.Sprintf("client%d", id)
			time.Sleep(10 * time.Millisecond) // Small delay to ensure clients are added first
			mgr.RemoveClient(clientID)
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Should have 5 clients remaining (10 added - 5 removed)
	expectedClients := 5
	if mgr.ClientCount() != expectedClients {
		t.Errorf("Expected %d clients, got %d", expectedClients, mgr.ClientCount())
	}
}

// @vibe: 🤖 -- ai
