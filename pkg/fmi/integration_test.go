//go:build integration
// +build integration

package fmi

import (
	"fmt"
	"testing"
	"time"
)

// TestFetchStationsIntegration tests fetching station metadata from FMI API
func TestFetchStationsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	// Test connection first
	err := client.TestConnection()
	if err != nil {
		t.Fatalf("FMI API connection test failed: %v", err)
	}
	t.Log("FMI API connection test passed")

	// Test fetching stations within Southern Finland bounding box
	stations, err := client.FetchStations(&SouthernFinlandBBox)
	if err != nil {
		t.Fatalf("Failed to fetch stations: %v", err)
	}

	if len(stations) == 0 {
		t.Fatal("No stations returned from FMI API")
	}

	t.Logf("Successfully fetched %d stations from FMI API", len(stations))

	// Validate station data structure
	helsinkiFound := false
	for _, station := range stations {
		// Basic validation
		if station.ID == "" {
			t.Error("Station has empty ID")
			continue
		}
		if station.FMISID == "" {
			t.Error("Station has empty FMISID")
			continue
		}
		if station.Name == "" {
			t.Error("Station has empty Name")
			continue
		}
		if station.Location.Lat == 0 && station.Location.Lon == 0 {
			t.Errorf("Station %s has invalid coordinates: %+v", station.ID, station.Location)
			continue
		}

		// Look for Helsinki Kaisaniemi as a known reference station
		if station.FMISID == "100971" {
			helsinkiFound = true
			t.Logf("Found Helsinki Kaisaniemi station: ID=%s, Name=%s, Lat=%.5f, Lon=%.5f, Network=%s",
				station.FMISID, station.Name, station.Location.Lat, station.Location.Lon, station.Network)

			// Validate Helsinki Kaisaniemi specific data
			if station.Name != "Helsinki Kaisaniemi" {
				t.Errorf("Expected station name 'Helsinki Kaisaniemi', got '%s'", station.Name)
			}

			// Check approximate coordinates (allow small variance)
			expectedLat, expectedLon := 60.17523, 24.94459
			tolerance := 0.001
			if abs(station.Location.Lat-expectedLat) > tolerance {
				t.Errorf("Helsinki Kaisaniemi latitude %.5f differs from expected %.5f by more than %.3f",
					station.Location.Lat, expectedLat, tolerance)
			}
			if abs(station.Location.Lon-expectedLon) > tolerance {
				t.Errorf("Helsinki Kaisaniemi longitude %.5f differs from expected %.5f by more than %.3f",
					station.Location.Lon, expectedLon, tolerance)
			}
		}
	}

	if !helsinkiFound {
		t.Error("Helsinki Kaisaniemi station (FMISID 100971) not found in Southern Finland results")
	}
}

// TestWindDataFetchingIntegration tests fetching wind data from FMI API
func TestWindDataFetchingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	// Test connection first
	err := client.TestConnection()
	if err != nil {
		t.Fatalf("FMI API connection test failed: %v", err)
	}

	// Prepare test data collection
	var stationDataResults []StationWindData
	var processingStats ProcessingStats
	errorOccurred := false
	parsingCompleted := false

	callbacks := WindDataCallbacks{
		OnStationData: func(stationData StationWindData) error {
			stationDataResults = append(stationDataResults, stationData)
			t.Logf("Received data for station %s (%s): %d observations",
				stationData.StationID, stationData.StationName, len(stationData.Observations))
			return nil
		},

		OnError: func(err error) {
			t.Logf("Parser error: %v", err)
			errorOccurred = true
		},

		OnStart: func() {
			t.Log("Wind data streaming started")
		},

		OnComplete: func(stats ProcessingStats) {
			processingStats = stats
			parsingCompleted = true
			t.Logf("Streaming completed: %d stations, %d observations in %v",
				stats.StationCount, stats.ProcessedObservations, stats.Duration)
		},
	}

	// Test fetching recent wind data for Helsinki Kaisaniemi
	endTime := time.Now()
	startTime := endTime.Add(-3 * time.Hour) // Last 3 hours

	req := WindDataRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		StationIDs: []string{"100971"}, // Helsinki Kaisaniemi
	}

	err = client.StreamWindDataByStation(req, callbacks)
	if err != nil {
		t.Fatalf("Failed to stream wind data: %v", err)
	}

	// Validate results
	if !parsingCompleted {
		t.Error("OnComplete callback was not called")
	}

	if errorOccurred {
		t.Log("Parsing reported errors (may be normal for recent data)")
	}

	if len(stationDataResults) == 0 {
		t.Error("No station data received from API")
		return
	}

	// Validate station data
	helsinkiData := stationDataResults[0]
	if helsinkiData.StationID != "100971" {
		t.Errorf("Expected station ID '100971', got '%s'", helsinkiData.StationID)
	}

	if helsinkiData.StationName != "Helsinki Kaisaniemi" {
		t.Errorf("Expected station name 'Helsinki Kaisaniemi', got '%s'", helsinkiData.StationName)
	}

	// Check coordinates
	expectedLat, expectedLon := 60.17523, 24.94459
	tolerance := 0.001
	if abs(helsinkiData.Location.Lat-expectedLat) > tolerance {
		t.Errorf("Station latitude %.5f differs from expected %.5f",
			helsinkiData.Location.Lat, expectedLat)
	}
	if abs(helsinkiData.Location.Lon-expectedLon) > tolerance {
		t.Errorf("Station longitude %.5f differs from expected %.5f",
			helsinkiData.Location.Lon, expectedLon)
	}

	t.Logf("Station coordinates: Lat=%.5f, Lon=%.5f",
		helsinkiData.Location.Lat, helsinkiData.Location.Lon)

	// Analyze observations
	if len(helsinkiData.Observations) > 0 {
		t.Logf("Received %d observations", len(helsinkiData.Observations))

		// Check first few observations for data quality
		for i, obs := range helsinkiData.Observations {
			if i >= 3 { // Limit output for readability
				break
			}

			speedVal := "nil"
			gustVal := "nil"
			dirVal := "nil"

			if obs.WindSpeed != nil {
				speedVal = fmt.Sprintf("%.1f", *obs.WindSpeed)
			}
			if obs.WindGust != nil {
				gustVal = fmt.Sprintf("%.1f", *obs.WindGust)
			}
			if obs.WindDirection != nil {
				dirVal = fmt.Sprintf("%.1f", *obs.WindDirection)
			}

			t.Logf("Observation %d: Speed=%s m/s, Gust=%s m/s, Dir=%s°, Time=%v",
				i, speedVal, gustVal, dirVal, obs.Timestamp)

			// Basic data validation
			if obs.Timestamp.IsZero() {
				t.Errorf("Observation %d has zero timestamp", i)
			}

			if obs.Timestamp.After(endTime) {
				t.Errorf("Observation %d has future timestamp: %v", i, obs.Timestamp)
			}

			if obs.Timestamp.Before(startTime) {
				t.Errorf("Observation %d has timestamp before requested range: %v", i, obs.Timestamp)
			}
		}

		// Count how many observations have each type of data
		speedCount, gustCount, dirCount := 0, 0, 0
		for _, obs := range helsinkiData.Observations {
			if obs.WindSpeed != nil {
				speedCount++
			}
			if obs.WindGust != nil {
				gustCount++
			}
			if obs.WindDirection != nil {
				dirCount++
			}
		}

		t.Logf("Data completeness: %d speed, %d gust, %d direction out of %d observations",
			speedCount, gustCount, dirCount, len(helsinkiData.Observations))

		// We should have at least some wind speed data
		if speedCount == 0 {
			t.Error("No wind speed observations found in API response")
		}
	} else {
		t.Log("No observations returned (may be normal for very recent time range)")
	}

	// Validate processing stats
	if processingStats.StationCount == 0 {
		t.Error("Processing stats shows 0 stations processed")
	}

	t.Logf("Final processing stats: %+v", processingStats)
}

// TestMultiStationWindDataIntegration tests fetching wind data from multiple stations
func TestMultiStationWindDataIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := NewClient()

	var stationDataResults []StationWindData

	callbacks := WindDataCallbacks{
		OnStationData: func(stationData StationWindData) error {
			stationDataResults = append(stationDataResults, stationData)
			return nil
		},
		OnComplete: func(stats ProcessingStats) {
			t.Logf("Multi-station processing completed: %d stations, %d observations in %v",
				stats.StationCount, stats.ProcessedObservations, stats.Duration)
		},
	}

	// Test with multiple stations in Helsinki area
	endTime := time.Now()
	startTime := endTime.Add(-2 * time.Hour)

	stationIDs := []string{"100971", "101104"} // Helsinki Kaisaniemi, Espoo Tapiola

	err := client.StreamWindDataStations(stationIDs, startTime, endTime, callbacks)
	if err != nil {
		t.Fatalf("Failed to stream multi-station wind data: %v", err)
	}

	// Should receive data for multiple stations
	t.Logf("Received data for %d stations", len(stationDataResults))

	if len(stationDataResults) > 1 {
		t.Log("Multi-station query successful:")
		for _, stationData := range stationDataResults {
			t.Logf("  Station %s (%s): %d observations",
				stationData.StationID, stationData.StationName, len(stationData.Observations))
		}
	}
}
