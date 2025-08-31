package fmi

import (
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

