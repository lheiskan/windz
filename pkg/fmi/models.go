package fmi

import (
	"encoding/xml"
	"fmt"
	"time"
)

// StationWindData represents all wind observations for a single station
type StationWindData struct {
	StationID    string            `json:"station_id"`
	StationName  string            `json:"station_name"`
	Location     Coordinates       `json:"coordinates"`
	Observations []WindObservation `json:"observations"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// WindObservation represents a single timestamped wind measurement
type WindObservation struct {
	Timestamp     time.Time `json:"timestamp"`
	WindSpeed     *float64  `json:"wind_speed_ms,omitempty"`
	WindGust      *float64  `json:"wind_gust_ms,omitempty"`
	WindDirection *float64  `json:"wind_direction_deg,omitempty"`
	Quality       string    `json:"quality,omitempty"`
}

// WindReading represents a single wind observation with station context
type WindReading struct {
	StationID     string      `json:"station_id"`
	StationName   string      `json:"station_name"`
	Network       string      `json:"network,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	Location      Coordinates `json:"coordinates"`
	WindSpeed     *float64    `json:"wind_speed_ms,omitempty"`
	WindGust      *float64    `json:"wind_gust_ms,omitempty"`
	WindDirection *float64    `json:"wind_direction_deg,omitempty"`
	Quality       string      `json:"quality,omitempty"`
}

// Coordinates represents geographic location
type Coordinates struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region,omitempty"`
}

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

// ProcessingStats provides summary of streaming operation
type ProcessingStats struct {
	TotalObservations     int           `json:"total_observations"`
	ProcessedObservations int           `json:"processed_observations"`
	SkippedObservations   int           `json:"skipped_observations"`
	StationCount          int           `json:"station_count"`
	ErrorCount            int           `json:"error_count"`
	Duration              time.Duration `json:"duration"`
	BytesProcessed        int64         `json:"bytes_processed"`
}

// WindParameter represents wind measurement parameters
type WindParameter string

const (
	WindSpeedMS   WindParameter = "windspeedms"
	WindGustMS    WindParameter = "windgust"
	WindDirection WindParameter = "winddirection"
)

// Network represents weather station networks
type Network string

const (
	AWS   Network = "AWS"   // Automatic Weather Stations
	SYNOP Network = "SYNOP" // Synoptic stations
	MAREO Network = "MAREO" // Mareograph stations
	BUOY  Network = "BUOY"  // Buoy stations
)

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

// Predefined bounding boxes for convenience
var (
	FinlandBBox         = BBox{19.08, 59.45, 31.59, 70.09} // All Finland
	SouthernFinlandBBox = BBox{19.5, 59.7, 31.6, 61.8}     // Southern Finland
	CentralFinlandBBox  = BBox{22.0, 61.8, 31.0, 65.0}     // Central Finland
	NorthernFinlandBBox = BBox{20.0, 65.0, 31.6, 70.1}     // Northern Finland
)

// FMI WFS XML parsing types for station queries

// WFSStationResponse represents the root WFS response for station queries
type WFSStationResponse struct {
	XMLName xml.Name           `xml:"FeatureCollection"`
	Members []WFSStationMember `xml:"member"`
}

// WFSStationMember represents a single station in the WFS response
type WFSStationMember struct {
	XMLName            xml.Name           `xml:"member"`
	MonitoringFacility MonitoringFacility `xml:"EnvironmentalMonitoringFacility"`
}

// MonitoringFacility represents the environmental monitoring facility (station)
type MonitoringFacility struct {
	XMLName    xml.Name      `xml:"EnvironmentalMonitoringFacility"`
	ID         string        `xml:"id,attr"`
	Identifier GMLIdentifier `xml:"identifier"`
	Names      []GMLName     `xml:"name"`
	StartDate  string        `xml:"operationalActivityPeriod>OperationalActivityPeriod>activityTime>TimePeriod>beginPosition"`
	Geometry   WFSGeometry   `xml:"representativePoint"`
	BelongsTo  []BelongsTo   `xml:"belongsTo"`
}

// GMLIdentifier represents an identifier element with codeSpace attribute
type GMLIdentifier struct {
	XMLName   xml.Name `xml:"identifier"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// GMLName represents a name element with codeSpace attribute
type GMLName struct {
	XMLName   xml.Name `xml:"name"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// WFSGeometry represents the geographic location of the station
type WFSGeometry struct {
	XMLName xml.Name `xml:"representativePoint"`
	Point   WFSPoint `xml:"Point"`
}

// WFSPoint represents a geographic point in WFS
type WFSPoint struct {
	XMLName     xml.Name `xml:"Point"`
	ID          string   `xml:"id,attr"`
	SrsName     string   `xml:"srsName,attr"`
	Coordinates string   `xml:"pos"`
}

// BelongsTo represents the network(s) the station belongs to
type BelongsTo struct {
	XMLName xml.Name `xml:"belongsTo"`
	Title   string   `xml:"title,attr"`
	Href    string   `xml:"href,attr"`
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
