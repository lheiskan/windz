package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"windz-monitor/pkg/fmi/observations"
)

// Station represents a weather station
type Station struct {
	ID     string
	Name   string
	Region string
}

// Wind reading data
type WindReading struct {
	StationID     string    `json:"station_id"`
	StationName   string    `json:"station_name"`
	Region        string    `json:"region"`
	Timestamp     time.Time `json:"timestamp"`
	WindSpeed     float64   `json:"wind_speed"`
	WindGust      float64   `json:"wind_gust"`
	WindDirection float64   `json:"wind_direction"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Polling state for adaptive algorithm
type StationPollingState struct {
	StationID         string        `json:"station_id"`
	CurrentInterval   time.Duration `json:"current_interval"`
	ConsecutiveMisses int           `json:"consecutive_misses"`
	LastPolled        time.Time     `json:"last_polled"`
	LastObservation   time.Time     `json:"last_observation"`
	SuccessRate       float64       `json:"success_rate"`
	TotalPolls        int           `json:"total_polls"`
	SuccessfulPolls   int           `json:"successful_polls"`
}

// Station status information for SSE updates
type StationStatus struct {
	Interval    string    `json:"interval"`
	SuccessRate float64   `json:"success_rate"`
	LastPolled  time.Time `json:"last_polled"`
}

// SSE message structure
type SSEMessage struct {
	Type          string         `json:"type"`
	StationID     string         `json:"station_id,omitempty"`
	Data          *WindReading   `json:"data,omitempty"`
	StationStatus *StationStatus `json:"station_status,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

// PerformanceMetrics tracks FMI API efficiency
type PerformanceMetrics struct {
	// Gzip compression metrics
	GzipRequests         int     `json:"gzip_requests"`
	GzipResponses        int     `json:"gzip_responses"`
	GzipSavedBytes       int64   `json:"gzip_saved_bytes"`
	GzipCompressionRatio float64 `json:"gzip_compression_ratio"`

	// Batching efficiency metrics
	TotalAPIRequests     int     `json:"total_api_requests"`
	TotalStationsQueried int     `json:"total_stations_queried"`
	BatchEfficiency      float64 `json:"batch_efficiency"` // stations per request
	LargestBatchSize     int     `json:"largest_batch_size"`
	TimeWindowGroups     int     `json:"time_window_groups"`

	// Response metrics
	TotalObservations   int           `json:"total_observations"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastUpdated         time.Time     `json:"last_updated"`
}

// Global state
var (
	stations = []Station{
		// Porkkala Area (KEY STATIONS)
		{ID: "101023", Name: "Emäsalo", Region: "Porvoo"},
		{ID: "101022", Name: "Kalbådagrund", Region: "Porkkala"},
		{ID: "105392", Name: "Itätoukki", Region: "Sipoo"},
		{ID: "151028", Name: "Vuosaari", Region: "Helsinki"},

		// Maritime & Coastal
		{ID: "100996", Name: "Harmaja", Region: "Helsinki Maritime"},
		{ID: "100969", Name: "Bågaskär", Region: "Inkoo Coastal"},
		{ID: "100965", Name: "Jussarö", Region: "Raasepori Maritime"},
		{ID: "100946", Name: "Tulliniemi", Region: "Hanko Coastal"},
		{ID: "100932", Name: "Russarö", Region: "Hanko Southern"},
		{ID: "100945", Name: "Vänö", Region: "Kemiönsaari"},
		{ID: "100908", Name: "Utö", Region: "Archipelago HELCOM"},

		// 	// Northern Coastal
		{ID: "101267", Name: "Tahkoluoto", Region: "Pori"},
		{ID: "101661", Name: "Tankar", Region: "Kokkola"},
		{ID: "101673", Name: "Ulkokalla", Region: "Kalajoki"},
		{ID: "101784", Name: "Marjaniemi", Region: "Hailuoto"},
		{ID: "101794", Name: "Vihreäsaari", Region: "Oulu"},
	}

	// Global state management
	windData      = make(map[string]*WindReading)
	windDataMutex sync.RWMutex

	pollingStates      = make(map[string]*StationPollingState)
	pollingStatesMutex sync.RWMutex

	sseClients      = make(map[chan SSEMessage]bool)
	sseClientsMutex sync.RWMutex

	// Performance metrics
	perfMetrics      = &PerformanceMetrics{}
	perfMetricsMutex sync.RWMutex

	// Configuration
	port      = flag.Int("port", 8080, "HTTP server port")
	stateFile = flag.String("state-file", "polling_state.json", "Polling state persistence file")
	debug     = flag.Bool("debug", false, "Enable debug logging")
)

// FMI XML structures for parsing
type WFSFeatureCollection struct {
	XMLName xml.Name    `xml:"FeatureCollection"`
	Members []WFSMember `xml:"member"`
}

type WFSMember struct {
	GridSeriesObservation GridSeriesObservation `xml:"GridSeriesObservation"`
}

type GridSeriesObservation struct {
	FeatureOfInterest FeatureOfInterest `xml:"featureOfInterest"`
	Result            ObservationResult `xml:"result"`
}

type FeatureOfInterest struct {
	SamplingFeature SamplingFeature `xml:"SF_SpatialSamplingFeature"`
}

type SamplingFeature struct {
	SampledFeature SampledFeature `xml:"sampledFeature"`
}

type SampledFeature struct {
	LocationCollection LocationCollection `xml:"LocationCollection"`
}

type LocationCollection struct {
	Members []LocationMember `xml:"member"`
}

type LocationMember struct {
	Location FMILocation `xml:"Location"`
}

type FMILocation struct {
	Identifier GMLIdentifier `xml:"identifier"`
	Name       []GMLName     `xml:"name"`
}

type GMLIdentifier struct {
	Value     string `xml:",chardata"`
	CodeSpace string `xml:"codeSpace,attr"`
}

type GMLName struct {
	Value     string `xml:",chardata"`
	CodeSpace string `xml:"codeSpace,attr"`
}

type ObservationResult struct {
	MultiPointCoverage MultiPointCoverage `xml:"MultiPointCoverage"`
}

type MultiPointCoverage struct {
	DomainSet DomainSet `xml:"domainSet"`
	RangeSet  RangeSet  `xml:"rangeSet"`
}

type DomainSet struct {
	SimpleMultiPoint SimpleMultiPoint `xml:"SimpleMultiPoint"`
}

type SimpleMultiPoint struct {
	Positions string `xml:"positions"`
}

type RangeSet struct {
	DataBlock DataBlock `xml:"DataBlock"`
}

type DataBlock struct {
	DoubleOrNilReasonTupleList string `xml:"doubleOrNilReasonTupleList"`
}

// Polling intervals
const (
	IntervalFast      = 1 * time.Minute
	IntervalMedium    = 10 * time.Minute
	IntervalSlow      = 60 * time.Minute
	IntervalUltraSlow = 24 * time.Hour
)

func main() {
	flag.Parse()

	log.Printf("WindZ Monitor starting on port %d", *port)
	log.Printf("Monitoring %d Finnish weather stations", len(stations))

	// Initialize polling states
	loadPollingStates()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start polling scheduler
	go runPollingScheduler(ctx)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/events", handleSSE)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/api/stations", handleAPIStations)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		cancel()
		savePollingStates()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Server starting at http://localhost:%d", *port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// Load polling states from file
func loadPollingStates() {
	data, err := os.ReadFile(*stateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading polling states: %v", err)
		}
		return
	}

	pollingStatesMutex.Lock()
	defer pollingStatesMutex.Unlock()
	if err := json.Unmarshal(data, &pollingStates); err != nil {
		log.Printf("Error parsing polling states: %v", err)
		return
	}

	log.Printf("Loaded polling states for %d stations", len(pollingStates))

}

// Save polling states to file
func savePollingStates() {
	pollingStatesMutex.RLock()
	data, err := json.MarshalIndent(pollingStates, "", "  ")
	pollingStatesMutex.RUnlock()

	if err != nil {
		log.Printf("Error marshaling polling states: %v", err)
		return
	}

	if err := os.WriteFile(*stateFile, data, 0644); err != nil {
		log.Printf("Error saving polling states: %v", err)
	}
}

// Polling scheduler
func runPollingScheduler(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Initial poll
	pollDueStations()

	for {
		select {
		case <-ticker.C:
			pollDueStations()
		case <-ctx.Done():
			return
		}
	}
}

// Poll stations that are due
func pollDueStations() {

	// Polling state is updated during this method
	// TODO: refactor?
	pollingStatesMutex.Lock()
	defer pollingStatesMutex.Unlock()

	now := time.Now()
	toPoll := []*StationPollingState{}

	// copy polling states for the stations we need to poll
	for _, station := range stations {
		state, exists := pollingStates[station.ID]

		// create polling state if it does not yet exist for the station
		if !exists {
			state = &StationPollingState{
				StationID:       station.ID,
				CurrentInterval: IntervalFast,
			}
			pollingStates[station.ID] = state
		}

		// schedule poll if needed
		if now.Sub(state.LastPolled) >= state.CurrentInterval {
			toPoll = append(toPoll, state)
		}
	}

	if len(toPoll) == 0 {
		return
	}

	if *debug {
		log.Printf("Polling %d stations: %v", len(toPoll), toPoll)
	}

	// Group stations by their effective time windows for batching
	const maxBatchSize = 20
	endTime := time.Now()
	defaultStartTime := endTime.Add(-2 * time.Hour)

	// Group stations by their effective start time
	timeWindowGroups := make(map[time.Time][]*StationPollingState)

	for _, state := range toPoll {
		effectiveStartTime := defaultStartTime
		if state.LastObservation.After(defaultStartTime) {
			// Round to second for grouping (stations within same second can be batched)
			effectiveStartTime = state.LastObservation.Add(time.Second).Truncate(time.Second)
		}

		timeWindowGroups[effectiveStartTime] = append(timeWindowGroups[effectiveStartTime], state)
	}

	// Update time window groups metric
	perfMetricsMutex.Lock()
	perfMetrics.TimeWindowGroups = len(timeWindowGroups)
	perfMetricsMutex.Unlock()

	// Process each time window group in batches of up to 20 stations
	for groupStartTime, groupStations := range timeWindowGroups {
		if *debug && len(groupStations) > 1 {
			log.Printf("Time window group (start: %v) has %d stations - batching opportunity",
				groupStartTime.Format("15:04:05"), len(groupStations))
		}

		// Process this time window group in batches
		for i := 0; i < len(groupStations); i += maxBatchSize {
			// Create batch of up to 20 stations with same time window
			end := i + maxBatchSize
			if end > len(groupStations) {
				end = len(groupStations)
			}
			batch := groupStations[i:end]

			// Collect station IDs for batch request
			stationIDs := make([]string, len(batch))
			for j, state := range batch {
				stationIDs[j] = state.StationID
			}

			// Execute batch request with the group's time window
			if *debug {
				log.Printf("Fetching batch of %d stations (time window: %v-%v): %v",
					len(stationIDs), groupStartTime.Format("15:04:05"), endTime.Format("15:04:05"), stationIDs)
			}

			batchResults, err := fetchWindDataBatch(stationIDs, groupStartTime, endTime)
			if err != nil {
				log.Printf("Error fetching wind data for batch of %d stations: %v", len(stationIDs), err)
				// Mark all stations in this batch as failed
				for _, state := range batch {
					state.TotalPolls++
					state.ConsecutiveMisses++
					state.LastPolled = time.Now()
					// Slow down polling on consecutive failures
					if state.ConsecutiveMisses >= 3 {
						state.CurrentInterval = getNextSlowerInterval(state.CurrentInterval)
					}
				}
				continue
			}

			// Process results for each station in the batch
			for _, state := range batch {
				stationID := state.StationID
				observations, hasObservations := batchResults[stationID]

				// If station not in results, treat as empty response (not error)
				if !hasObservations {
					observations = []WindObservation{}
				}

				oldInterval := state.CurrentInterval
				latestObservation, hasData := updatePollingState(state, observations)

				if oldInterval != state.CurrentInterval {
					broadcastSSE(SSEMessage{
						Type:      "status",
						StationID: stationID,
						StationStatus: &StationStatus{
							Interval:    formatInterval(state.CurrentInterval),
							SuccessRate: state.SuccessRate,
							LastPolled:  state.LastPolled,
						},
						Timestamp: time.Now(),
					})
				}

				if hasData {
					updateWindData(stationID, latestObservation)
				}
			}
		}
	}

	return
}

// Fetch wind data from FMI API
// fetchWindData fetches wind data using the new observations package (legacy single-station function)
func fetchWindData(stationID string, startTime, endTime time.Time) (result []WindObservation, err error) {
	// Use the new batched function for single station
	stationResults, err := fetchWindDataBatch([]string{stationID}, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Return results for the requested station
	if results, found := stationResults[stationID]; found {
		return results, nil
	}

	return []WindObservation{}, nil
}

// fetchWindDataBatch fetches wind data for multiple stations in a single API call
func fetchWindDataBatch(stationIDs []string, startTime, endTime time.Time) (map[string][]WindObservation, error) {
	if len(stationIDs) == 0 {
		return make(map[string][]WindObservation), nil
	}

	// Create observations query with default HTTP client
	query := observations.NewQuery("https://opendata.fmi.fi/wfs", &http.Client{Timeout: 60 * time.Second})

	// Create request for multiple stations
	req := observations.Request{
		StartTime:  startTime,
		EndTime:    endTime,
		StationIDs: stationIDs,
		UseGzip:    true,
	}

	// Execute query and track performance metrics
	requestStart := time.Now()
	response, err := query.Execute(req)
	requestDuration := time.Since(requestStart)

	if err != nil {
		// Update failed request metrics
		updatePerformanceMetrics(len(stationIDs), 0, false, false, requestDuration, 0, 0)
		return nil, fmt.Errorf("failed to fetch wind data for %d stations: %w", len(stationIDs), err)
	}

	// Update successful request metrics (we'll improve gzip detection later)
	updatePerformanceMetrics(len(stationIDs), response.Stats.ProcessedObservations,
		true, true, requestDuration, 0, 0) // Assume gzip worked for now

	if *debug {
		log.Printf("FMI batch processing completed: %d stations requested, %d stations returned, %d total observations (duration: %v)",
			len(stationIDs), len(response.Stations), response.Stats.ProcessedObservations, requestDuration)
	}

	// Create result map indexed by station ID
	results := make(map[string][]WindObservation)

	// Convert observations data to our WindObservation format
	for _, station := range response.Stations {
		stationResults := make([]WindObservation, 0, len(station.Observations))

		for _, obs := range station.Observations {
			windObs := WindObservation{
				Timestamp: obs.Timestamp,
			}

			// Convert FMI float pointers to our float64 values
			if obs.WindSpeed != nil {
				windObs.WindSpeed = *obs.WindSpeed
			}
			if obs.WindGust != nil {
				windObs.WindGust = *obs.WindGust
			}
			if obs.WindDirection != nil {
				windObs.WindDirection = *obs.WindDirection
			}

			// Only include observations with valid wind speed
			if windObs.WindSpeed >= 0 && windObs.WindSpeed < 100 {
				stationResults = append(stationResults, windObs)
			}
		}

		results[station.StationID] = stationResults
	}

	return results, nil
}

// updatePerformanceMetrics updates the global performance metrics
func updatePerformanceMetrics(stationsQueried, observations int, success, gzipUsed bool,
	duration time.Duration, uncompressedBytes, compressedBytes int64) {

	perfMetricsMutex.Lock()
	defer perfMetricsMutex.Unlock()

	// Update API request metrics
	perfMetrics.TotalAPIRequests++
	perfMetrics.TotalStationsQueried += stationsQueried
	perfMetrics.TotalObservations += observations

	// Update batch efficiency
	perfMetrics.BatchEfficiency = float64(perfMetrics.TotalStationsQueried) / float64(perfMetrics.TotalAPIRequests)
	if stationsQueried > perfMetrics.LargestBatchSize {
		perfMetrics.LargestBatchSize = stationsQueried
	}

	// Update gzip metrics if enabled
	if success {
		perfMetrics.GzipRequests++
		if gzipUsed {
			perfMetrics.GzipResponses++
		}

		// Update compression ratio if we have bytes data
		if uncompressedBytes > 0 && compressedBytes > 0 {
			perfMetrics.GzipSavedBytes += (uncompressedBytes - compressedBytes)
			perfMetrics.GzipCompressionRatio = float64(compressedBytes) / float64(uncompressedBytes)
		}
	}

	// Update response time (simple moving average)
	if perfMetrics.AverageResponseTime == 0 {
		perfMetrics.AverageResponseTime = duration
	} else {
		// Simple exponential moving average (alpha = 0.1)
		perfMetrics.AverageResponseTime = time.Duration(
			0.9*float64(perfMetrics.AverageResponseTime) + 0.1*float64(duration))
	}

	perfMetrics.LastUpdated = time.Now()
}

// Wind observation from FMI
type WindObservation struct {
	Timestamp     time.Time
	WindSpeed     float64
	WindGust      float64
	WindDirection float64
}

// Update polling state based on results
func updatePollingState(state *StationPollingState, observations []WindObservation) (
	WindObservation, bool) {

	state.LastPolled = time.Now()
	state.TotalPolls++

	var lastObservation WindObservation
	hadData := false

	if len(observations) > 0 {

		hadData = true
		lastObservation = observations[len(observations)-1]

		// Success - we got data
		state.SuccessfulPolls++
		state.ConsecutiveMisses = 0
		state.LastObservation = lastObservation.Timestamp
		state.SuccessRate = float64(state.SuccessfulPolls) / float64(state.TotalPolls)

		// Analyze observation intervals and adapt
		if len(observations) >= 2 {
			minInterval, ok := analyzeObservationIntervals(observations)
			if ok && minInterval < state.CurrentInterval {
				// Speed up polling to match data frequency
				state.CurrentInterval = roundToStandardInterval(minInterval)
			}
		}

	} else {
		// No data - increment misses
		state.ConsecutiveMisses++
		state.SuccessRate = float64(state.SuccessfulPolls) / float64(state.TotalPolls)

		// Back off after 2 consecutive misses
		if state.ConsecutiveMisses >= 2 {
			state.CurrentInterval = getNextSlowerInterval(state.CurrentInterval)
			state.ConsecutiveMisses = 0
		}
	}
	return lastObservation, hadData
}

// Analyze observation intervals to detect publishing frequency
func analyzeObservationIntervals(observations []WindObservation) (time.Duration, bool) {
	if len(observations) < 2 {
		return IntervalFast, false
	}

	intervals := []time.Duration{}
	for i := 1; i < len(observations); i++ {
		interval := observations[i].Timestamp.Sub(observations[i-1].Timestamp)
		// Filter out anomalies
		if interval > 30*time.Second && interval < 2*time.Hour {
			intervals = append(intervals, interval)
		}
	}

	if len(intervals) == 0 {
		return IntervalFast, false
	}

	// Find minimum interval
	minInterval := intervals[0]
	for _, interval := range intervals {
		if interval < minInterval {
			minInterval = interval
		}
	}

	return minInterval, true
}

// Round to standard interval
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

// Get next slower interval for backoff
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

// Update wind data and broadcast to SSE clients
func updateWindData(stationID string, obs WindObservation) {
	station := getStation(stationID)
	if station == nil {
		return
	}

	reading := &WindReading{
		StationID:     stationID,
		StationName:   station.Name,
		Region:        station.Region,
		Timestamp:     obs.Timestamp,
		WindSpeed:     obs.WindSpeed,
		WindGust:      obs.WindGust,
		WindDirection: obs.WindDirection,
		UpdatedAt:     time.Now(),
	}

	windDataMutex.Lock()
	windData[stationID] = reading
	windDataMutex.Unlock()

	// Broadcast to SSE clients
	broadcastSSE(SSEMessage{
		Type:      "data",
		StationID: stationID,
		Data:      reading,
		Timestamp: time.Now(),
	})
}

// Broadcast message to all SSE clients
func broadcastSSE(msg SSEMessage) {
	sseClientsMutex.RLock()
	defer sseClientsMutex.RUnlock()

	for client := range sseClients {
		select {
		case client <- msg:
		default:
			// Client buffer full, skip
		}
	}
}

// Get station by ID
func getStation(id string) *Station {
	for i := range stations {
		if stations[i].ID == id {
			return &stations[i]
		}
	}
	return nil
}

// Parse unix timestamp from FMI format
func parseUnixTime(s string) time.Time {
	var timestamp int64
	if n, err := fmt.Sscanf(s, "%d", &timestamp); err == nil && n == 1 {
		return time.Unix(timestamp, 0)
	}
	return time.Time{}
}

// Parse float from string
func parseFloat(s string) float64 {
	if s == "NaN" || s == "" {
		return -1
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// HTTP Handlers

// Handle main page
func handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlContent)
}

// Handle SSE connections
func handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client channel
	clientChan := make(chan SSEMessage, 100)

	// Register client
	sseClientsMutex.Lock()
	sseClients[clientChan] = true
	sseClientsMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		sseClientsMutex.Lock()
		delete(sseClients, clientChan)
		sseClientsMutex.Unlock()
		close(clientChan)
	}()

	// Send initial data and status
	windDataMutex.RLock()
	for _, reading := range windData {
		clientChan <- SSEMessage{
			Type:      "data",
			StationID: reading.StationID,
			Data:      reading,
			Timestamp: time.Now(),
		}
	}
	windDataMutex.RUnlock()

	// Send initial status for all stations
	pollingStatesMutex.RLock()
	for stationID, state := range pollingStates {
		clientChan <- SSEMessage{
			Type:      "status",
			StationID: stationID,
			StationStatus: &StationStatus{
				Interval:    formatInterval(state.CurrentInterval),
				SuccessRate: state.SuccessRate,
				LastPolled:  state.LastPolled,
			},
			Timestamp: time.Now(),
		}
	}
	pollingStatesMutex.RUnlock()

	// Get flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send events
	for {
		select {
		case msg := <-clientChan:
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "event: %s\n", msg.Type)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

		case <-r.Context().Done():
			return

		case <-time.After(30 * time.Second):
			// Send heartbeat
			fmt.Fprint(w, ":heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// Handle health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"stations": len(stations),
		"clients":  len(sseClients),
		"uptime":   time.Since(startTime).String(),
	})
}

// Handle metrics endpoint
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	pollingStatesMutex.RLock()
	defer pollingStatesMutex.RUnlock()

	active, slow, offline := 0, 0, 0
	totalPolls, successfulPolls := 0, 0

	for _, state := range pollingStates {
		totalPolls += state.TotalPolls
		successfulPolls += state.SuccessfulPolls

		switch state.CurrentInterval {
		case IntervalFast:
			active++
		case IntervalMedium:
			active++
		case IntervalSlow:
			slow++
		case IntervalUltraSlow:
			offline++
		}
	}

	// Get performance metrics
	perfMetricsMutex.RLock()
	perfMetricsCopy := *perfMetrics
	perfMetricsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		// Existing polling metrics
		"stations_active":  active,
		"stations_slow":    slow,
		"stations_offline": offline,
		"total_polls":      totalPolls,
		"successful_polls": successfulPolls,
		"success_rate":     float64(successfulPolls) / math.Max(float64(totalPolls), 1),
		"sse_connections":  len(sseClients),
		"uptime_hours":     time.Since(startTime).Hours(),

		// New FMI API performance metrics
		"fmi_api": map[string]interface{}{
			// Batching efficiency
			"total_requests":       perfMetricsCopy.TotalAPIRequests,
			"total_stations":       perfMetricsCopy.TotalStationsQueried,
			"batch_efficiency":     perfMetricsCopy.BatchEfficiency,
			"largest_batch_size":   perfMetricsCopy.LargestBatchSize,
			"time_window_groups":   perfMetricsCopy.TimeWindowGroups,
			"total_observations":   perfMetricsCopy.TotalObservations,
			"avg_response_time_ms": float64(perfMetricsCopy.AverageResponseTime.Nanoseconds()) / 1e6,

			// Gzip compression
			"gzip_requests":          perfMetricsCopy.GzipRequests,
			"gzip_responses":         perfMetricsCopy.GzipResponses,
			"gzip_usage_rate":        float64(perfMetricsCopy.GzipResponses) / math.Max(float64(perfMetricsCopy.GzipRequests), 1),
			"gzip_saved_bytes":       perfMetricsCopy.GzipSavedBytes,
			"gzip_compression_ratio": perfMetricsCopy.GzipCompressionRatio,

			"last_updated": perfMetricsCopy.LastUpdated,
		},
	})
}

// Handle API stations endpoint
func handleAPIStations(w http.ResponseWriter, r *http.Request) {
	pollingStatesMutex.RLock()
	windDataMutex.RLock()
	defer pollingStatesMutex.RUnlock()
	defer windDataMutex.RUnlock()

	type StationStatus struct {
		Station
		PollingInterval string       `json:"polling_interval"`
		LastPolled      time.Time    `json:"last_polled"`
		LastObservation time.Time    `json:"last_observation"`
		SuccessRate     float64      `json:"success_rate"`
		LatestData      *WindReading `json:"latest_data,omitempty"`
	}

	statuses := []StationStatus{}
	for _, station := range stations {
		state := pollingStates[station.ID]
		status := StationStatus{
			Station:         station,
			PollingInterval: formatInterval(state.CurrentInterval),
			LastPolled:      state.LastPolled,
			LastObservation: state.LastObservation,
			SuccessRate:     state.SuccessRate,
			LatestData:      windData[station.ID],
		}
		statuses = append(statuses, status)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}

// Format interval for display
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

var startTime = time.Now()

// Embedded HTML content
const htmlContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Finnish Wind Monitor</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
        }

        header {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 15px;
            padding: 20px 30px;
            margin-bottom: 20px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
        }

        h1 {
            color: #2d3748;
            font-size: 28px;
            margin-bottom: 10px;
        }

        .subtitle {
            color: #718096;
            font-size: 14px;
        }

        .status-bar {
            background: rgba(255, 255, 255, 0.95);
            border-radius: 10px;
            padding: 15px 20px;
            margin-bottom: 20px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            box-shadow: 0 5px 20px rgba(0, 0, 0, 0.1);
        }

        .status-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .status-indicator {
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: #48bb78;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%, 100% { opacity: 1; }
            50% { opacity: 0.5; }
        }

        .status-indicator.disconnected {
            background: #f56565;
            animation: none;
        }

        .wind-table {
            background: rgba(255, 255, 255, 0.98);
            border-radius: 15px;
            overflow: hidden;
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
        }

        table {
            width: 100%;
            border-collapse: collapse;
        }

        thead {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }

        th {
            padding: 15px;
            text-align: left;
            font-weight: 600;
            font-size: 13px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        tbody tr {
            border-bottom: 1px solid #e2e8f0;
            transition: all 0.3s ease;
        }

        tbody tr:hover {
            background: #f7fafc;
        }

        td {
            padding: 12px 15px;
            font-size: 14px;
            color: #2d3748;
        }

        .station-name {
            font-weight: 600;
            color: #2d3748;
        }

        .region {
            color: #718096;
            font-size: 13px;
        }

        .wind-speed {
            font-weight: 600;
            font-size: 16px;
        }

        .wind-gust {
            color: #e53e3e;
            font-weight: 500;
        }

        .wind-direction {
            display: flex;
            align-items: center;
            gap: 5px;
        }

        .compass {
            font-weight: 600;
            color: #4a5568;
        }

        .updated {
            color: #718096;
            font-size: 12px;
        }

        .status-badge {
            display: inline-block;
            padding: 3px 8px;
            border-radius: 12px;
            font-size: 11px;
            font-weight: 600;
        }

        .status-badge.fast {
            background: #c6f6d5;
            color: #22543d;
        }

        .status-badge.medium {
            background: #fef5e7;
            color: #744210;
        }

        .status-badge.slow {
            background: #fed7d7;
            color: #742a2a;
        }

        .status-badge.offline {
            background: #e2e8f0;
            color: #4a5568;
        }

        @keyframes rowUpdate {
            0% {
                background-color: #fef5e7;
                transform: scale(1.01);
            }
            100% {
                background-color: transparent;
                transform: scale(1);
            }
        }

        .row-updated {
            animation: rowUpdate 1.5s ease-out;
        }

        .no-data {
            color: #cbd5e0;
            font-style: italic;
        }

        .wind-high {
            color: #e53e3e;
        }

        .wind-medium {
            color: #ed8936;
        }

        .wind-low {
            color: #48bb78;
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #718096;
        }

        @media (max-width: 768px) {
            .status-bar {
                flex-direction: column;
                gap: 10px;
            }

            th, td {
                padding: 8px 10px;
                font-size: 12px;
            }

            .wind-speed {
                font-size: 14px;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🌬️ Finnish Wind Monitor</h1>
            <div class="subtitle">Real-time wind data from 16 coastal and maritime weather stations</div>
        </header>

        <div class="status-bar">
            <div class="status-item">
                <div class="status-indicator" id="connection-indicator"></div>
                <span id="connection-status">Connecting...</span>
            </div>
            <div class="status-item">
                <span>Stations: <strong id="station-count">0</strong> / 16</span>
            </div>
            <div class="status-item">
                <span>Last Update: <strong id="last-update">-</strong></span>
            </div>
        </div>

        <div class="wind-table">
            <table>
                <thead>
                    <tr>
                        <th>Station</th>
                        <th>Wind Speed</th>
                        <th>Gust</th>
                        <th>Direction</th>
                        <th>Updated</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody id="wind-data">
                    <tr>
                        <td colspan="6" class="loading">Loading wind data...</td>
                    </tr>
                </tbody>
            </table>
        </div>
    </div>

    <script>
        const stations = [
            {id: "101023", name: "Emäsalo", region: "Porvoo"},
            {id: "101022", name: "Kalbådagrund", region: "Porkkala"},
            {id: "105392", name: "Itätoukki", region: "Sipoo"},
            {id: "151028", name: "Vuosaari", region: "Helsinki"},
            {id: "100996", name: "Harmaja", region: "Helsinki Maritime"},
            {id: "100969", name: "Bågaskär", region: "Inkoo Coastal"},
            {id: "100965", name: "Jussarö", region: "Raasepori Maritime"},
            {id: "100946", name: "Tulliniemi", region: "Hanko Coastal"},
            {id: "100932", name: "Russarö", region: "Hanko Southern"},
            {id: "100945", name: "Vänö", region: "Kemiönsaari"},
            {id: "100908", name: "Utö", region: "Archipelago HELCOM"},
            {id: "101267", name: "Tahkoluoto", region: "Pori"},
            {id: "101661", name: "Tankar", region: "Kokkola"},
            {id: "101673", name: "Ulkokalla", region: "Kalajoki"},
            {id: "101784", name: "Marjaniemi", region: "Hailuoto"},
            {id: "101794", name: "Vihreäsaari", region: "Oulu"}
        ];

        const stationData = new Map();
        const stationRows = new Map();
        let eventSource = null;
        let isConnected = false;

        function initializeTable() {
            const tbody = document.getElementById('wind-data');
            tbody.innerHTML = '';

            stations.forEach(station => {
                const row = document.createElement('tr');
                row.id = 'station-' + station.id;
                row.innerHTML = ` + "`" + `
                    <td>
                        <div class="station-name">${station.name}</div>
                        <div class="region">${station.region}</div>
                    </td>
                    <td class="wind-speed no-data">-</td>
                    <td class="wind-gust no-data">-</td>
                    <td class="wind-direction no-data">-</td>
                    <td class="updated no-data">-</td>
                    <td class="status">
                        <span class="status-badge offline">offline</span>
                    </td>
                ` + "`" + `;
                tbody.appendChild(row);
                stationRows.set(station.id, row);
            });
        }

        function formatWindSpeed(speed) {
            if (speed < 0) return '-';
            return speed.toFixed(1) + ' m/s';
        }

        function formatDirection(degrees) {
            if (degrees < 0) return '-';
            const directions = ['N', 'NNE', 'NE', 'ENE', 'E', 'ESE', 'SE', 'SSE',
                               'S', 'SSW', 'SW', 'WSW', 'W', 'WNW', 'NW', 'NNW'];
            const index = Math.round(degrees / 22.5) % 16;
            return ` + "`" + `${Math.round(degrees)}° <span class="compass">${directions[index]}</span>` + "`" + `;
        }

        function getWindSpeedClass(speed) {
            if (speed >= 15) return 'wind-high';
            if (speed >= 8) return 'wind-medium';
            return 'wind-low';
        }

        function formatTimeAgo(timestamp) {
            const now = new Date();
            const time = new Date(timestamp);
            const diff = Math.floor((now - time) / 1000);

            if (diff < 60) return 'just now';
            if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
            if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
            return Math.floor(diff / 86400) + 'd ago';
        }

        function getPollingStatus(interval) {
            switch(interval) {
                case '1m': return {class: 'fast', text: '1m'};
                case '10m': return {class: 'medium', text: '10m'};
                case '60m': return {class: 'slow', text: '60m'};
                case '24h': return {class: 'offline', text: '24h'};
                default: return {class: 'offline', text: 'unknown'};
            }
        }

        function updateStationRow(data) {
            const row = stationRows.get(data.station_id);
            if (!row) return;

            const windSpeedCell = row.cells[1];
            const windGustCell = row.cells[2];
            const windDirectionCell = row.cells[3];
            const updatedCell = row.cells[4];

            windSpeedCell.innerHTML = formatWindSpeed(data.wind_speed);
            windSpeedCell.className = 'wind-speed ' + getWindSpeedClass(data.wind_speed);

            windGustCell.innerHTML = formatWindSpeed(data.wind_gust);
            windGustCell.className = 'wind-gust';

            windDirectionCell.innerHTML = formatDirection(data.wind_direction);
            windDirectionCell.className = 'wind-direction';

            updatedCell.innerHTML = formatTimeAgo(data.timestamp);
            updatedCell.className = 'updated';

            // Add update animation
            row.classList.add('row-updated');
            setTimeout(() => row.classList.remove('row-updated'), 1500);

            stationData.set(data.station_id, data);
            updateStats();
        }

        function updateStationStatus(stationId, status) {
            const row = stationRows.get(stationId);
            if (!row || !row.cells[5]) return;

            const statusInfo = getPollingStatus(status.interval);
            row.cells[5].innerHTML = ` + "`" + `<span class="status-badge ${statusInfo.class}">${statusInfo.text}</span>` + "`" + `;
        }

        function updateStats() {
            document.getElementById('station-count').textContent = stationData.size;
            document.getElementById('last-update').textContent = new Date().toLocaleTimeString();
        }

        function updateConnectionStatus(connected) {
            isConnected = connected;
            const indicator = document.getElementById('connection-indicator');
            const status = document.getElementById('connection-status');

            if (connected) {
                indicator.classList.remove('disconnected');
                status.textContent = 'Connected';
            } else {
                indicator.classList.add('disconnected');
                status.textContent = 'Disconnected';
            }
        }

        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }

            eventSource = new EventSource('/events');

            eventSource.onopen = function() {
                updateConnectionStatus(true);
            };

            eventSource.addEventListener('data', function(event) {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.data) {
                        updateStationRow(msg.data);
                    }
                } catch (e) {
                    console.error('Error parsing SSE data:', e);
                }
            });

            eventSource.addEventListener('status', function(event) {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.station_status) {
                        updateStationStatus(msg.station_id, msg.station_status);
                    }
                } catch (e) {
                    console.error('Error parsing SSE status:', e);
                }
            });

            eventSource.onerror = function() {
                updateConnectionStatus(false);
                setTimeout(connectSSE, 5000);
            };
        }

        // Fetch initial status data
        async function fetchStationStatus() {
            try {
                const response = await fetch('/api/stations');
                const stations = await response.json();
                
                stations.forEach(station => {
                    const row = stationRows.get(station.ID);
                    if (row && row.cells[5]) {
                        const status = getPollingStatus(station.polling_interval);
                        row.cells[5].innerHTML = ` + "`" + `<span class="status-badge ${status.class}">${status.text}</span>` + "`" + `;
                    }
                    
                    if (station.latest_data) {
                        updateStationRow(station.latest_data);
                    }
                });
            } catch (e) {
                console.error('Error fetching station status:', e);
            }
        }

        // Update time ago every 30 seconds
        setInterval(() => {
            stationData.forEach((data, stationId) => {
                const row = stationRows.get(stationId);
                if (row && row.cells[4]) {
                    row.cells[4].innerHTML = formatTimeAgo(data.timestamp);
                }
            });
        }, 30000);

        // Periodically fetch status updates
        setInterval(fetchStationStatus, 60000);

        // Initialize
        initializeTable();
        connectSSE();
        fetchStationStatus();
    </script>
</body>
</html>
`
