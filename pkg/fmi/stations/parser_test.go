// @vibe: 🤖 -- ai
package stations

import (
	"strings"
	"testing"
)

func TestParseStationsXML(t *testing.T) {
	// Simple test XML for station parsing
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<wfs:FeatureCollection xmlns:wfs="http://www.opengis.net/wfs/2.0"
                       xmlns:ef="http://inspire.ec.europa.eu/schemas/ef/4.0"
                       xmlns:gml="http://www.opengis.net/gml/3.2">
  <wfs:member>
    <ef:EnvironmentalMonitoringFacility gml:id="station-100996">
      <gml:identifier codeSpace="http://xml.fmi.fi/namespace/stationcode/fmisid">100996</gml:identifier>
      <gml:name codeSpace="http://xml.fmi.fi/namespace/locationcode/name">Helsinki Harmaja</gml:name>
      <ef:representativePoint>
        <gml:Point>
          <gml:pos>60.10512 24.97539</gml:pos>
        </gml:Point>
      </ef:representativePoint>
      <ef:operationalActivityPeriod>
        <ef:OperationalActivityPeriod>
          <ef:activityTime>
            <gml:TimePeriod>
              <gml:beginPosition>2000-01-01T00:00:00Z</gml:beginPosition>
            </gml:TimePeriod>
          </ef:activityTime>
        </ef:OperationalActivityPeriod>
      </ef:operationalActivityPeriod>
      <ef:belongsTo title="AWS"/>
    </ef:EnvironmentalMonitoringFacility>
  </wfs:member>
</wfs:FeatureCollection>`

	parser := NewParser()
	reader := strings.NewReader(testXML)
	
	response, err := parser.ParseXML(reader)
	if err != nil {
		t.Fatalf("Failed to parse stations XML: %v", err)
	}

	// Verify response
	if response.Count != 1 {
		t.Errorf("Expected 1 station, got %d", response.Count)
	}

	if len(response.Stations) != 1 {
		t.Errorf("Expected 1 station in array, got %d", len(response.Stations))
	}

	// Verify station details
	station := response.Stations[0]
	if station.FMISID != "100996" {
		t.Errorf("Expected FMISID '100996', got '%s'", station.FMISID)
	}

	if station.Name != "Helsinki Harmaja" {
		t.Errorf("Expected name 'Helsinki Harmaja', got '%s'", station.Name)
	}

	if station.Location.Lat != 60.10512 {
		t.Errorf("Expected latitude 60.10512, got %f", station.Location.Lat)
	}

	if station.Location.Lon != 24.97539 {
		t.Errorf("Expected longitude 24.97539, got %f", station.Location.Lon)
	}

	if station.Network != "AWS" {
		t.Errorf("Expected network 'AWS', got '%s'", station.Network)
	}
}

func TestParseCoordinates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []float64
	}{
		{
			name:     "Valid_Coordinates",
			input:    "60.10512 24.97539",
			expected: []float64{60.10512, 24.97539},
		},
		{
			name:     "Empty_String",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single_Value",
			input:    "60.10512",
			expected: nil,
		},
		{
			name:     "Three_Values",
			input:    "60.10512 24.97539 100.0",
			expected: []float64{60.10512, 24.97539, 100.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCoordinates(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d coordinates, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Coordinate %d: expected %f, got %f", i, expected, result[i])
				}
			}
		})
	}
}

func TestIsValidCoordinate(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lon  float64
		want bool
	}{
		{"Valid_Finnish_Coords", 60.10512, 24.97539, true},
		{"Valid_Northern_Finland", 70.0, 30.0, true},
		{"Invalid_Too_South", 58.0, 25.0, false},
		{"Invalid_Too_North", 72.0, 25.0, false},
		{"Invalid_Too_West", 65.0, 18.0, false},
		{"Invalid_Too_East", 65.0, 33.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCoordinate(tt.lat, tt.lon)
			if result != tt.want {
				t.Errorf("isValidCoordinate(%f, %f) = %v, want %v", tt.lat, tt.lon, result, tt.want)
			}
		})
	}
}

func TestExtractStationName(t *testing.T) {
	tests := []struct {
		name     string
		names    []GMLName
		expected string
	}{
		{
			name: "Finnish_Name",
			names: []GMLName{
				{CodeSpace: "http://xml.fmi.fi/namespace/locationcode/name", Value: "Helsinki Harmaja"},
			},
			expected: "Helsinki Harmaja",
		},
		{
			name: "Multiple_Names",
			names: []GMLName{
				{CodeSpace: "other", Value: "Other Name"},
				{CodeSpace: "http://xml.fmi.fi/namespace/locationcode/name", Value: "Helsinki Harmaja"},
			},
			expected: "Helsinki Harmaja",
		},
		{
			name:     "Empty_Names",
			names:    []GMLName{},
			expected: "",
		},
		{
			name: "Fallback_Name",
			names: []GMLName{
				{CodeSpace: "other", Value: "Fallback Name"},
			},
			expected: "Fallback Name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStationName(tt.names)
			if result != tt.expected {
				t.Errorf("extractStationName() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestExtractFMISID(t *testing.T) {
	tests := []struct {
		name       string
		identifier GMLIdentifier
		expected   string
	}{
		{
			name: "Valid_FMISID",
			identifier: GMLIdentifier{
				CodeSpace: "http://xml.fmi.fi/namespace/stationcode/fmisid",
				Value:     "100996",
			},
			expected: "100996",
		},
		{
			name: "Numeric_Fallback",
			identifier: GMLIdentifier{
				CodeSpace: "other",
				Value:     "123456",
			},
			expected: "123456",
		},
		{
			name: "Non_Numeric_Fallback",
			identifier: GMLIdentifier{
				CodeSpace: "other",
				Value:     "abc123",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFMISID(tt.identifier)
			if result != tt.expected {
				t.Errorf("extractFMISID() = '%s', want '%s'", result, tt.expected)
			}
		})
	}
}

func TestStationCollectionMethods(t *testing.T) {
	// Create test collection
	collection := &StationCollection{
		Stations: []Station{
			{
				ID:     "station-1",
				FMISID: "100996",
				Name:   "Harmaja",
				Location: Coordinates{
					Lat: 60.10512,
					Lon: 24.97539,
				},
				Network:      "AWS",
				Capabilities: []string{"WS_PT1H_AVG", "WD_PT1H_AVG"},
			},
			{
				ID:     "station-2",
				FMISID: "101023",
				Name:   "Emäsalo",
				Location: Coordinates{
					Lat: 60.20382,
					Lon: 25.62546,
				},
				Network:      "SYNOP",
				Capabilities: []string{"WS_PT1H_AVG"},
			},
		},
	}

	// Test GetStationByID
	station := collection.GetStationByID("station-1")
	if station == nil || station.Name != "Harmaja" {
		t.Error("GetStationByID failed to find station-1")
	}

	// Test GetStationByFMISID
	station = collection.GetStationByFMISID("101023")
	if station == nil || station.Name != "Emäsalo" {
		t.Error("GetStationByFMISID failed to find station 101023")
	}

	// Test FilterByNetwork
	awsStations := collection.FilterByNetwork("AWS")
	if len(awsStations) != 1 || awsStations[0].Name != "Harmaja" {
		t.Error("FilterByNetwork failed for AWS network")
	}

	// Test FilterByCapabilities
	windSpeedStations := collection.FilterByCapabilities([]string{"WS_PT1H_AVG"})
	if len(windSpeedStations) != 2 {
		t.Errorf("FilterByCapabilities: expected 2 stations with WS_PT1H_AVG, got %d", len(windSpeedStations))
	}

	bothCapStations := collection.FilterByCapabilities([]string{"WS_PT1H_AVG", "WD_PT1H_AVG"})
	if len(bothCapStations) != 1 {
		t.Errorf("FilterByCapabilities: expected 1 station with both capabilities, got %d", len(bothCapStations))
	}

	// Test FilterByBounds
	bbox := BBox{MinLat: 60.0, MaxLat: 60.15, MinLon: 24.0, MaxLon: 25.0}
	inBounds := collection.FilterByBounds(bbox)
	if len(inBounds) != 1 || inBounds[0].Name != "Harmaja" {
		t.Error("FilterByBounds failed")
	}
}

func TestBBoxString(t *testing.T) {
	bbox := BBox{
		MinLon: 24.0,
		MinLat: 60.0,
		MaxLon: 25.0,
		MaxLat: 61.0,
	}

	result := bbox.String()
	expected := "24.00,60.00,25.00,61.00"

	if result != expected {
		t.Errorf("BBox.String() = '%s', want '%s'", result, expected)
	}
}