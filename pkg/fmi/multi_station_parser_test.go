package fmi

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseMultiStationResponse(t *testing.T) {
	// Read test XML file
	xmlData, err := os.ReadFile(filepath.Join("testdata", "test_three_station_response.xml"))
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Parse the response
	reader := bytes.NewReader(xmlData)
	stations, err := ParseMultiStationResponse(reader)
	if err != nil {
		t.Fatalf("Failed to parse multi-station response: %v", err)
	}

	// Verify we got 3 stations
	if len(stations) != 3 {
		t.Errorf("Expected 3 stations, got %d", len(stations))
	}

	// Create map for easier testing
	stationMap := make(map[string]StationWindData)
	for _, station := range stations {
		stationMap[station.StationID] = station
	}

	// Test Harmaja station (100996)
	t.Run("Harmaja_Station", func(t *testing.T) {
		station, exists := stationMap["100996"]
		if !exists {
			t.Fatal("Station 100996 (Harmaja) not found")
		}

		if station.StationName != "Helsinki Harmaja" {
			t.Errorf("Expected station name 'Helsinki Harmaja', got '%s'", station.StationName)
		}

		if station.Location.Region != "Helsinki" {
			t.Errorf("Expected region 'Helsinki', got '%s'", station.Location.Region)
		}

		// Check coordinates (approximately)
		if station.Location.Lat < 60.10 || station.Location.Lat > 60.11 {
			t.Errorf("Unexpected latitude: %f", station.Location.Lat)
		}

		if station.Location.Lon < 24.97 || station.Location.Lon > 24.98 {
			t.Errorf("Unexpected longitude: %f", station.Location.Lon)
		}

		// Should have observations
		if len(station.Observations) == 0 {
			t.Error("No observations found for Harmaja")
		}

		// Check metadata
		if station.Metadata["wmo"] != "2795" {
			t.Errorf("Expected WMO code '2795', got '%s'", station.Metadata["wmo"])
		}
	})

	// Test Emäsalo station (101023)
	t.Run("Emasalo_Station", func(t *testing.T) {
		station, exists := stationMap["101023"]
		if !exists {
			t.Fatal("Station 101023 (Emäsalo) not found")
		}

		if station.StationName != "Porvoo Emäsalo" {
			t.Errorf("Expected station name 'Porvoo Emäsalo', got '%s'", station.StationName)
		}

		if station.Location.Region != "Porvoo" {
			t.Errorf("Expected region 'Porvoo', got '%s'", station.Location.Region)
		}

		// Should have observations
		if len(station.Observations) == 0 {
			t.Error("No observations found for Emäsalo")
		}
	})

	// Test Vuosaari station (151028)
	t.Run("Vuosaari_Station", func(t *testing.T) {
		station, exists := stationMap["151028"]
		if !exists {
			t.Fatal("Station 151028 (Vuosaari) not found")
		}

		if station.StationName != "Helsinki Vuosaari satama" {
			t.Errorf("Expected station name 'Helsinki Vuosaari satama', got '%s'", station.StationName)
		}

		// Should have observations
		if len(station.Observations) == 0 {
			t.Error("No observations found for Vuosaari")
		}
	})

	// Test observation data
	t.Run("Observation_Data", func(t *testing.T) {
		// Check first station's observations
		for _, station := range stations {
			if len(station.Observations) > 0 {
				obs := station.Observations[0]

				// Should have timestamp
				if obs.Timestamp.IsZero() {
					t.Error("Observation has zero timestamp")
				}

				// Should have at least one wind parameter
				if obs.WindSpeed == nil && obs.WindGust == nil && obs.WindDirection == nil {
					t.Error("Observation has no wind data")
				}

				// If wind speed exists, it should be reasonable
				if obs.WindSpeed != nil && (*obs.WindSpeed < 0 || *obs.WindSpeed > 100) {
					t.Errorf("Unreasonable wind speed: %f", *obs.WindSpeed)
				}

				// Wind direction should be 0-360
				if obs.WindDirection != nil && (*obs.WindDirection < 0 || *obs.WindDirection > 360) {
					t.Errorf("Invalid wind direction: %f", *obs.WindDirection)
				}

				break // Just test first station
			}
		}
	})
}

func TestCoordinateMatching(t *testing.T) {
	parser := &MultiStationParser{
		coordToStation: make(map[string]string),
		stations:       make(map[string]*StationMetadata),
	}

	// Add test stations
	parser.stations["100996"] = &StationMetadata{
		ID:   "100996",
		Name: "Harmaja",
		Lat:  60.10512,
		Lon:  24.97539,
	}

	parser.stations["101023"] = &StationMetadata{
		ID:   "101023",
		Name: "Emäsalo",
		Lat:  60.20382,
		Lon:  25.62546,
	}

	// Build coordinate index
	for id, metadata := range parser.stations {
		coordKey := formatCoordinateKey(metadata.Lat, metadata.Lon)
		parser.coordToStation[coordKey] = id
	}

	tests := []struct {
		name       string
		lat        float64
		lon        float64
		expectedID string
	}{
		{
			name:       "Exact_Harmaja_Coordinates",
			lat:        60.10512,
			lon:        24.97539,
			expectedID: "100996",
		},
		{
			name:       "Exact_Emasalo_Coordinates",
			lat:        60.20382,
			lon:        25.62546,
			expectedID: "101023",
		},
		{
			name:       "Unknown_Coordinates",
			lat:        61.00000,
			lon:        25.00000,
			expectedID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stationID := parser.getStationIDForCoordinate(tt.lat, tt.lon)
			if stationID != tt.expectedID {
				t.Errorf("Expected station ID '%s', got '%s'", tt.expectedID, stationID)
			}
		})
	}
}

func TestFormatCoordinateKey(t *testing.T) {
	tests := []struct {
		lat      float64
		lon      float64
		expected string
	}{
		{60.10512, 24.97539, "60.10512,24.97539"},
		{60.20382, 25.62546, "60.20382,25.62546"},
		{-12.34567, 123.45678, "-12.34567,123.45678"},
		{0.0, 0.0, "0.00000,0.00000"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatCoordinateKey(tt.lat, tt.lon)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestParseCoordinateString(t *testing.T) {
	tests := []struct {
		input   string
		wantLat float64
		wantLon float64
		wantErr bool
	}{
		{"60.10512 24.97539", 60.10512, 24.97539, false},
		{"60.10512 24.97539 ", 60.10512, 24.97539, false},
		{" 60.10512  24.97539 ", 60.10512, 24.97539, false},
		{"-12.345 123.456", -12.345, 123.456, false},
		{"60.10512", 0, 0, true}, // Only one coordinate
		{"", 0, 0, true},         // Empty string
		{"abc def", 0, 0, true},  // Invalid numbers
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			coords, err := parseCoordinateString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCoordinateString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if coords.Lat != tt.wantLat {
					t.Errorf("Expected lat %f, got %f", tt.wantLat, coords.Lat)
				}
				if coords.Lon != tt.wantLon {
					t.Errorf("Expected lon %f, got %f", tt.wantLon, coords.Lon)
				}
			}
		})
	}
}

func TestExtractParametersFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected []WindParameter
	}{
		{
			name:     "Three_Parameters",
			url:      "https://opendata.fmi.fi/meta?observableProperty=observation&param=windspeedms,windgust,winddirection&language=eng",
			expected: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
		},
		{
			name:     "Single_Parameter",
			url:      "https://example.com?param=windspeedms",
			expected: []WindParameter{WindSpeedMS},
		},
		{
			name:     "Two_Parameters",
			url:      "https://example.com?param=windgust,winddirection",
			expected: []WindParameter{WindGustMS, WindDirection},
		},
		{
			name:     "No_Parameters",
			url:      "https://example.com?other=value",
			expected: nil,
		},
		{
			name:     "Parameters_With_Spaces",
			url:      "https://example.com?param=windspeedms, windgust , winddirection",
			expected: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractParametersFromURL(tt.url)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d parameters, got %d", len(tt.expected), len(result))
				return
			}

			for i, param := range result {
				if param != tt.expected[i] {
					t.Errorf("Parameter %d: expected %s, got %s", i, tt.expected[i], param)
				}
			}
		})
	}
}

func TestParsePositions(t *testing.T) {
	parser := &MultiStationParser{}

	// Test valid positions string
	positionsStr := "60.10512 24.97539 1756627440 60.20382 25.62546 1756627500"

	positions, err := parser.parsePositions(positionsStr)
	if err != nil {
		t.Fatalf("Failed to parse positions: %v", err)
	}

	if len(positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(positions))
	}

	// Check first position
	if positions[0].Lat != 60.10512 {
		t.Errorf("Expected lat 60.10512, got %f", positions[0].Lat)
	}

	if positions[0].Lon != 24.97539 {
		t.Errorf("Expected lon 24.97539, got %f", positions[0].Lon)
	}

	expectedTime := time.Unix(1756627440, 0)
	if !positions[0].Timestamp.Equal(expectedTime) {
		t.Errorf("Expected timestamp %v, got %v", expectedTime, positions[0].Timestamp)
	}

	// Test invalid positions string
	invalidPositions := "60.10512 24.97539" // Missing timestamp
	_, err = parser.parsePositions(invalidPositions)
	if err == nil {
		t.Error("Expected error for invalid positions format")
	}
}

func TestParseDataValues(t *testing.T) {
	parser := &MultiStationParser{}

	dataStr := `5.3 5.8 238.0
5.2 5.7 239.0
NaN 6.1 240.0
4.8 NaN 241.0`

	values, err := parser.parseDataValues(dataStr)
	if err != nil {
		t.Fatalf("Failed to parse data values: %v", err)
	}

	if len(values) != 4 {
		t.Errorf("Expected 4 data rows, got %d", len(values))
	}

	// Check first row
	if values[0][0] != 5.3 {
		t.Errorf("Expected 5.3, got %f", values[0][0])
	}

	// Check NaN handling
	if values[2][0] != 0 {
		t.Errorf("Expected NaN to be converted to 0, got %f", values[2][0])
	}
}

func TestGzipSupport(t *testing.T) {
	// Create test data
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<wfs:FeatureCollection xmlns:wfs="http://www.opengis.net/wfs/2.0">
  <wfs:member/>
</wfs:FeatureCollection>`

	// Compress the data
	var gzipBuffer bytes.Buffer
	gzWriter := gzip.NewWriter(&gzipBuffer)
	if _, err := gzWriter.Write([]byte(testXML)); err != nil {
		t.Fatalf("Failed to write gzip data: %v", err)
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("Failed to close gzip writer: %v", err)
	}

	// Test parsing with gzip
	_, err := ParseMultiStationResponseWithGzip(&gzipBuffer, true)
	if err != nil && !strings.Contains(err.Error(), "no observation data") {
		t.Errorf("Unexpected error parsing gzipped data: %v", err)
	}

	// Test parsing without gzip
	plainReader := bytes.NewReader([]byte(testXML))
	_, err = ParseMultiStationResponseWithGzip(plainReader, false)
	if err != nil && !strings.Contains(err.Error(), "no observation data") {
		t.Errorf("Unexpected error parsing plain data: %v", err)
	}
}

func TestWindObservationCreation(t *testing.T) {
	parser := &MultiStationParser{
		paramIndices: map[WindParameter]int{
			WindSpeedMS:   0,
			WindGustMS:    1,
			WindDirection: 2,
		},
	}

	timestamp := time.Now()

	tests := []struct {
		name   string
		values []float64
		check  func(t *testing.T, obs WindObservation)
	}{
		{
			name:   "All_Values_Present",
			values: []float64{5.3, 6.8, 245.0},
			check: func(t *testing.T, obs WindObservation) {
				if obs.WindSpeed == nil || *obs.WindSpeed != 5.3 {
					t.Errorf("Expected wind speed 5.3, got %v", obs.WindSpeed)
				}
				if obs.WindGust == nil || *obs.WindGust != 6.8 {
					t.Errorf("Expected wind gust 6.8, got %v", obs.WindGust)
				}
				if obs.WindDirection == nil || *obs.WindDirection != 245.0 {
					t.Errorf("Expected wind direction 245.0, got %v", obs.WindDirection)
				}
			},
		},
		{
			name:   "Zero_Values_Skipped",
			values: []float64{0, 6.8, 0},
			check: func(t *testing.T, obs WindObservation) {
				if obs.WindSpeed != nil {
					t.Error("Wind speed should be nil for zero value")
				}
				if obs.WindGust == nil || *obs.WindGust != 6.8 {
					t.Errorf("Expected wind gust 6.8, got %v", obs.WindGust)
				}
				if obs.WindDirection != nil {
					t.Error("Wind direction should be nil for negative value")
				}
			},
		},
		{
			name:   "Direction_Zero_Allowed",
			values: []float64{5.0, 6.0, 0.0},
			check: func(t *testing.T, obs WindObservation) {
				if obs.WindDirection == nil || *obs.WindDirection != 0.0 {
					t.Error("Wind direction 0 should be allowed")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := parser.createWindObservation(timestamp, tt.values)
			if !obs.Timestamp.Equal(timestamp) {
				t.Errorf("Expected timestamp %v, got %v", timestamp, obs.Timestamp)
			}
			tt.check(t, obs)
		})
	}
}

func TestMultiStationDataOrdering(t *testing.T) {
	// Read test XML file
	xmlData, err := os.ReadFile(filepath.Join("testdata", "test_three_station_response.xml"))
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Parse the response
	reader := bytes.NewReader(xmlData)
	stations, err := ParseMultiStationResponse(reader)
	if err != nil {
		t.Fatalf("Failed to parse multi-station response: %v", err)
	}

	// Check that observations are in chronological order for each station
	for _, station := range stations {
		if len(station.Observations) < 2 {
			continue
		}

		for i := 1; i < len(station.Observations); i++ {
			prev := station.Observations[i-1].Timestamp
			curr := station.Observations[i].Timestamp

			if curr.Before(prev) {
				t.Errorf("Station %s: observations not in chronological order at index %d",
					station.StationID, i)
			}
		}
	}
}

// Benchmark parsing performance
func BenchmarkParseMultiStationResponse(b *testing.B) {
	xmlData, err := os.ReadFile(filepath.Join("testdata", "test_three_station_response.xml"))
	if err != nil {
		b.Fatalf("Failed to read test XML file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(xmlData)
		_, err := ParseMultiStationResponse(reader)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test edge cases
func TestEmptyResponse(t *testing.T) {
	emptyXML := `<?xml version="1.0" encoding="UTF-8"?>
<wfs:FeatureCollection xmlns:wfs="http://www.opengis.net/wfs/2.0">
</wfs:FeatureCollection>`

	reader := strings.NewReader(emptyXML)
	_, err := ParseMultiStationResponse(reader)

	if err == nil {
		t.Error("Expected error for empty response")
	}

	if !strings.Contains(err.Error(), "no observation data") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestMismatchedDataCounts(t *testing.T) {
	// This would test a malformed response where position count doesn't match data count
	// For now, we'll skip this as it requires constructing a complex malformed XML
	t.Skip("Test for mismatched data counts - requires malformed XML construction")
}

// Helper function for debugging
func TestPrintStationSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping summary in short mode")
	}

	xmlData, err := os.ReadFile(filepath.Join("testdata", "test_three_station_response.xml"))
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	reader := bytes.NewReader(xmlData)
	stations, err := ParseMultiStationResponse(reader)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	for _, station := range stations {
		fmt.Printf("Station %s (%s):\n", station.StationID, station.StationName)
		fmt.Printf("  Location: %.5f, %.5f (%s)\n",
			station.Location.Lat, station.Location.Lon, station.Location.Region)
		fmt.Printf("  Observations: %d\n", len(station.Observations))

		if len(station.Observations) > 0 {
			first := station.Observations[0]
			last := station.Observations[len(station.Observations)-1]
			fmt.Printf("  Time range: %s to %s\n",
				first.Timestamp.Format("15:04:05"),
				last.Timestamp.Format("15:04:05"))
		}
		fmt.Println()
	}
}
