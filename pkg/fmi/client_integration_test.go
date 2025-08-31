package fmi

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// Integration test - skipped by default, run with: go test -tags=integration
func TestClientFetchMultiStationData(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	client := NewClient()

	// Test with 3 stations for last hour
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	req := WindDataRequest{
		StartTime: startTime,
		EndTime:   endTime,
		StationIDs: []string{
			"100996", // Harmaja
			"101023", // Emäsalo
			"151028", // Vuosaari
		},
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
		UseGzip:    true, // Test gzip support
	}

	stations, err := client.FetchMultiStationData(req)
	if err != nil {
		t.Fatalf("Failed to fetch multi-station data: %v", err)
	}

	if len(stations) == 0 {
		t.Error("No stations returned")
	}

	// Print summary for manual verification
	for _, station := range stations {
		t.Logf("Station %s (%s):", station.StationID, station.StationName)
		t.Logf("  Location: %.5f, %.5f", station.Location.Lat, station.Location.Lon)
		t.Logf("  Observations: %d", len(station.Observations))

		if len(station.Observations) > 0 {
			latest := station.Observations[len(station.Observations)-1]
			t.Logf("  Latest observation: %s", latest.Timestamp.Format("15:04:05"))
			if latest.WindSpeed != nil {
				t.Logf("    Wind speed: %.1f m/s", *latest.WindSpeed)
			}
			if latest.WindGust != nil {
				t.Logf("    Wind gust: %.1f m/s", *latest.WindGust)
			}
			if latest.WindDirection != nil {
				t.Logf("    Wind direction: %.0f°", *latest.WindDirection)
			}
		}
	}
}

// Test error handling
func TestClientErrorHandling(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	client := NewClient()

	// Test with invalid station ID
	req := WindDataRequest{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"999999"}, // Invalid station ID
	}

	stations, err := client.FetchMultiStationData(req)
	if err == nil {
		t.Error("Expected error for invalid station ID")
	}

	if len(stations) > 0 {
		t.Error("Should not return stations for invalid ID")
	}
}

// Example usage function for documentation
func ExampleClient_FetchMultiStationData() {
	client := NewClient()

	// Fetch wind data for multiple stations
	req := WindDataRequest{
		StartTime: time.Now().Add(-2 * time.Hour),
		EndTime:   time.Now(),
		StationIDs: []string{
			"100996", // Helsinki Harmaja
			"101023", // Porvoo Emäsalo
			"151028", // Helsinki Vuosaari
		},
		Parameters: []WindParameter{
			WindSpeedMS,
			WindGustMS,
			WindDirection,
		},
		UseGzip: true, // Enable compression
	}

	stations, err := client.FetchMultiStationData(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Process the results
	for _, station := range stations {
		fmt.Printf("Station: %s (%s)\n", station.StationName, station.StationID)
		fmt.Printf("  Observations: %d\n", len(station.Observations))

		// Get latest observation
		if len(station.Observations) > 0 {
			latest := station.Observations[len(station.Observations)-1]
			if latest.WindSpeed != nil {
				fmt.Printf("  Latest wind speed: %.1f m/s\n", *latest.WindSpeed)
			}
		}
	}
}
