package stations

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// RegisterHandlers registers the station HTTP handlers
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
	mux.HandleFunc("/api/stations", handleStations(mgr))
	mux.HandleFunc("/api/stations/", handleStation(mgr))
}

// handleStations handles the stations list endpoint
func handleStations(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Check for region filter
		region := r.URL.Query().Get("region")

		var stations []Station
		if region != "" {
			stations = mgr.GetStationsByRegion(region)
		} else {
			stations = mgr.GetAllStations()
		}

		if err := json.NewEncoder(w).Encode(stations); err != nil {
			log.Printf("Error encoding stations response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}

// handleStation handles individual station lookup
func handleStation(mgr Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract station ID from path
		path := strings.TrimPrefix(r.URL.Path, "/api/stations/")
		if path == "" {
			http.Error(w, "Station ID required", http.StatusBadRequest)
			return
		}

		station, exists := mgr.GetStation(path)
		if !exists {
			http.Error(w, "Station not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(station); err != nil {
			log.Printf("Error encoding station response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
}
