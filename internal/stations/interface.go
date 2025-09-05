package stations

// Manager defines the interface for station metadata management
type Manager interface {
	// GetAllStations returns all available stations
	GetAllStations() []Station

	// GetStation returns a specific station by ID
	GetStation(stationID string) (Station, bool)

	// GetStationsByRegion returns all stations in a specific region
	GetStationsByRegion(region string) []Station
}

// Station represents a weather station with its metadata
type Station struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Region    string  `json:"region"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
