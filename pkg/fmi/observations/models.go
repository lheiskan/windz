package observations

import (
	"fmt"
	"time"
)

// StationWindData represents wind observations for a single station
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

// Coordinates represents geographic location
type Coordinates struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region,omitempty"`
}

// ProcessingStats provides summary of parsing operation
type ProcessingStats struct {
	TotalObservations     int           `json:"total_observations"`
	ProcessedObservations int           `json:"processed_observations"`
	StationCount          int           `json:"station_count"`
	ErrorCount            int           `json:"error_count"`
	Duration              time.Duration `json:"duration"`
}

// WindParameter represents wind measurement parameters
type WindParameter string

const (
	WindSpeedMS   WindParameter = "windspeedms"
	WindGustMS    WindParameter = "windgust"
	WindDirection WindParameter = "winddirection"
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

// Request represents a request for wind observations
type Request struct {
	StartTime  time.Time
	EndTime    time.Time
	StationIDs []string
	BBox       *BBox
	Parameters []WindParameter
	UseGzip    bool
}

// Response represents the parsed response from FMI
type Response struct {
	Stations []StationWindData `json:"stations"`
	Stats    ProcessingStats   `json:"stats"`
}

// StationMetadata holds station information during parsing
type StationMetadata struct {
	ID     string
	Name   string
	Region string
	Lat    float64
	Lon    float64
	WMO    string
	GeoID  string
}

// PositionEntry represents a position with timestamp from FMI XML
type PositionEntry struct {
	Lat       float64
	Lon       float64
	Timestamp time.Time
}
