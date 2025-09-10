package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"windz/internal/observations"
	"windz/internal/sse"
	"windz/internal/stations"
)

// Build metadata - injected at build time
var (
	BuildDate    = "unknown"
	BuildCommit  = "unknown"
	BuildVersion = "dev"
)

var (
	port         = flag.Int("port", 8080, "HTTP server port")
	stateFile    = flag.String("state-file", "polling_state.json", "Polling state persistence file")
	windDataFile = flag.String("wind-data-file", "wind_data.json", "Wind data cache persistence file")
	debug        = flag.Bool("debug", false, "Enable debug logging")
)

func main() {
	flag.Parse()

	log.Printf("WindZ Monitor starting on port %d", *port)
	log.Printf("Build: %s (%s) - %s", BuildVersion, BuildCommit, BuildDate)

	// Initialize managers
	sseManager := sse.NewManager()
	stationManager := stations.NewManager()
	observationManager := observations.NewManager(
		stationManager,
		sseManager,
		*stateFile,
		*windDataFile,
		*debug,
	)

	allStations := stationManager.GetAllStations()
	log.Printf("Monitoring %d Finnish weather stations", len(allStations))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup HTTP server
	mux := http.NewServeMux()

	// Legacy handlers (for backward compatibility with existing HTML)
	mux.HandleFunc("/", handleIndex(stationManager, observationManager))
	mux.HandleFunc("/health", handleHealth(stationManager, sseManager))
	mux.HandleFunc("/metrics", handleMetrics(observationManager))

	// Set up callback for SSE client connections to send initial data
	sseManager.SetClientConnectCallback(func(clientID string) {
		// Get all current observations
		allObservations := observationManager.GetAllLatestObservations()

		// Send each observation as a data message to the specific client
		for stationID, obs := range allObservations {
			dataMsg := sse.Message{
				ID:        obs.UpdatedAt.Unix(),
				Type:      "data",
				StationID: stationID,
				Data:      obs,
			}
			sseManager.SendToClient(clientID, dataMsg)
		}

		log.Printf("Sent %d initial observations to SSE client %s", len(allObservations), clientID)
	})

	// Register module handlers
	sse.RegisterHandlers(mux, sseManager)
	stations.RegisterHandlers(mux, stationManager)
	observations.RegisterHandlers(mux, observationManager)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	// Start observation polling
	go func() {
		if err := observationManager.Start(ctx); err != nil {
			log.Printf("Error starting observation manager: %v", err)
		}
	}()

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		cancel()

		if err := observationManager.Stop(); err != nil {
			log.Printf("Error stopping observation manager: %v", err)
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		server.Shutdown(shutdownCtx)
	}()

	log.Printf("Server starting at http://localhost:%d", *port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// Legacy handlers for backward compatibility

func handleIndex(stationMgr stations.Manager, obsMgr observations.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		// Get all stations and their data
		allStations := stationMgr.GetAllStations()
		allObservations := obsMgr.GetAllLatestObservations()

		// Create template data structure similar to original
		type StationRowData struct {
			ID           string
			Name         string
			Region       string
			WindData     *observations.WindObservation
			PollingState *observations.PollingState
		}

		var stationRows []StationRowData
		for _, station := range allStations {
			var windData *observations.WindObservation
			var pollingState *observations.PollingState

			if obs, exists := allObservations[station.ID]; exists {
				windData = &obs
			}

			if state, exists := obsMgr.GetPollingState(station.ID); exists {
				pollingState = &state
			}

			stationRows = append(stationRows, StationRowData{
				ID:           station.ID,
				Name:         station.Name,
				Region:       station.Region,
				WindData:     windData,
				PollingState: pollingState,
			})
		}

		templateData := struct {
			Stations []StationRowData
		}{
			Stations: stationRows,
		}

		// For now, return a simple response
		// The full HTML template from the original main.go can be moved to a separate template file
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
    <title>WindZ Monitor - Modular Architecture</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .station { margin: 10px 0; padding: 10px; border: 1px solid #ccc; }
        .data { color: green; }
        .no-data { color: #999; }
    </style>
</head>
<body>
    <h1>WindZ Monitor</h1>
    <p>Modular architecture with %d stations monitored</p>
    <div class="stations">`, len(templateData.Stations))

		for _, station := range templateData.Stations {
			status := "no-data"
			dataText := "No data"
			if station.WindData != nil {
				status = "data"
				dataText = fmt.Sprintf("%.1f m/s, gust %.1f m/s, %s",
					station.WindData.WindSpeed,
					station.WindData.WindGust,
					station.WindData.UpdatedAt.Format("15:04"))
			}

			fmt.Fprintf(w, `
        <div class="station">
            <strong>%s</strong> - %s<br>
            <span class="%s">%s</span>
        </div>`, station.Name, station.Region, status, dataText)
		}

		fmt.Fprint(w, `
    </div>
    <p><a href="/api/stations">View Stations API</a> | <a href="/api/observations/latest">View Latest Observations</a></p>
    <script>
        let stationData = new Map();
		let eventSource = connectSSE();

        function connectSSE() {
			let eventSource = new EventSource('/events');
            eventSource.onopen = function() {
                console.log('SSE connected event');
				updateConnectionStatus(getState())
            };
            eventSource.addEventListener('connected', function(event) {
                console.log('SSE connection confirmed:', event.data);
            });
            eventSource.addEventListener('data', function(event) {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.data) {
                        updateStationData(msg.data);
                        stationData.set(msg.station_id, msg.data);
                    }
                } catch (e) {
                    console.error('Error parsing SSE data:', e);
                }
            });
            eventSource.onerror = function() {
                console.log('SSE connection error');
				updateConnectionStatus(getState())
            };
			return eventSource;
        }


        function updateStationData(data) {
            // Simple DOM update for the station data
            const stationDivs = document.querySelectorAll('.station');
            stationDivs.forEach(div => {
                const stationName = div.querySelector('strong').textContent;
                if (data.station_name === stationName) {
                    const dataSpan = div.querySelector('span');
                    const windSpeed = data.wind_speed >= 0 ? data.wind_speed.toFixed(1) : '-';
                    const windGust = data.wind_gust >= 0 ? data.wind_gust.toFixed(1) : '-';
                    const time = new Date(data.updated_at).toLocaleTimeString('en-GB', {hour: '2-digit', minute: '2-digit'});
                    
                    dataSpan.textContent = windSpeed + ' m/s, gust ' + windGust + ' m/s, ' + time;
                    dataSpan.className = 'data';
                }
            });
        }

		function getState() {
		  let state = "empty"
		  if(eventSource) {
			  switch(eventSource.readyState) {
				case 0:
					state = "connecting";
					break;
				case 1:
					state = "connected";
					break;
				case 2:
					state = "disconnected";
					break;
			  }
		  }
		  return state
		}

        function updateConnectionStatus(state) {
			console.log("updating connection state: " + state)
            // Add a simple connection indicator
            let indicator = document.getElementById('connection-status');
            if (!indicator) {
                indicator = document.createElement('div');
                indicator.id = 'connection-status';
                indicator.style.cssText = 'position:fixed;top:10px;right:10px;padding:5px 10px;border-radius:3px;font-size:12px;z-index:1000;';
                document.body.appendChild(indicator);
            }
			switch (state) {
				case "connected":
				  indicator.textContent = "🟢 Connected";
				  indicator.style.background = "#d4edda";
				  indicator.style.color = "#155724";
				  break;
				case "disconnected":
				  indicator.textContent = "🔴 Disconnected";
				  indicator.style.background = "#f8d7da";
				  indicator.style.color = "#721c24";
				  break;
				case "connecting":
				  indicator.textContent = "🟡 Connecting…";
				  indicator.style.background = "#fff3cd";
				  indicator.style.color = "#856404";
				  break;
				case "empty":
				  indicator.textContent = "⚫ No event source";
				  indicator.style.background = "#e2e3e5";
				  indicator.style.color = "#383d41";
				  break;
			}
		}

        document.addEventListener('visibilitychange', function() {
			updateConnectionStatus(getState())
        });

    </script>
</body>
</html>`)
	}
}

func handleHealth(stationMgr stations.Manager, sseMgr sse.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		allStations := stationMgr.GetAllStations()
		clientCount := sseMgr.ClientCount()

		fmt.Fprintf(w, `{
    "status": "ok",
    "stations": %d,
    "sse_clients": %d,
    "build": {
        "version": "%s",
        "commit": "%s",
        "date": "%s"
    }
}`, len(allStations), clientCount, BuildVersion, BuildCommit, BuildDate)
	}
}

func handleMetrics(obsMgr observations.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		allObservations := obsMgr.GetAllLatestObservations()

		active, withData := 0, 0
		for _, obs := range allObservations {
			active++
			if obs.WindSpeed >= 0 {
				withData++
			}
		}

		fmt.Fprintf(w, `{
    "total_stations": %d,
    "stations_with_data": %d,
    "build": {
        "version": "%s",
        "commit": "%s",
        "date": "%s"
    }
}`, active, withData, BuildVersion, BuildCommit, BuildDate)
	}
}
