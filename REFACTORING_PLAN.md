# WindZ Refactoring Plan

## Overview
Refactor the monolithic `main.go` into a modular architecture with clear separation of concerns using three functional modules: SSE, Stations, and Observations.

## Architecture Design

### Module Organization
Each functional module contains:
- `interface.go` - Public API definition
- `manager.go` - Implementation of the interface
- `handlers.go` - HTTP/SSE endpoints (if applicable)

### Module Responsibilities

#### 1. **SSE Module** (`internal/sse/`)
- **Purpose**: Manage all Server-Sent Events functionality
- **Files**:
  - `interface.go` - SSEManager interface
  - `manager.go` - SSE client management implementation
  - `handlers.go` - SSE endpoint handler
- **Responsibilities**:
  - Track connected clients
  - Handle client registration/deregistration
  - Broadcast messages to all clients
  - Serve SSE endpoint

#### 2. **Stations Module** (`internal/stations/`)
- **Purpose**: Manage station metadata and information
- **Files**:
  - `interface.go` - StationManager interface
  - `manager.go` - Station metadata management
  - `handlers.go` - Station API endpoints (list stations, get station)
- **Responsibilities**:
  - Load station configuration
  - Provide station lookup by ID
  - Serve station metadata via HTTP API

#### 3. **Observations Module** (`internal/observations/`)
- **Purpose**: Handle weather observations and polling
- **Files**:
  - `interface.go` - ObservationManager interface
  - `manager.go` - Observation polling and data management
  - `handlers.go` - Observation API endpoints
- **Responsibilities**:
  - Poll FMI API for observations
  - Manage adaptive polling intervals
  - Maintain observation history
  - Persist/restore state
  - Publish updates via SSE module

## File Structure
```
windz/
├── main.go                          # Application initialization and wiring
├── internal/
│   ├── sse/
│   │   ├── interface.go           # SSEManager interface
│   │   ├── manager.go             # SSE implementation
│   │   └── handlers.go            # /events endpoint
│   ├── stations/
│   │   ├── interface.go           # StationManager interface
│   │   ├── manager.go             # Station management
│   │   └── handlers.go            # /api/stations endpoints
│   └── observations/
│       ├── interface.go           # ObservationManager interface
│       ├── manager.go             # Observation polling logic
│       └── handlers.go            # /api/observations endpoints
└── pkg/
    └── fmi/                        # Existing FMI API client (unchanged)
        └── observations/
```

## Module Interfaces

### SSE Module (`internal/sse/interface.go`)
```go
package sse

type Manager interface {
    AddClient(clientID string, client chan Message)
    RemoveClient(clientID string)
    HasClients() bool
    Broadcast(message Message)
}

type Message struct {
    ID        int64       
    Type      string      
    StationID string      
    Data      interface{} 
    Timestamp time.Time   
}
```

### Stations Module (`internal/stations/interface.go`)
```go
package stations

type Manager interface {
    GetAllStations() []Station
    GetStation(stationID string) (Station, error)
}

type Station struct {
    ID        string  
    Name      string  
    Region    string  
    Latitude  float64 
    Longitude float64 
}
```

### Observations Module (`internal/observations/interface.go`)
```go
package observations

type Manager interface {
    Start(ctx context.Context) error
    Stop() error
    GetLatestObservation(stationID string) (WindObservation, error)
    GetAllLatestObservations() map[string]WindObservation
}

type WindObservation struct {
    StationID     string    
    StationName   string    
    Region        string    
    Timestamp     time.Time 
    WindSpeed     float64   
    WindGust      float64   
    WindDirection float64   
    UpdatedAt     time.Time 
}
```

## Implementation Steps

### Phase 1: Create Module Structure
1. **Create SSE Module**
   - Define interface
   - Move SSE logic from main.go to manager.go
   - Extract SSE handler to handlers.go

2. **Create Stations Module**
   - Define interface
   - Extract station data and logic to manager.go
   - Create station API handlers

3. **Create Observations Module**
   - Define interface
   - Move polling logic to manager.go
   - Create observation API handlers

### Phase 2: Wire Everything in main.go
```go
func main() {
    // Initialize managers
    sseManager := sse.NewManager()
    stationManager := stations.NewManager()
    observationManager := observations.NewManager(
        stationManager,
        sseManager,
        fmiClient,
        "wind_data.json",
    )

    // Setup HTTP routes
    mux := http.NewServeMux()
    
    // Module handlers
    sse.RegisterHandlers(mux, sseManager)
    stations.RegisterHandlers(mux, stationManager)
    observations.RegisterHandlers(mux, observationManager)
    
    // Start observation polling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go observationManager.Start(ctx)
    
    // Start server
    server := &http.Server{
        Addr:    ":8081",
        Handler: mux,
    }
    
    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        <-sigChan
        cancel()
        observationManager.Stop()
        server.Shutdown(context.Background())
    }()
    
    log.Fatal(server.ListenAndServe())
}
```

### Phase 3: Module Handler Registration
Each module provides a registration function:

```go
// internal/sse/handlers.go
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
    mux.HandleFunc("/events", handleSSE(mgr))
}

// internal/stations/handlers.go  
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
    mux.HandleFunc("/api/stations", handleStations(mgr))
    mux.HandleFunc("/api/stations/", handleStation(mgr))
}

// internal/observations/handlers.go
func RegisterHandlers(mux *http.ServeMux, mgr Manager) {
    mux.HandleFunc("/api/observations", handleObservations(mgr))
    mux.HandleFunc("/api/observations/latest", handleLatest(mgr))
}
```

## Benefits of This Organization

1. **Functional Cohesion**
   - All SSE-related code in one place
   - All station-related code together
   - All observation logic grouped

2. **Clear Dependencies**
   - Observations depends on SSE and Stations
   - SSE and Stations are independent
   - Dependencies flow in one direction

3. **Easy to Navigate**
   - Want to change SSE? Look in `internal/sse/`
   - Need station logic? Check `internal/stations/`
   - Observation polling? It's in `internal/observations/`

4. **Testability**
   - Each module can be tested independently
   - Handlers can be tested with mock managers
   - Managers can be tested with mock dependencies

5. **Maintainability**
   - Add new station endpoints? Update `stations/handlers.go`
   - Change polling logic? Modify `observations/manager.go`
   - Update SSE protocol? Touch only `sse/` module

## Migration Strategy

1. **Create module structure** with empty implementations
2. **Move code piece by piece** from main.go to modules
3. **Test each module** as it's completed
4. **Wire modules together** in simplified main.go
5. **Remove old code** from main.go
6. **Validate end-to-end** functionality

## Next Steps
1. Create the `internal/` directory structure
2. Start with SSE module (least dependencies)
3. Implement Stations module (static data)
4. Implement Observations module (most complex)
5. Refactor main.go to use modules
6. Add tests for each module

@vibe: 🤖 -- ai