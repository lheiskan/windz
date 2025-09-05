# WindZ Monitor Architecture

## Overview

WindZ Monitor is built with a clean, modular architecture that separates concerns into three main functional modules. The design emphasizes maintainability, testability, and performance.

## Design Principles

### 🏗️ **Functional Organization**
- Code is organized by business function (SSE, Stations, Observations) rather than technical layers
- Each module has clear, single responsibility
- Dependencies flow in one direction: `Observations → SSE + Stations`

### 🔌 **Interface-Driven Design**
- All modules expose clean interfaces for loose coupling
- Dependency injection enables easy testing and modularity
- Mock implementations available for all interfaces in tests

### 🔒 **Thread Safety**
- All shared state protected by appropriate mutexes
- Minimal lock contention through careful design
- Value semantics (copies) preferred over shared pointers

### 📦 **Encapsulation**
- Implementation details hidden within modules
- Clean API boundaries between modules
- No circular dependencies

## Module Structure

Each functional module follows a consistent structure:

```
internal/{module}/
├── interface.go    # Public API definition
├── manager.go      # Implementation
├── handlers.go     # HTTP endpoints (if applicable)
└── manager_test.go # Comprehensive tests
```

### 📡 **SSE Module** (`internal/sse/`)

**Purpose**: Manage all Server-Sent Events functionality for real-time updates

**Key Components**:
- `Manager`: Thread-safe client connection management
- `Message`: Standardized SSE message structure with timestamps
- Battery-saving client connection callbacks

**Features**:
- Automatic client registration/cleanup
- Targeted messaging to specific clients
- Callback system for initial data sending
- Connection status tracking

**Thread Safety**:
- Client map protected by `sync.RWMutex`
- Callback registration protected by separate mutex
- Non-blocking client notifications via goroutines

### 🏢 **Stations Module** (`internal/stations/`)

**Purpose**: Manage weather station metadata and information

**Key Components**:
- Station metadata with coordinates
- Regional grouping and lookup
- Future-ready for map functionality

**Features**:
- In-memory station database with efficient lookups
- Regional filtering support
- Coordinate data for future mapping features
- Thread-safe concurrent access

**Data Structure**:
```go
type Station struct {
    ID        string  `json:"id"`
    Name      string  `json:"name"` 
    Region    string  `json:"region"`
    Latitude  float64 `json:"latitude"`
    Longitude float64 `json:"longitude"`
}
```

### 🌊 **Observations Module** (`internal/observations/`)

**Purpose**: Handle weather observation polling, data management, and FMI API integration

**Key Components**:
- Adaptive polling algorithm
- FMI API batching and optimization
- Data persistence and state management
- Real-time broadcasting via SSE

**Features**:
- Intelligent polling intervals (1m → 10m → 60m → 24h)
- Multi-station API batching (up to 20 stations per call)
- Time window grouping for maximum efficiency
- Automatic state persistence
- Thread-safe concurrent operations

**Concurrency Design**:
- **Three-Phase Polling**: Collect → Process → Update
- **Minimal Lock Contention**: Locks held only during map operations (microseconds)
- **Value Semantics**: Work with copies during processing
- **Non-Blocking**: Network operations don't hold locks

## Battery-Saving SSE Architecture

### 🔋 **Page Visibility Integration**

The application implements intelligent SSE management to optimize mobile battery usage:

```javascript
// Client-side: Page Visibility API
document.addEventListener('visibilitychange', function() {
    if (document.hidden) {
        // Disconnect SSE to save battery
        eventSource.close();
    } else {
        // Reconnect and get fresh data
        connectSSE();
    }
});
```

### 🔄 **Reconnection Flow**

1. **Client Connects** → `handleSSE()` registers client
2. **Callback Triggered** → `NotifyClientConnected()` called
3. **Initial Data Sent** → All current observations via `SendToClient()`
4. **Page Hidden** → Client disconnects (JavaScript)
5. **Page Visible** → Client reconnects → Fresh data automatically sent

### 📡 **SSE Message Flow**

```
New Client Connection
    ↓
SSE Handler Registration
    ↓
Client Connect Callback (Goroutine)
    ↓
Observation Manager Queries Current Data
    ↓
Individual Messages via SendToClient()
    ↓
Client Receives Complete Dataset
```

## Performance Optimizations

### 🚀 **Polling Efficiency**

**Adaptive Algorithm**:
- Stations start at 1-minute intervals
- Back off after consecutive misses: 1m → 10m → 60m → 24h
- Speed up when faster data detected
- SSE client presence affects polling frequency

**API Batching**:
- Time window grouping (stations with similar `LastObservation` times)
- Up to 20 stations per API call
- Typically reduces API calls by 75-90%
- GZIP compression for bandwidth optimization

### 🧵 **Concurrency Performance**

**Lock Optimization**:
```go
// Before: Long lock hold during network operations
m.pollingStatesMutex.Lock()
defer m.pollingStatesMutex.Unlock()
// ... network calls that take seconds ...

// After: Brief locks only for map operations
m.pollingStatesMutex.Lock()
toPoll := []PollingState{} // collect work
m.pollingStatesMutex.Unlock()
// ... do network calls without locks ...
m.pollingStatesMutex.Lock() 
// ... write results back ...
m.pollingStatesMutex.Unlock()
```

**Benefits**:
- HTTP handlers never blocked during polling
- SSE connections remain responsive
- Concurrent operations perform optimally

### 📊 **Memory Efficiency**

- Value semantics prevent accidental sharing
- Bounded channel sizes (100-message buffer)
- Efficient map lookups with pre-built indices
- Minimal garbage collection pressure

## Testing Strategy

### 🧪 **Comprehensive Test Coverage**

Each module includes:
- **Unit Tests**: Core functionality and edge cases
- **Concurrency Tests**: Thread safety validation
- **Integration Tests**: Module interaction verification
- **Mock Implementations**: Clean test isolation

**Test Categories**:
- Interface compliance verification
- Thread safety (concurrent operations)
- Error handling and recovery
- Performance characteristics

### 🎭 **Mock Strategy**

Clean mocking enables isolated testing:
```go
type mockSSEManager struct {
    messages []sse.Message
    clients  int
}

func (m *mockSSEManager) Broadcast(msg sse.Message) {
    m.messages = append(m.messages, msg)
}
```

## Future Architecture Considerations

### 📍 **Mapping Integration**
- Station coordinates already available
- Geographic clustering for efficient map rendering
- Potential integration with mapping libraries

### 📈 **Scaling Capabilities**
- Horizontal scaling via load balancing
- Database backend for state persistence
- Redis for distributed SSE client management

### 🔌 **Plugin Architecture**
- Interface-based design enables easy extension
- Additional data sources (other weather APIs)
- Alternative notification systems (WebSockets, polling)

## Deployment Architecture

### 🐳 **Container Deployment**
- Single binary with embedded assets
- Minimal Alpine Linux base image
- Health checks and graceful shutdown
- Configurable via environment variables

### 🏗️ **Infrastructure Requirements**
- **CPU**: <1% average usage
- **Memory**: ~30-50MB including Go runtime  
- **Network**: HTTPS outbound to FMI API
- **Storage**: Minimal (state files ~1-10MB)

The modular architecture ensures WindZ Monitor remains maintainable, performant, and ready for future enhancements while providing reliable wind monitoring for maritime and coastal applications.