package stations

import "sync"

// manager implements the Station Manager interface
type manager struct {
	stations         []Station
	stationsByID     map[string]Station
	stationsByRegion map[string][]Station
	mu               sync.RWMutex
}

// NewManager creates a new station manager instance
func NewManager() Manager {
	m := &manager{
		stations:         make([]Station, 0),
		stationsByID:     make(map[string]Station),
		stationsByRegion: make(map[string][]Station),
	}

	// Load default stations from configuration
	m.loadDefaultStations()

	return m
}

// GetAllStations returns all available stations
func (m *manager) GetAllStations() []Station {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Station, len(m.stations))
	copy(result, m.stations)
	return result
}

// GetStation returns a specific station by ID
func (m *manager) GetStation(stationID string) (Station, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	station, exists := m.stationsByID[stationID]
	return station, exists
}

// GetStationsByRegion returns all stations in a specific region
func (m *manager) GetStationsByRegion(region string) []Station {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stations, exists := m.stationsByRegion[region]
	if !exists {
		return []Station{}
	}

	// Return a copy to prevent external modification
	result := make([]Station, len(stations))
	copy(result, stations)
	return result
}

// loadDefaultStations loads the hardcoded station configuration
func (m *manager) loadDefaultStations() {
	defaultStations := []Station{
		// Porkkala Area (KEY STATIONS)
		{ID: "101023", Name: "Emäsalo", Region: "Porvoo", Latitude: 60.2042, Longitude: 25.6258},
		{ID: "101022", Name: "Kalbådagrund", Region: "Porkkala", Latitude: 59.9747, Longitude: 24.5281},
		{ID: "105392", Name: "Itätoukki", Region: "Sipoo", Latitude: 60.2653, Longitude: 25.2097},
		{ID: "151028", Name: "Vuosaari", Region: "Helsinki", Latitude: 60.2075, Longitude: 25.1947},

		// Maritime & Coastal
		{ID: "100996", Name: "Harmaja", Region: "Helsinki Maritime", Latitude: 60.1042, Longitude: 24.9758},
		{ID: "100969", Name: "Bågaskär", Region: "Inkoo Coastal", Latitude: 59.9025, Longitude: 24.0419},
		{ID: "100965", Name: "Jussarö", Region: "Raasepori Maritime", Latitude: 59.8133, Longitude: 23.5639},
		{ID: "100946", Name: "Tulliniemi", Region: "Hanko Coastal", Latitude: 59.8458, Longitude: 22.9028},
		{ID: "100932", Name: "Russarö", Region: "Hanko Southern", Latitude: 59.7686, Longitude: 22.9533},
		{ID: "100945", Name: "Vänö", Region: "Kemiönsaari", Latitude: 59.8906, Longitude: 23.2569},
		{ID: "100908", Name: "Utö", Region: "Archipelago HELCOM", Latitude: 59.7800, Longitude: 21.3719},

		// Northern Coastal
		{ID: "101267", Name: "Tahkoluoto", Region: "Pori", Latitude: 61.6231, Longitude: 21.4081},
		{ID: "101661", Name: "Tankar", Region: "Kokkola", Latitude: 63.9583, Longitude: 23.2681},
		{ID: "101673", Name: "Ulkokalla", Region: "Kalajoki", Latitude: 64.3286, Longitude: 23.3442},
		{ID: "101784", Name: "Marjaniemi", Region: "Hailuoto", Latitude: 65.0361, Longitude: 24.5583},
		{ID: "101794", Name: "Vihreäsaari", Region: "Oulu", Latitude: 65.0403, Longitude: 25.4244},
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.stations = defaultStations

	// Build lookup maps
	m.stationsByID = make(map[string]Station)
	m.stationsByRegion = make(map[string][]Station)

	for _, station := range defaultStations {
		m.stationsByID[station.ID] = station
		m.stationsByRegion[station.Region] = append(m.stationsByRegion[station.Region], station)
	}
}
