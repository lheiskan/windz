# WindZ Monitor - Development Context for Claude

## Project Overview
WindZ Monitor is a high-performance, modular Go web application that monitors wind conditions at 16 Finnish coastal weather stations, primarily used by surfers and windsurfers. The application features intelligent API optimization, battery-saving SSE reconnection, real-time data streaming, and comprehensive performance metrics.

**Live URL**: https://windz.nimble.fi  
**Target Audience**: Surf and windsurf community in Finland

## Current Architecture (2025) - MODULAR REFACTOR

### Core Application Stack
- **Backend**: Modular Go architecture with three functional modules
- **Frontend**: Server-side rendered HTML + battery-saving SSE updates  
- **Data Source**: Finnish Meteorological Institute (FMI) Open Data API
- **Deployment**: ARM64 Linux with automated SSH deployment
- **Favicon**: 🌊 Wave emoji for surf/windsurf theming

### 🏗️ **Modular Architecture (NEW)**
**Functional Organization** (not technical layers):
- **SSE Module** (`internal/sse/`) - Server-Sent Events with battery optimization
- **Stations Module** (`internal/stations/`) - Weather station metadata with coordinates  
- **Observations Module** (`internal/observations/`) - FMI API integration and adaptive polling

### 📱 **Battery & Mobile Optimization (NEW)**
1. **Page Visibility API**: Automatically disconnects SSE when tab hidden (saves mobile battery)
2. **Smart Reconnection**: Full data refresh when page becomes visible again
3. **Zero Data Loss**: Complete catch-up on reconnect ensures no missed observations
4. **Connection Status**: Visual indicators for SSE connection state

### Key Performance Features
1. **Multi-Station Batching**: Groups up to 20 stations per API call (75-90% API reduction)
2. **Adaptive Polling**: 1m→10m→60m→24h intervals based on station activity
3. **Persistent Cache**: `wind_data.json` for instant page loads (no empty HTML)
4. **Gzip Compression**: 100% usage rate for bandwidth optimization
5. **Build Metadata**: Git commit tracking in production metrics
6. **Thread-Safe Concurrency**: Microsecond lock holds vs seconds (fixed major bottleneck)

## File Structure - MODULAR

### Core Files
- `main.go` - Application coordination and module integration (~365 lines, down from 1800+)
- `Makefile` - ARM64 cross-compilation with metadata injection  
- `deploy.sh` - Automated SSH deployment with health checks
- `deploy.env.example` - Deployment configuration template
- `CLAUDE.md` - This context file
- `README.md` - Updated with modular architecture documentation
- `ARCHITECTURE.md` - Comprehensive technical documentation

### 🏗️ **Modular Structure (NEW)**
```
internal/
├── sse/                    # Server-Sent Events module
│   ├── interface.go       # SSE Manager interface
│   ├── manager.go         # Thread-safe client management
│   ├── handlers.go        # HTTP endpoints with battery-saving features
│   └── manager_test.go    # Comprehensive tests
├── stations/              # Weather station metadata module
│   ├── interface.go       # Station Manager interface
│   ├── manager.go         # Station data and coordinate management
│   ├── handlers.go        # Station API endpoints
│   └── manager_test.go    # Comprehensive tests
└── observations/          # Weather observation polling module
    ├── interface.go       # Observation Manager interface
    ├── manager.go         # FMI API integration and adaptive polling
    ├── handlers.go        # Observation API endpoints
    └── manager_test.go    # Comprehensive tests
```

### Package Structure  
- `pkg/fmi/observations/` - FMI API client with multi-station support
  - `query.go` - HTTP client with gzip and batching
  - `models.go` - Wind data structures
  - `parser.go` - XML parsing and transformation

### State Files (gitignored)
- `polling_state.json` - Station polling intervals and success rates
- `wind_data.json` - Cached wind measurements for instant loading
- `deploy.env` - Production server configuration

## API Optimization System

### Intelligent Batching Logic
Stations are grouped by compatible `LastObservation` times to maintain accuracy:
```go
timeWindowGroups := make(map[time.Time][]*StationPollingState)
// Groups stations that can share the same time window
```
**Result**: 16 individual calls → 1-4 batch calls per cycle

### Performance Metrics Available
- Batch efficiency (typically 5-16x improvement)
- Gzip usage rate (100%)  
- Response times (sub-100ms average)
- API call reduction (95%+ overall)

## Deployment System

### Build & Deploy
```bash
make linux-arm64  # ARM64 binary with git metadata
./deploy.sh       # Automated deployment to windz.nimble.fi
```

### Deployment Features
- Passwordless sudo automation
- Binary backup with timestamps
- Service health verification  
- Build metadata tracking
- Graceful service restart

### Production Infrastructure
- **Server**: windz.nimble.fi (ARM64 Ubuntu)
- **Path**: `/opt/windz/windz`
- **Service**: systemd with auto-restart
- **Port**: 8081 (behind HTTPS proxy)

## Development Guidelines

### MANDATORY Go File Workflow - NEVER SKIP
**CRITICAL**: Always format Go files BEFORE staging them for git operations:

```bash
# 1. Make changes to .go files
# 2. Format BEFORE staging (MANDATORY - pre-commit hook WILL fail otherwise)
gofmt -w main.go
gofmt -w internal/**/*.go
gofmt -w pkg/**/*.go
# 3. THEN stage and commit  
git add main.go internal/ pkg/
git commit -m "..."
```

**⚠️ REMINDER**: The project has a pre-commit hook that enforces gofmt formatting. Never `git add` Go files without running `gofmt -w` first!

### Testing Commands (UPDATED)
```bash
# Run all tests (includes all modules)
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

### Code Standards
- Use conventional commit format: `feat:`, `fix:`, `docs:`  
- Run `gofmt -w` before commits (pre-commit hook enforces)
- Include `@vibe: 🤖 -- ai` comment in AI-generated files
- Add performance metrics to commit messages when relevant

### Testing Approach
- Run `go test ./...` before commits
- Use `RUN_INTEGRATION_TESTS=true` for integration tests  
- Manual testing with curl commands for API verification
- Production monitoring via `/metrics` endpoint

## Key API Endpoints

- `/` - Dashboard with server-side rendered wind data
- `/events` - SSE stream for real-time updates  
- `/health` - Application health status
- `/metrics` - Performance analytics and build info
- `/api/stations` - JSON API with station data

## Station Focus Areas

### Porkkala Area (Primary)
- **Kalbådagrund** (101022) - Porkkala lighthouse ⭐ 
- **Emäsalo** (101023) - Porvoo area, closest to Porkkala
- **Itätoukki** (105392) - Sipoo area
- **Vuosaari** (151028) - Helsinki area

### Maritime Stations
- Harmaja (100996), Russarö (100932), Utö (100908)
- Total: 16 coastal/maritime stations across Finland

## Recent Major Features (2025) - MODULAR REFACTOR

### 🏗️ **Major Architectural Refactor**
1. **Modular Architecture** - Transformed monolithic 1800+ line main.go into clean functional modules
2. **Battery-Saving SSE** - Page Visibility API integration for mobile battery optimization
3. **Concurrency Fixes** - Reduced mutex lock hold time from seconds to microseconds
4. **Interface-Based Design** - Dependency injection and loose coupling between modules
5. **Comprehensive Testing** - Full test coverage across all modules

### 🚀 **Performance & API Features**
1. **Multi-Station Batching** - Intelligent API call grouping (up to 20 stations per call)
2. **Performance Metrics** - Comprehensive monitoring dashboard  
3. **Deployment Automation** - One-command production deployment
4. **Wind Data Persistence** - Eliminates empty page loads with cached state
5. **Build Metadata** - Production version tracking with git commit info
6. **Server-Side Rendering** - Instant data visibility on page load
7. **Thread-Safe Operations** - All shared state protected with proper mutex usage

## Development Workflow

### Common Commands
```bash
# Local development  
go run . -port 8080 -debug

# Test build
make windz && ./windz

# Run tests (all modules)
go test ./...

# Deploy to production
make linux-arm64 && ./deploy.sh

# Check production status
curl https://windz.nimble.fi/metrics | jq '.build'

# Format Go files (REQUIRED before commits)
gofmt -w main.go internal/**/*.go pkg/**/*.go
```

### Troubleshooting
- **Empty pages**: Check `wind_data.json` validity
- **Deploy fails**: Verify `deploy.env` and SSH keys
- **API errors**: Monitor FMI service status
- **Build issues**: Check git metadata availability

## Performance Benchmarks

### Current Production Stats
- **API Efficiency**: 95%+ call reduction vs individual polling
- **Batching**: 5-16x improvement (typical 16→1-4 calls)
- **Response Time**: ~80ms average with exponential moving average
- **Gzip Usage**: 100% compression rate
- **Memory**: 30-50MB including Go runtime
- **Binary Size**: ~8MB optimized build

## Future Enhancement Areas
- Mobile-responsive design improvements
- Historical data visualization
- Wind forecast integration  
- User preferences for favorite stations
- Prometheus metrics export
- Database integration for analytics
- **Map Integration**: Station coordinates are already available for mapping features
- **Plugin Architecture**: Interface-based design enables easy extension
- **Horizontal Scaling**: Load balancing with Redis for distributed SSE clients

---

*This context is current as of September 2025. Major modular architecture refactor completed.*

