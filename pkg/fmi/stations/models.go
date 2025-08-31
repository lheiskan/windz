// @vibe: 🤖 -- ai
package stations

import (
	"fmt"
	"time"
)

// Station represents a weather station
type Station struct {
	ID           string            `json:"id"`
	FMISID       string            `json:"fmisid"`
	Name         string            `json:"name"`
	Location     Coordinates       `json:"coordinates"`
	StartDate    time.Time         `json:"start_date"`
	EndDate      *time.Time        `json:"end_date,omitempty"`
	Network      string            `json:"network"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// StationCollection holds cached station data with metadata
type StationCollection struct {
	// When this data was last fetched from FMI
	LastUpdated time.Time `json:"lastUpdated"`

	// All weather stations
	Stations []Station `json:"stations"`
}

// Coordinates represents geographic location
type Coordinates struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region,omitempty"`
}

// BBox represents a geographic bounding box
type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// String returns the bounding box as a comma-separated string for API queries
func (b BBox) String() string {
	return fmt.Sprintf("%.2f,%.2f,%.2f,%.2f", b.MinLon, b.MinLat, b.MaxLon, b.MaxLat)
}

// Request represents a request for station metadata
type Request struct {
	BBox    *BBox
	Network Network
	UseGzip bool
}

// Response represents the parsed response from FMI stations API
type Response struct {
	Stations []Station `json:"stations"`
	Count    int       `json:"count"`
}

// Network represents weather station networks
type Network string

const (
	AWS   Network = "AWS"   // Automatic Weather Stations
	SYNOP Network = "SYNOP" // Synoptic stations
	MAREO Network = "MAREO" // Mareograph stations
	BUOY  Network = "BUOY"  // Buoy stations
)

// Predefined bounding boxes for convenience
var (
	FinlandBBox         = BBox{19.08, 59.45, 31.59, 70.09} // All Finland
	SouthernFinlandBBox = BBox{19.5, 59.7, 31.6, 61.8}     // Southern Finland
	CentralFinlandBBox  = BBox{22.0, 61.8, 31.0, 65.0}     // Central Finland
	NorthernFinlandBBox = BBox{20.0, 65.0, 31.6, 70.1}     // Northern Finland
)

// Collection methods

// IsStale checks if the station data is older than the specified duration
func (sc *StationCollection) IsStale(maxAge time.Duration) bool {
	return time.Since(sc.LastUpdated) > maxAge
}

// GetStationByID returns a station by its ID, or nil if not found
func (sc *StationCollection) GetStationByID(id string) *Station {
	for i := range sc.Stations {
		if sc.Stations[i].ID == id {
			return &sc.Stations[i]
		}
	}
	return nil
}

// GetStationByFMISID returns a station by its FMIS ID, or nil if not found
func (sc *StationCollection) GetStationByFMISID(fmisID string) *Station {
	for i := range sc.Stations {
		if sc.Stations[i].FMISID == fmisID {
			return &sc.Stations[i]
		}
	}
	return nil
}

// FilterByCapabilities returns stations that have all the specified capabilities
func (sc *StationCollection) FilterByCapabilities(requiredCapabilities []string) []Station {
	var filtered []Station

	for _, station := range sc.Stations {
		hasAllCapabilities := true

		for _, required := range requiredCapabilities {
			found := false
			for _, capability := range station.Capabilities {
				if capability == required {
					found = true
					break
				}
			}
			if !found {
				hasAllCapabilities = false
				break
			}
		}

		if hasAllCapabilities {
			filtered = append(filtered, station)
		}
	}

	return filtered
}

// FilterByBounds returns stations within the specified geographic bounds
func (sc *StationCollection) FilterByBounds(bbox BBox) []Station {
	var filtered []Station

	for _, station := range sc.Stations {
		if station.Location.Lat >= bbox.MinLat && station.Location.Lat <= bbox.MaxLat &&
			station.Location.Lon >= bbox.MinLon && station.Location.Lon <= bbox.MaxLon {
			filtered = append(filtered, station)
		}
	}

	return filtered
}

// FilterByNetwork returns stations that match the specified network
func (sc *StationCollection) FilterByNetwork(network string) []Station {
	var filtered []Station

	for _, station := range sc.Stations {
		if station.Network == network {
			filtered = append(filtered, station)
		}
	}

	return filtered
}

// GetDefaultWindCapabilities returns a default set of wind measurement capabilities
// This is used when we can't determine actual capabilities from API responses
func GetDefaultWindCapabilities() []string {
	return []string{
		"WS_PT1H_AVG", // Wind speed (hourly average)
		"WD_PT1H_AVG", // Wind direction (hourly average)
		"WG_PT1H_MAX", // Wind gust (hourly maximum)
	}
}

// Common measurement parameter codes used by FMI for wind data
var WindParameterCodes = map[string]string{
	"WS_PT1H_AVG":  "Wind speed (hourly average)",
	"WS_PT1H_MIN":  "Wind speed (hourly minimum)",
	"WS_PT1H_MAX":  "Wind speed (hourly maximum)",
	"WS_PT10M_AVG": "Wind speed (10-minute average)",
	"WD_PT1H_AVG":  "Wind direction (hourly average)",
	"WD_PT10M_AVG": "Wind direction (10-minute average)",
	"WG_PT1H_MAX":  "Wind gust (hourly maximum)",
}