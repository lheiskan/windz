package observations

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"windz/internal/sse"
	"windz/internal/stations"
	"windz/pkg/fmi/observations"
)

// Polling intervals
const (
	IntervalFast      = 1 * time.Minute
	IntervalMedium    = 10 * time.Minute
	IntervalSlow      = 60 * time.Minute
	IntervalUltraSlow = 24 * time.Hour
)

// manager implements the Observations Manager interface
type manager struct {
	stationMgr   stations.Manager
	sseMgr       sse.Manager
	fmiClient    *http.Client
	stateFile    string
	windDataFile string
	debug        bool

	// State management
	windData      map[string]WindObservation
	windDataMutex sync.RWMutex

	pollingStates      map[string]*PollingState
	pollingStatesMutex sync.RWMutex

	// Polling control
	ctx       context.Context
	cancel    context.CancelFunc
	stopCh    chan struct{}
	isRunning bool
	runningMu sync.RWMutex
}

// NewManager creates a new observation manager instance
func NewManager(stationMgr stations.Manager, sseMgr sse.Manager, stateFile, windDataFile string, debug bool) Manager {
	return &manager{
		stationMgr:    stationMgr,
		sseMgr:        sseMgr,
		fmiClient:     &http.Client{Timeout: 60 * time.Second},
		stateFile:     stateFile,
		windDataFile:  windDataFile,
		debug:         debug,
		windData:      make(map[string]WindObservation),
		pollingStates: make(map[string]*PollingState),
		stopCh:        make(chan struct{}),
	}
}

// Start begins the observation polling process
func (m *manager) Start(ctx context.Context) error {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	if m.isRunning {
		return fmt.Errorf("observation manager is already running")
	}

	// Create context for this manager
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Load persistent state
	m.loadPollingStates()
	m.loadWindData()

	// Start polling scheduler
	go m.runPollingScheduler()

	m.isRunning = true
	log.Println("Observation manager started")

	return nil
}

// Stop stops the observation polling process
func (m *manager) Stop() error {
	m.runningMu.Lock()
	defer m.runningMu.Unlock()

	if !m.isRunning {
		return nil
	}

	if m.cancel != nil {
		m.cancel()
	}

	// Save state before stopping
	m.savePollingStates()
	m.saveWindData()

	// Signal stop and wait
	close(m.stopCh)
	m.isRunning = false

	log.Println("Observation manager stopped")
	return nil
}

// GetLatestObservation returns the latest observation for a specific station
func (m *manager) GetLatestObservation(stationID string) (WindObservation, bool) {
	m.windDataMutex.RLock()
	defer m.windDataMutex.RUnlock()

	obs, exists := m.windData[stationID]
	return obs, exists
}

// GetAllLatestObservations returns all latest observations indexed by station ID
func (m *manager) GetAllLatestObservations() map[string]WindObservation {
	m.windDataMutex.RLock()
	defer m.windDataMutex.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]WindObservation)
	for k, v := range m.windData {
		result[k] = v
	}
	return result
}

// GetPollingState returns the current polling state for a station
func (m *manager) GetPollingState(stationID string) (PollingState, bool) {
	m.pollingStatesMutex.RLock()
	defer m.pollingStatesMutex.RUnlock()

	state, exists := m.pollingStates[stationID]
	if !exists {
		return PollingState{}, false
	}

	// Return a copy
	return *state, true
}

// runPollingScheduler runs the main polling loop
func (m *manager) runPollingScheduler() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial poll
	m.pollDueStations()

	for {
		select {
		case <-ticker.C:
			m.pollDueStations()
		case <-m.ctx.Done():
			return
		case <-m.stopCh:
			return
		}
	}
}

// pollDueStations polls stations that are due for updates
func (m *manager) pollDueStations() {
	// Check if we have any SSE clients connected
	hasSSEClients := m.sseMgr.HasClients()

	// Get all stations from station manager
	allStations := m.stationMgr.GetAllStations()

	// Phase 1: Collect stations to poll (hold lock briefly)
	m.pollingStatesMutex.Lock()
	now := time.Now()
	toPoll := []PollingState{} // Values, not pointers

	// Determine which stations need polling
	for _, station := range allStations {
		state, exists := m.pollingStates[station.ID]

		if !exists {
			state = &PollingState{
				StationID:       station.ID,
				CurrentInterval: IntervalFast,
			}
			m.pollingStates[station.ID] = state
		}

		// Check if polling is due
		effectiveInterval := getEffectivePollingInterval(state.CurrentInterval, hasSSEClients)
		if now.Sub(state.LastPolled) >= effectiveInterval {
			// Append a copy of the state
			toPoll = append(toPoll, *state)
		}
	}
	m.pollingStatesMutex.Unlock() // Release lock early!

	if len(toPoll) == 0 {
		return
	}

	if m.debug {
		log.Printf("Polling %d stations", len(toPoll))
	}

	// Phase 2: Process polling without holding lock
	m.processBatchedPolling(toPoll, hasSSEClients)

	// Phase 3: Write back updated states
	m.pollingStatesMutex.Lock()
	for i := range toPoll {
		if currentState, exists := m.pollingStates[toPoll[i].StationID]; exists {
			// Update the actual state with polling results
			*currentState = toPoll[i]
		}
	}
	m.pollingStatesMutex.Unlock()
}

// processBatchedPolling handles the batched FMI API requests
func (m *manager) processBatchedPolling(toPoll []PollingState, hasSSEClients bool) {
	const maxBatchSize = 20
	endTime := time.Now()
	defaultStartTime := endTime.Add(-2 * time.Hour)

	// Group stations by effective time windows (store indices)
	timeWindowGroups := make(map[time.Time][]int)

	for i := range toPoll {
		effectiveStartTime := defaultStartTime
		if toPoll[i].LastObservation.After(defaultStartTime) {
			effectiveStartTime = toPoll[i].LastObservation.Add(time.Second).Truncate(time.Second)
		}
		timeWindowGroups[effectiveStartTime] = append(timeWindowGroups[effectiveStartTime], i)
	}

	// Process each time window group in batches
	for groupStartTime, indices := range timeWindowGroups {
		for start := 0; start < len(indices); start += maxBatchSize {
			end := start + maxBatchSize
			if end > len(indices) {
				end = len(indices)
			}
			batchIndices := indices[start:end]

			// Execute batch request
			stationIDs := make([]string, len(batchIndices))
			for j, idx := range batchIndices {
				stationIDs[j] = toPoll[idx].StationID
			}

			batchResults, err := m.fetchWindDataBatch(stationIDs, groupStartTime, endTime)
			if err != nil {
				log.Printf("Error fetching wind data for batch: %v", err)
				// Mark all stations as failed
				for _, idx := range batchIndices {
					m.updateFailedPollingState(&toPoll[idx])
				}
				continue
			}

			// Process results
			for _, idx := range batchIndices {
				state := &toPoll[idx] // Get pointer to modify the slice element
				observations, hasObservations := batchResults[state.StationID]
				if !hasObservations {
					observations = []FMIWindObservation{}
				}

				oldInterval := state.CurrentInterval
				latestObs, hasData := m.updatePollingState(state, observations)

				// Broadcast status change if interval changed
				if oldInterval != state.CurrentInterval {
					m.broadcastStatusUpdate(state)
				}

				// Update wind data and broadcast if we have new data
				if hasData {
					m.updateWindData(state.StationID, latestObs)
				}
			}
		}
	}
}

// fetchWindDataBatch fetches wind data for multiple stations
func (m *manager) fetchWindDataBatch(stationIDs []string, startTime, endTime time.Time) (map[string][]FMIWindObservation, error) {
	if len(stationIDs) == 0 {
		return make(map[string][]FMIWindObservation), nil
	}

	query := observations.NewQuery("https://opendata.fmi.fi/wfs", m.fmiClient)

	req := observations.Request{
		StartTime:  startTime,
		EndTime:    endTime,
		StationIDs: stationIDs,
		UseGzip:    true,
	}

	response, err := query.Execute(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wind data: %w", err)
	}

	// Convert to our format
	results := make(map[string][]FMIWindObservation)
	for _, station := range response.Stations {
		stationResults := make([]FMIWindObservation, 0, len(station.Observations))

		for _, obs := range station.Observations {
			windObs := FMIWindObservation{
				Timestamp: obs.Timestamp,
			}

			if obs.WindSpeed != nil {
				windObs.WindSpeed = *obs.WindSpeed
			}
			if obs.WindGust != nil {
				windObs.WindGust = *obs.WindGust
			}
			if obs.WindDirection != nil {
				windObs.WindDirection = *obs.WindDirection
			}

			// Only include valid observations
			if windObs.WindSpeed >= 0 && windObs.WindSpeed < 100 {
				stationResults = append(stationResults, windObs)
			}
		}

		results[station.StationID] = stationResults
	}

	return results, nil
}

// FMIWindObservation represents wind data from FMI API
type FMIWindObservation struct {
	Timestamp     time.Time
	WindSpeed     float64
	WindGust      float64
	WindDirection float64
}

// updatePollingState updates polling state based on observation results
func (m *manager) updatePollingState(state *PollingState, observations []FMIWindObservation) (FMIWindObservation, bool) {
	state.LastPolled = time.Now()
	state.TotalPolls++

	var lastObservation FMIWindObservation
	hadData := false

	if len(observations) > 0 {
		hadData = true
		lastObservation = observations[len(observations)-1]

		state.SuccessfulPolls++
		state.ConsecutiveMisses = 0
		state.LastObservation = lastObservation.Timestamp
		state.SuccessRate = float64(state.SuccessfulPolls) / float64(state.TotalPolls)

		// Adaptive interval adjustment
		if len(observations) >= 2 {
			if minInterval, ok := analyzeObservationIntervals(observations); ok && minInterval < state.CurrentInterval {
				state.CurrentInterval = roundToStandardInterval(minInterval)
			}
		}
	} else {
		state.ConsecutiveMisses++
		state.SuccessRate = float64(state.SuccessfulPolls) / float64(state.TotalPolls)

		if state.ConsecutiveMisses >= 2 {
			state.CurrentInterval = getNextSlowerInterval(state.CurrentInterval)
			state.ConsecutiveMisses = 0
		}
	}

	return lastObservation, hadData
}

// updateFailedPollingState marks a polling attempt as failed
func (m *manager) updateFailedPollingState(state *PollingState) {
	state.TotalPolls++
	state.ConsecutiveMisses++
	state.LastPolled = time.Now()

	if state.ConsecutiveMisses >= 3 {
		state.CurrentInterval = getNextSlowerInterval(state.CurrentInterval)
	}
}

// updateWindData updates wind data and broadcasts via SSE
func (m *manager) updateWindData(stationID string, obs FMIWindObservation) {
	station, exists := m.stationMgr.GetStation(stationID)
	if !exists {
		return
	}

	windObs := WindObservation{
		StationID:     stationID,
		StationName:   station.Name,
		Region:        station.Region,
		Timestamp:     obs.Timestamp,
		WindSpeed:     obs.WindSpeed,
		WindGust:      obs.WindGust,
		WindDirection: obs.WindDirection,
		UpdatedAt:     time.Now(),
	}

	m.windDataMutex.Lock()
	m.windData[stationID] = windObs
	m.windDataMutex.Unlock()

	// Broadcast to SSE clients
	m.sseMgr.Broadcast(sse.Message{
		Type:      "data",
		StationID: stationID,
		Data:      windObs,
		Timestamp: time.Now(),
	})
}

// broadcastStatusUpdate broadcasts polling status changes
func (m *manager) broadcastStatusUpdate(state *PollingState) {
	statusData := map[string]interface{}{
		"interval":     formatInterval(state.CurrentInterval),
		"success_rate": state.SuccessRate,
		"last_polled":  state.LastPolled,
	}

	m.sseMgr.Broadcast(sse.Message{
		Type:      "status",
		StationID: state.StationID,
		Data:      statusData,
		Timestamp: time.Now(),
	})
}

// Utility functions (same as from main.go)

func analyzeObservationIntervals(observations []FMIWindObservation) (time.Duration, bool) {
	if len(observations) < 2 {
		return IntervalFast, false
	}

	intervals := []time.Duration{}
	for i := 1; i < len(observations); i++ {
		interval := observations[i].Timestamp.Sub(observations[i-1].Timestamp)
		if interval > 30*time.Second && interval < 2*time.Hour {
			intervals = append(intervals, interval)
		}
	}

	if len(intervals) == 0 {
		return IntervalFast, false
	}

	minInterval := intervals[0]
	for _, interval := range intervals {
		if interval < minInterval {
			minInterval = interval
		}
	}

	return minInterval, true
}

func roundToStandardInterval(d time.Duration) time.Duration {
	if d <= 90*time.Second {
		return IntervalFast
	} else if d <= 12*time.Minute {
		return IntervalMedium
	} else if d <= 70*time.Minute {
		return IntervalSlow
	}
	return IntervalUltraSlow
}

func getNextSlowerInterval(current time.Duration) time.Duration {
	switch current {
	case IntervalFast:
		return IntervalMedium
	case IntervalMedium:
		return IntervalSlow
	case IntervalSlow:
		return IntervalUltraSlow
	default:
		return IntervalUltraSlow
	}
}

func getEffectivePollingInterval(baseInterval time.Duration, hasSSEClients bool) time.Duration {
	if !hasSSEClients {
		return IntervalSlow // Save resources when no clients
	}
	return baseInterval
}

func formatInterval(d time.Duration) string {
	switch d {
	case IntervalFast:
		return "1m"
	case IntervalMedium:
		return "10m"
	case IntervalSlow:
		return "60m"
	case IntervalUltraSlow:
		return "24h"
	default:
		return d.String()
	}
}

// State persistence methods

func (m *manager) loadPollingStates() {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading polling states: %v", err)
		}
		return
	}

	m.pollingStatesMutex.Lock()
	defer m.pollingStatesMutex.Unlock()

	if err := json.Unmarshal(data, &m.pollingStates); err != nil {
		log.Printf("Error parsing polling states: %v", err)
		return
	}

	log.Printf("Loaded polling states for %d stations", len(m.pollingStates))
}

func (m *manager) savePollingStates() {
	m.pollingStatesMutex.RLock()
	data, err := json.MarshalIndent(m.pollingStates, "", "  ")
	m.pollingStatesMutex.RUnlock()

	if err != nil {
		log.Printf("Error marshaling polling states: %v", err)
		return
	}

	if err := os.WriteFile(m.stateFile, data, 0644); err != nil {
		log.Printf("Error saving polling states: %v", err)
	}
}

func (m *manager) loadWindData() {
	data, err := os.ReadFile(m.windDataFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading wind data: %v", err)
		}
		return
	}

	m.windDataMutex.Lock()
	defer m.windDataMutex.Unlock()

	if err := json.Unmarshal(data, &m.windData); err != nil {
		log.Printf("Error parsing wind data: %v", err)
		return
	}

	log.Printf("Loaded wind data for %d stations", len(m.windData))
}

func (m *manager) saveWindData() {
	m.windDataMutex.RLock()
	data, err := json.MarshalIndent(m.windData, "", "  ")
	m.windDataMutex.RUnlock()

	if err != nil {
		log.Printf("Error marshaling wind data: %v", err)
		return
	}

	if err := os.WriteFile(m.windDataFile, data, 0644); err != nil {
		log.Printf("Error saving wind data: %v", err)
	}
}
