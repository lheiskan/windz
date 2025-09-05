package observations

import (
	"context"
	"testing"
	"time"
	"windz/internal/sse"
	"windz/internal/stations"
)

// mockSSEManager implements a mock SSE manager for testing
type mockSSEManager struct {
	clients   int
	messages  []sse.Message
	hasClient bool
}

func (m *mockSSEManager) AddClient(clientID string) <-chan sse.Message {
	return make(<-chan sse.Message)
}

func (m *mockSSEManager) RemoveClient(clientID string) {}

func (m *mockSSEManager) HasClients() bool {
	return m.hasClient
}

func (m *mockSSEManager) ClientCount() int {
	return m.clients
}

func (m *mockSSEManager) Broadcast(message sse.Message) {
	m.messages = append(m.messages, message)
}

func TestNewManager(t *testing.T) {
	stationMgr := stations.NewManager()
	sseMgr := &mockSSEManager{}

	mgr := NewManager(stationMgr, sseMgr, "test_state.json", "test_wind.json", false)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	// Test that we can get empty observations initially
	observations := mgr.GetAllLatestObservations()
	if len(observations) != 0 {
		t.Errorf("Expected 0 observations initially, got %d", len(observations))
	}
}

func TestGetLatestObservation(t *testing.T) {
	stationMgr := stations.NewManager()
	sseMgr := &mockSSEManager{}

	mgr := NewManager(stationMgr, sseMgr, "test_state.json", "test_wind.json", false)

	// Test non-existing observation
	_, exists := mgr.GetLatestObservation("101023")
	if exists {
		t.Error("Expected non-existing observation to return false")
	}

	// Note: We can't easily test with real data without mocking the FMI API
	// These tests focus on the manager structure and basic functionality
}

func TestGetPollingState(t *testing.T) {
	stationMgr := stations.NewManager()
	sseMgr := &mockSSEManager{}

	mgr := NewManager(stationMgr, sseMgr, "test_state.json", "test_wind.json", false)

	// Test non-existing polling state
	_, exists := mgr.GetPollingState("101023")
	if exists {
		t.Error("Expected non-existing polling state to return false")
	}
}

func TestStartStop(t *testing.T) {
	stationMgr := stations.NewManager()
	sseMgr := &mockSSEManager{}

	mgr := NewManager(stationMgr, sseMgr, "test_state.json", "test_wind.json", false)

	// Test starting
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mgr.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	// Test double start should fail
	err = mgr.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running manager")
	}

	// Test stopping
	err = mgr.Stop()
	if err != nil {
		t.Fatalf("Failed to stop manager: %v", err)
	}

	// Test double stop should not fail
	err = mgr.Stop()
	if err != nil {
		t.Errorf("Unexpected error when stopping already stopped manager: %v", err)
	}
}

func TestGetAllLatestObservations(t *testing.T) {
	stationMgr := stations.NewManager()
	sseMgr := &mockSSEManager{}

	mgr := NewManager(stationMgr, sseMgr, "test_state.json", "test_wind.json", false)

	observations := mgr.GetAllLatestObservations()
	if observations == nil {
		t.Error("GetAllLatestObservations should never return nil")
	}

	if len(observations) != 0 {
		t.Error("Expected empty observations map initially")
	}

	// Test that we get a copy (not reference to internal map)
	observations["test"] = WindObservation{StationID: "test"}
	observationsAgain := mgr.GetAllLatestObservations()
	if _, exists := observationsAgain["test"]; exists {
		t.Error("GetAllLatestObservations should return copies, not references")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test interval formatting
	tests := []struct {
		interval time.Duration
		expected string
	}{
		{IntervalFast, "1m"},
		{IntervalMedium, "10m"},
		{IntervalSlow, "60m"},
		{IntervalUltraSlow, "24h"},
		{5 * time.Minute, "5m0s"},
	}

	for _, tt := range tests {
		result := formatInterval(tt.interval)
		if result != tt.expected {
			t.Errorf("formatInterval(%v) = %s, expected %s", tt.interval, result, tt.expected)
		}
	}

	// Test interval progression
	if getNextSlowerInterval(IntervalFast) != IntervalMedium {
		t.Error("Expected Fast -> Medium interval progression")
	}
	if getNextSlowerInterval(IntervalMedium) != IntervalSlow {
		t.Error("Expected Medium -> Slow interval progression")
	}
	if getNextSlowerInterval(IntervalSlow) != IntervalUltraSlow {
		t.Error("Expected Slow -> UltraSlow interval progression")
	}

	// Test standard interval rounding
	if roundToStandardInterval(30*time.Second) != IntervalFast {
		t.Error("Expected 30s to round to IntervalFast")
	}
	if roundToStandardInterval(5*time.Minute) != IntervalMedium {
		t.Error("Expected 5m to round to IntervalMedium")
	}
	if roundToStandardInterval(30*time.Minute) != IntervalSlow {
		t.Error("Expected 30m to round to IntervalSlow")
	}

	// Test effective polling interval
	if getEffectivePollingInterval(IntervalFast, false) != IntervalSlow {
		t.Error("Expected fallback to slow interval when no SSE clients")
	}
	if getEffectivePollingInterval(IntervalFast, true) != IntervalFast {
		t.Error("Expected to maintain interval when SSE clients present")
	}
}

func TestAnalyzeObservationIntervals(t *testing.T) {
	// Test with empty observations
	_, ok := analyzeObservationIntervals([]FMIWindObservation{})
	if ok {
		t.Error("Expected false for empty observations")
	}

	// Test with single observation
	single := []FMIWindObservation{{Timestamp: time.Now()}}
	_, ok = analyzeObservationIntervals(single)
	if ok {
		t.Error("Expected false for single observation")
	}

	// Test with regular intervals
	now := time.Now()
	regular := []FMIWindObservation{
		{Timestamp: now},
		{Timestamp: now.Add(10 * time.Minute)},
		{Timestamp: now.Add(20 * time.Minute)},
	}
	interval, ok := analyzeObservationIntervals(regular)
	if !ok {
		t.Error("Expected successful analysis of regular intervals")
	}
	if interval != 10*time.Minute {
		t.Errorf("Expected 10 minute interval, got %v", interval)
	}
}
