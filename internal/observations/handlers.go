package observations

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// RegisterHandlers registers the observation HTTP handlers
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
	mux.HandleFunc("/api/observations", handleObservations(mgr))
	mux.HandleFunc("/api/observations/latest", handleLatestObservations(mgr))
	mux.HandleFunc("/api/observations/", handleStationObservation(mgr))
}

// StationStatus represents station status for API responses
type StationStatus struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Region          string           `json:"region"`
	PollingInterval string           `json:"polling_interval"`
	LastPolled      time.Time        `json:"last_polled"`
	LastObservation time.Time        `json:"last_observation"`
	SuccessRate     float64          `json:"success_rate"`
	LatestData      *WindObservation `json:"latest_data,omitempty"`
}

// handleObservations handles the general observations endpoint
func handleObservations(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get all observations
		observations := mgr.GetAllLatestObservations()

		if err := json.NewEncoder(w).Encode(observations); err != nil {
			log.Printf("Error encoding observations response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// handleLatestObservations handles the latest observations endpoint
func handleLatestObservations(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get all latest observations
		observations := mgr.GetAllLatestObservations()

		// Convert to slice for easier consumption
		result := make([]WindObservation, 0, len(observations))
		for _, obs := range observations {
			result = append(result, obs)
		}

		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("Error encoding latest observations response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// handleStationObservation handles individual station observation lookup
func handleStationObservation(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract station ID from path
		path := strings.TrimPrefix(r.URL.Path, "/api/observations/")
		if path == "" {
			http.Error(w, "Station ID required", http.StatusBadRequest)
			return
		}

		observation, exists := mgr.GetLatestObservation(path)
		if !exists {
			http.Error(w, "Observation not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(observation); err != nil {
			log.Printf("Error encoding observation response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
