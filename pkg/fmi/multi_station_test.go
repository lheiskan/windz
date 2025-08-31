package fmi

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestParseMultiStationResponse(t *testing.T) {
	// Load the test XML file with 3 stations
	xmlData, err := os.ReadFile("../../test_three_station_response.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Parse the XML response
	stationData := make(map[string]*StationWindData)

	callbacks := WindDataCallbacks{
		OnStationData: func(data StationWindData) error {
			stationData[data.StationID] = &data
			t.Logf("Parsed station %s (%s): %d observations",
				data.StationID, data.StationName, len(data.Observations))
			return nil
		},
		OnError: func(err error) {
			t.Errorf("Parsing error: %v", err)
		},
		OnComplete: func(stats ProcessingStats) {
			t.Logf("Processing complete: %d stations, %d observations",
				stats.StationCount, stats.ProcessedObservations)
		},
	}

	// Create parser and parse the data
	parser := NewStationGroupingParser(bytes.NewReader(xmlData), callbacks)
	err = parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Verify we got data for all 3 stations
	expectedStations := map[string]struct {
		name string
		lat  float64
		lon  float64
	}{
		"100996": {"Helsinki Harmaja", 60.10512, 24.97539},
		"101023": {"Porvoo Emäsalo", 60.20382, 25.62546},
		"151028": {"Helsinki Vuosaari satama", 60.20867, 25.19590},
	}

	// Check that we received data for each expected station
	for stationID, expected := range expectedStations {
		data, exists := stationData[stationID]
		if !exists {
			t.Errorf("Missing data for station %s (%s)", stationID, expected.name)
			continue
		}

		// Verify station name
		if data.StationName != expected.name {
			t.Errorf("Station %s: expected name '%s', got '%s'",
				stationID, expected.name, data.StationName)
		}

		// Verify coordinates
		if data.Location.Lat != expected.lat || data.Location.Lon != expected.lon {
			t.Errorf("Station %s: expected coordinates (%.5f, %.5f), got (%.5f, %.5f)",
				stationID, expected.lat, expected.lon,
				data.Location.Lat, data.Location.Lon)
		}

		// Verify we have observations (should be 61 per station for 1 hour of minute data)
		if len(data.Observations) == 0 {
			t.Errorf("Station %s: no observations found", stationID)
		} else if len(data.Observations) != 61 {
			t.Logf("Station %s: expected 61 observations, got %d",
				stationID, len(data.Observations))
		}

		// Check first and last observation timestamps
		if len(data.Observations) > 0 {
			firstObs := data.Observations[0]
			lastObs := data.Observations[len(data.Observations)-1]

			// Verify timestamps are in expected range (around 2025-08-31 07:29 - 08:28 UTC)
			expectedStart := time.Date(2025, 8, 31, 7, 29, 0, 0, time.UTC)
			expectedEnd := time.Date(2025, 8, 31, 8, 28, 0, 0, time.UTC)

			if firstObs.Timestamp.Before(expectedStart.Add(-5*time.Minute)) ||
				firstObs.Timestamp.After(expectedStart.Add(5*time.Minute)) {
				t.Errorf("Station %s: first observation timestamp %v is outside expected range",
					stationID, firstObs.Timestamp)
			}

			if lastObs.Timestamp.Before(expectedEnd.Add(-5*time.Minute)) ||
				lastObs.Timestamp.After(expectedEnd.Add(5*time.Minute)) {
				t.Errorf("Station %s: last observation timestamp %v is outside expected range",
					stationID, lastObs.Timestamp)
			}

			// Verify wind data is present and reasonable
			validObsCount := 0
			for _, obs := range data.Observations {
				if obs.WindSpeed != nil && *obs.WindSpeed >= 0 && *obs.WindSpeed < 50 {
					validObsCount++
				}
			}

			if validObsCount == 0 {
				t.Errorf("Station %s: no valid wind speed observations", stationID)
			} else {
				t.Logf("Station %s: %d valid wind observations out of %d total",
					stationID, validObsCount, len(data.Observations))
			}
		}
	}

	// Verify we didn't get extra unexpected stations
	if len(stationData) != len(expectedStations) {
		t.Errorf("Expected %d stations, got %d",
			len(expectedStations), len(stationData))
	}
}

func TestMultiStationDataOrdering(t *testing.T) {
	// This test verifies that observations maintain correct time order
	xmlData, err := os.ReadFile("../../test_three_station_response.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	stationData := make(map[string]*StationWindData)

	callbacks := WindDataCallbacks{
		OnStationData: func(data StationWindData) error {
			stationData[data.StationID] = &data
			return nil
		},
	}

	parser := NewStationGroupingParser(bytes.NewReader(xmlData), callbacks)
	err = parser.Parse()
	if err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	// Check that observations are in chronological order for each station
	for stationID, data := range stationData {
		if len(data.Observations) < 2 {
			continue
		}

		for i := 1; i < len(data.Observations); i++ {
			prevTime := data.Observations[i-1].Timestamp
			currTime := data.Observations[i].Timestamp

			if currTime.Before(prevTime) {
				t.Errorf("Station %s: observations out of order at index %d: %v < %v",
					stationID, i, currTime, prevTime)
			}

			// For minute data, verify roughly 60-second intervals
			timeDiff := currTime.Sub(prevTime)
			if timeDiff < 50*time.Second || timeDiff > 70*time.Second {
				t.Logf("Station %s: unusual time interval at index %d: %v",
					stationID, i, timeDiff)
			}
		}
	}
}
