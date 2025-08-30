# WindZ Monitor

A lightweight, production-ready wind monitoring application for Finnish weather stations. Features real-time data streaming via Server-Sent Events (SSE) and an adaptive polling algorithm that optimizes API usage.

## Features

- **16 Coastal & Maritime Stations**: Monitors key Finnish weather stations including Porkkala lighthouse area
- **Adaptive Polling**: Automatically adjusts polling frequency (1m-24h) based on station activity
- **Real-time Updates**: SSE streaming with instant updates when new data arrives
- **Single Binary**: Fully self-contained application with embedded HTML/CSS/JS
- **Low Resource Usage**: <50MB RAM, minimal CPU usage
- **Production Ready**: Health checks, metrics, graceful shutdown

## Quick Start

```bash
# Build
go build -o windz-monitor

# Run with defaults (port 8080)
./windz-monitor

# Run with custom port
./windz-monitor -port 9000

# Enable debug logging
./windz-monitor -debug
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

## Adaptive Polling Algorithm

The application uses an intelligent polling system that:

1. **Starts Fast**: All stations begin with 1-minute polling
2. **Backs Off**: After 2 consecutive misses, moves to slower interval (1m→10m→60m→24h)
3. **Speeds Up**: Instantly adjusts when faster data is detected
4. **Saves Resources**: Reduces API calls by ~90% for inactive stations

### Polling Intervals
- **1m** - Active stations with frequent updates
- **10m** - Standard weather stations
- **60m** - Stations with hourly updates
- **24h** - Inactive or offline stations

## API Endpoints

- `/` - Main dashboard with real-time wind data table
- `/events` - SSE stream for real-time updates
- `/health` - Application health status
- `/metrics` - Polling and performance metrics
- `/api/stations` - JSON API with station status and latest data

## Configuration

### Command Line Flags
```bash
-port int          HTTP server port (default 8080)
-state-file string Polling state persistence file (default "polling_state.json")
-debug            Enable debug logging
```

### Environment Variables
```bash
WINDZ_PORT=8080
WINDZ_STATE_FILE=/var/lib/windz/state.json
```

## Building for Production

```bash
# Optimized binary
go build -ldflags="-s -w" -o windz-monitor

# Cross-compilation
GOOS=linux GOARCH=amd64 go build -o windz-monitor-linux
GOOS=darwin GOARCH=arm64 go build -o windz-monitor-darwin
GOOS=windows GOARCH=amd64 go build -o windz-monitor.exe
```

## Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags="-s -w" -o windz-monitor

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/windz-monitor /
EXPOSE 8080
CMD ["/windz-monitor"]
```

```bash
docker build -t windz-monitor .
docker run -p 8080:8080 windz-monitor
```

## Systemd Service

```ini
[Unit]
Description=WindZ Monitor
After=network.target

[Service]
Type=simple
User=windz
WorkingDirectory=/opt/windz-monitor
ExecStart=/opt/windz-monitor/windz-monitor
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
- **API Efficiency**: ~90% reduction in API calls vs fixed polling
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