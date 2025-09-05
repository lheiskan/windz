package observations

import (
	"context"
	"time"
)

// Manager defines the interface for observation polling and data management
type Manager interface {
	// Start begins the observation polling process
	Start(ctx context.Context) error

	// Stop stops the observation polling process
	Stop() error

	// GetLatestObservation returns the latest observation for a specific station
	GetLatestObservation(stationID string) (WindObservation, bool)

	// GetAllLatestObservations returns all latest observations indexed by station ID
	GetAllLatestObservations() map[string]WindObservation

	// GetPollingState returns the current polling state for a station
	GetPollingState(stationID string) (PollingState, bool)
}

// WindObservation represents a wind observation from FMI
type WindObservation struct {
	StationID     string    `json:"station_id"`
	StationName   string    `json:"station_name"`
	Region        string    `json:"region"`
	Timestamp     time.Time `json:"timestamp"`
	WindSpeed     float64   `json:"wind_speed"`
	WindGust      float64   `json:"wind_gust"`
	WindDirection float64   `json:"wind_direction"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// PollingState represents the adaptive polling state for a station
type PollingState struct {
	StationID         string        `json:"station_id"`
	CurrentInterval   time.Duration `json:"current_interval"`
	ConsecutiveMisses int           `json:"consecutive_misses"`
	LastPolled        time.Time     `json:"last_polled"`
	LastObservation   time.Time     `json:"last_observation"`
	SuccessRate       float64       `json:"success_rate"`
	TotalPolls        int           `json:"total_polls"`
	SuccessfulPolls   int           `json:"successful_polls"`
}
