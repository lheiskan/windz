# WindZ Monitor

A high-performance, modular wind monitoring application for Finnish weather stations with intelligent API optimization, battery-saving SSE reconnection, and real-time data streaming. Built with clean architecture and comprehensive performance optimization.

## Features

### 🚀 **Core Functionality**
- **16 Coastal & Maritime Stations**: Monitors key Finnish weather stations including Porkkala lighthouse area
- **Intelligent Batching**: Groups up to 20 stations per API call based on compatible time windows
- **Gzip Compression**: Automatic compression for reduced bandwidth usage
- **Adaptive Polling**: Automatically adjusts polling frequency (1m-24h) based on station activity
- **Real-time Updates**: SSE streaming with instant updates when new data arrives

### 📱 **Battery & Mobile Optimization**
- **Page Visibility API**: Automatically disconnects SSE when tab is hidden to save mobile battery
- **Smart Reconnection**: Full data refresh when page becomes visible again
- **Zero Data Loss**: Complete catch-up on reconnect ensures no missed observations
- **Connection Status**: Visual indicators for SSE connection state

### 🏗️ **Modern Architecture**
- **Modular Design**: Clean separation into SSE, Stations, and Observations modules
- **Thread-Safe**: Proper concurrency control with minimal lock contention
- **Interface-Based**: Loose coupling with dependency injection
- **Comprehensive Testing**: Full test coverage across all modules

### ⚡ **Performance & Production**
- **Single Binary**: Fully self-contained application with embedded HTML/CSS/JS
- **Low Resource Usage**: <50MB RAM, minimal CPU usage
- **Production Ready**: Health checks, metrics, graceful shutdown
- **Performance Metrics**: Comprehensive tracking of API efficiency and optimization

## Quick Start

```bash
# Build
go build -o windz

# Run with defaults (port 8080)
./windz

# Run with custom port
./windz -port 9000

# Enable debug logging
./windz -debug
```

Visit http://localhost:8080 to view the wind data dashboard.

## Monitored Stations

### Porkkala Area (Key Stations)
- **Emäsalo** (101023) - Porvoo area, closest to Porkkala
- **Kalbådagrund** (101022) - Porkkala lighthouse area
- **Itätoukki** (105392) - Sipoo area
- **Vuosaari** (151028) - Helsinki Vuosaari

### Maritime & Coastal
- **Harmaja** (100996) - Helsinki lighthouse
- **Bågaskär** (100969) - Inkoo coastal
- **Jussarö** (100965) - Raasepori maritime
- **Tulliniemi** (100946) - Hanko coastal
- **Russarö** (100932) - Hanko southern coast
- **Vänö** (100945) - Kemiönsaari archipelago
- **Utö** (100908) - Southern archipelago, HELCOM station

### Northern Coastal
- **Tahkoluoto** (101267) - Pori harbor
- **Tankar** (101661) - Kokkola west coast
- **Ulkokalla** (101673) - Kalajoki northern coast
- **Marjaniemi** (101784) - Hailuoto
- **Vihreäsaari** (101794) - Oulu harbor

## Intelligent API Optimization

The application features advanced FMI API optimization:

### Multi-Station Batching
- **Time Window Grouping**: Groups stations with compatible `LastObservation` times
- **Batch Size Limit**: Up to 20 stations per API call for optimal performance
- **Efficiency Gains**: Typically reduces API calls by 75-90% (16 individual → 1-4 batch calls)
- **Gzip Compression**: Automatic request/response compression

### Adaptive Polling Algorithm
1. **Starts Fast**: All stations begin with 1-minute polling
2. **Backs Off**: After 2 consecutive misses, moves to slower interval (1m→10m→60m→24h)
3. **Speeds Up**: Instantly adjusts when faster data is detected
4. **Saves Resources**: Combined with batching, reduces API load by 95%+

### Polling Intervals
- **1m** - Active stations with frequent updates
- **10m** - Standard weather stations
- **60m** - Stations with hourly updates
- **24h** - Inactive or offline stations

## Architecture

### 🏗️ **Modular Design**

WindZ is built with a clean, modular architecture that separates concerns and enables easy testing and maintenance:

```
windz/
├── main.go                 # Application entry point and coordination
├── internal/               # Internal modules
│   ├── sse/               # Server-Sent Events module
│   │   ├── interface.go   # SSE Manager interface
│   │   ├── manager.go     # Thread-safe client management
│   │   ├── handlers.go    # HTTP endpoint with battery-saving features
│   │   └── manager_test.go
│   ├── stations/          # Weather station metadata module
│   │   ├── interface.go   # Station Manager interface
│   │   ├── manager.go     # Station data and coordinate management
│   │   ├── handlers.go    # Station API endpoints
│   │   └── manager_test.go
│   └── observations/      # Weather observation polling module
│       ├── interface.go   # Observation Manager interface
│       ├── manager.go     # FMI API integration and adaptive polling
│       ├── handlers.go    # Observation API endpoints
│       └── manager_test.go
└── pkg/fmi/              # FMI API client library
```

### 🔄 **Battery-Saving SSE Flow**

1. **Page Load** → SSE connects → Callback sends all current observations
2. **Tab Hidden** → JavaScript disconnects SSE (saves battery)
3. **Tab Visible** → JavaScript reconnects → Fresh data automatically sent
4. **Network Issues** → Browser auto-reconnects → Complete data refresh

### 🧵 **Concurrency & Performance**

- **Minimal Lock Contention**: Polling holds locks only during brief map operations (microseconds)
- **Thread-Safe Design**: All modules use proper mutex protection for concurrent access
- **Non-Blocking Operations**: SSE callbacks run in goroutines to prevent blocking
- **Value Semantics**: Uses copies instead of shared pointers for data safety

## API Endpoints

### 🌐 **Web Interface**
- `/` - Main dashboard with real-time wind data table and battery-saving SSE
- `/events` - SSE stream with automatic initial data and reconnection support

### 📊 **JSON APIs**
- `/health` - Application health status with build information
- `/metrics` - Comprehensive polling and FMI API performance metrics
- `/api/stations` - Station metadata with coordinates and filtering
- `/api/stations/{id}` - Individual station lookup
- `/api/observations` - All latest wind observations
- `/api/observations/latest` - Latest observations as array
- `/api/observations/{id}` - Specific station observation

### Metrics Data
The `/metrics` endpoint provides detailed performance analytics:
- **Batching Efficiency**: Stations per request, largest batch sizes
- **Gzip Performance**: Compression rates, bandwidth savings
- **Response Times**: Average response times with exponential moving averages
- **Polling Stats**: Success rates, station activity distribution
- **Time Window Groups**: Real-time batching group counts

## Development

### 🧪 **Running Tests**

```bash
# Run all tests
go test ./...

# Run specific module tests
go test ./internal/sse/
go test ./internal/stations/
go test ./internal/observations/

# Run with coverage
go test -cover ./...

# Run with race detection
go test -race ./...
```

### 🔧 **Development Commands**

```bash
# Build for development
go build -o windz-dev .

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
air

# Format code
go fmt ./...

# Lint code (requires golangci-lint)
golangci-lint run
```

## Configuration

### Command Line Flags
```bash
-port int             HTTP server port (default 8080)
-state-file string    Polling state persistence file (default "polling_state.json")
-wind-data-file string Wind data cache persistence file (default "wind_data.json")
-debug               Enable debug logging with detailed SSE reconnection info
```

### Environment Variables
```bash
WINDZ_PORT=8080
WINDZ_STATE_FILE=/var/lib/windz/polling_state.json
WINDZ_WIND_DATA_FILE=/var/lib/windz/wind_data.json
WINDZ_DEBUG=true
```

### 🔍 **Debug Mode Features**
When running with `-debug` flag:
- Detailed FMI API batch processing logs  
- SSE client connection/disconnection tracking
- Polling state transitions and interval changes
- Performance metrics for time window grouping

## Building for Production

```bash
# Optimized binary
go build -ldflags="-s -w" -o windz

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o windz-linux
GOOS=darwin GOARCH=arm64 go build -o windz-darwin
GOOS=windows GOARCH=amd64 go build -o windz.exe
```

## Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o windz

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/windz /
EXPOSE 8080
CMD ["/windz"]
```

```bash
docker build -t windz .
docker run -p 8080:8080 windz
```

## Systemd Service

```ini
[Unit]
Description=WindZ Monitor
After=network.target

[Service]
Type=simple
User=windz
WorkingDirectory=/opt/windz
ExecStart=/opt/windz/windz
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

## Performance

- **Memory**: ~30-50MB including Go runtime
- **CPU**: <1% average usage
- **API Efficiency**: 95%+ reduction in API calls vs individual station polling
- **Batching Performance**: 5-16x improvement (75-90% fewer requests)
- **Gzip Compression**: ~60-80% bandwidth reduction
- **Startup Time**: <1 second
- **Binary Size**: ~10MB (with -ldflags="-s -w")

## Data Source

Wind data is fetched from the Finnish Meteorological Institute (FMI) Open Data API:
- Service: https://opendata.fmi.fi
- Update frequency: Varies by station (typically 10-60 minutes)
- Parameters: Wind speed, gust, and direction

## License

MIT License

## Acknowledgments

- Finnish Meteorological Institute (FMI) for providing open weather data
- All weather station operators maintaining these critical maritime observations