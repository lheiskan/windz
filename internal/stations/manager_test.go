package stations

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	stations := mgr.GetAllStations()
	if len(stations) == 0 {
		t.Error("New manager should have stations loaded")
	}

	expectedStations := 16 // Based on the default stations list
	if len(stations) != expectedStations {
		t.Errorf("Expected %d stations, got %d", expectedStations, len(stations))
	}
}

func TestGetStation(t *testing.T) {
	mgr := NewManager()

	// Test existing station
	station, exists := mgr.GetStation("101023")
	if !exists {
		t.Error("Expected station 101023 to exist")
	}
	if station.Name != "Emäsalo" {
		t.Errorf("Expected station name 'Emäsalo', got '%s'", station.Name)
	}
	if station.Region != "Porvoo" {
		t.Errorf("Expected station region 'Porvoo', got '%s'", station.Region)
	}

	// Test non-existing station
	_, exists = mgr.GetStation("999999")
	if exists {
		t.Error("Expected non-existing station to not exist")
	}
}

func TestGetStationsByRegion(t *testing.T) {
	mgr := NewManager()

	// Test existing region
	stations := mgr.GetStationsByRegion("Helsinki")
	if len(stations) != 1 {
		t.Errorf("Expected 1 station in Helsinki region, got %d", len(stations))
	}
	if len(stations) > 0 && stations[0].ID != "151028" {
		t.Errorf("Expected station ID '151028' in Helsinki, got '%s'", stations[0].ID)
	}

	// Test non-existing region
	stations = mgr.GetStationsByRegion("NonExistentRegion")
	if len(stations) != 0 {
		t.Errorf("Expected 0 stations in non-existent region, got %d", len(stations))
	}

	// Test region with multiple stations (assuming some exist)
	porvoostations := mgr.GetStationsByRegion("Porvoo")
	if len(porvoostations) != 1 {
		t.Errorf("Expected 1 station in Porvoo region, got %d", len(porvoostations))
	}
}

func TestGetAllStations(t *testing.T) {
	mgr := NewManager()

	stations := mgr.GetAllStations()
	if len(stations) == 0 {
		t.Error("GetAllStations should return stations")
	}

	// Verify we get copies (not references that could be modified)
	originalCount := len(stations)
	stations[0].Name = "Modified Name"

	// Get stations again and verify the original data is unchanged
	stationsAgain := mgr.GetAllStations()
	if stationsAgain[0].Name == "Modified Name" {
		t.Error("GetAllStations should return copies, not references")
	}

	if len(stationsAgain) != originalCount {
		t.Error("Station count should remain consistent")
	}
}

func TestStationDataIntegrity(t *testing.T) {
	mgr := NewManager()

	// Verify all stations have required fields
	stations := mgr.GetAllStations()
	for _, station := range stations {
		if station.ID == "" {
			t.Error("Station ID should not be empty")
		}
		if station.Name == "" {
			t.Error("Station name should not be empty")
		}
		if station.Region == "" {
			t.Error("Station region should not be empty")
		}
		if station.Latitude == 0 && station.Longitude == 0 {
			t.Errorf("Station %s (%s) has invalid coordinates", station.ID, station.Name)
		}
	}
}

func TestStationCoordinates(t *testing.T) {
	mgr := NewManager()

	// Test specific station coordinates
	station, exists := mgr.GetStation("101023") // Emäsalo
	if !exists {
		t.Fatal("Expected station 101023 to exist")
	}

	// These coordinates should be in Finland (rough bounds check)
	if station.Latitude < 59.5 || station.Latitude > 70.0 {
		t.Errorf("Station latitude %f seems outside Finland bounds", station.Latitude)
	}
	if station.Longitude < 20.0 || station.Longitude > 32.0 {
		t.Errorf("Station longitude %f seems outside Finland bounds", station.Longitude)
	}
}
