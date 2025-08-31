# FMI (Finnish Meteorological Institute) API Client

This package provides a Go client for accessing the Finnish Meteorological Institute (FMI) Open Data API. It's organized by functionality to make it easy to work with different types of weather data.

## Package Structure

The package is organized by functionality rather than technical layers:

```
pkg/fmi/
├── README.md                    # This documentation
├── client.go                   # Main client (legacy, will be refactored)
├── models.go                   # Shared data models and types
├── xml_types.go               # Common XML parsing structures
├── fetch_data.sh              # Script for fetching test data from FMI API
├── testdata/                  # Shared test data
│   └── test_three_station_response.xml
│
├── observations/              # Weather observations functionality
│   ├── models.go             # Observation-specific data models
│   ├── xml_types.go          # XML structures for observation responses
│   ├── parser.go             # Multi-station XML parser
│   ├── query.go              # Query building and execution
│   ├── parser_test.go        # Parser unit tests
│   ├── query_test.go         # Query functionality tests
│   └── testdata/
│       └── test_three_station_response.xml
│
└── stations/                  # Station metadata functionality (future)
    ├── models.go
    ├── parser.go
    ├── query.go
    └── testdata/
```

## Functional Areas

### 1. Observations (`pkg/fmi/observations`)

Handles weather observation data from FMI's `fmi::observations::weather::multipointcoverage` stored query.

**Features:**
- Multi-station data parsing in single request
- Coordinate matching to correlate data with stations
- Support for wind parameters: speed, gust, direction
- Gzip compression support
- Comprehensive error handling

**Usage:**
```go
import "windz/pkg/fmi/observations"

// Create query handler
query := observations.NewQuery("https://opendata.fmi.fi/wfs", httpClient)

// Execute query
req := observations.Request{
    StartTime:  time.Now().Add(-1 * time.Hour),
    EndTime:    time.Now(),
    StationIDs: []string{"100996", "101023", "151028"},
    Parameters: []observations.WindParameter{
        observations.WindSpeedMS,
        observations.WindGustMS, 
        observations.WindDirection,
    },
    UseGzip: true,
}

response, err := query.Execute(req)
if err != nil {
    log.Fatal(err)
}

for _, station := range response.Stations {
    fmt.Printf("Station %s: %d observations\n", 
        station.StationName, len(station.Observations))
}
```

### 2. Stations (Future)

Will handle station metadata from FMI's `fmi::ef::stations` stored query.

## FMI API Stored Queries

The FMI Open Data API uses "stored queries" to access different types of data:

### Currently Supported

| Stored Query ID | Purpose | Package | Status |
|-----------------|---------|---------|---------|
| `fmi::observations::weather::multipointcoverage` | Weather observations | `observations/` | ✅ Implemented |

### Planned

| Stored Query ID | Purpose | Package | Status |
|-----------------|---------|---------|---------|
| `fmi::ef::stations` | Station metadata | `stations/` | 🚧 Planned |
| `fmi::observations::lightning::simple` | Lightning data | `lightning/` | 🚧 Future |
| `fmi::radar::composite::rr` | Radar precipitation | `radar/` | 🚧 Future |

## Data Fetching Script

The `fetch_data.sh` script can be used to fetch test data from the FMI API:

```bash
# Fetch data for 3 default stations (last 1 hour)
./fetch_data.sh

# Fetch data for specific station
./fetch_data.sh 100996 2 harmaja.xml

# Fetch data for custom time period
./fetch_data.sh 24 daily_data.xml
```

**Script Features:**
- Supports single or multi-station queries
- Configurable time ranges
- Automatic error detection and validation
- Outputs station summaries and observation counts

**Default Test Stations:**
- 101023 - Emäsalo (Porvoo) - Porkkala area
- 100996 - Harmaja (Helsinki Maritime) - Key lighthouse station  
- 151028 - Vuosaari (Helsinki) - Harbor station

## Testing

Each functional package includes comprehensive tests:

```bash
# Test observations package
go test -v ./pkg/fmi/observations

# Run with benchmarks
go test -bench=. ./pkg/fmi/observations

# Integration tests (requires real API access)
RUN_INTEGRATION_TESTS=true go test -v ./pkg/fmi/observations
```

**Test Coverage:**
- XML parsing with real FMI data
- Coordinate matching algorithms  
- HTTP client functionality
- Error handling scenarios
- Performance benchmarks

## Performance

Current performance characteristics (Apple M2):

| Operation | Time | Notes |
|-----------|------|-------|
| Parse 3 stations, 174 observations | ~366μs | Multi-station XML parsing |
| Build query URL | ~2μs | URL construction with parameters |
| HTTP request + parse | ~200-500ms | Depends on FMI API response time |

## Station IDs

Common Finnish weather stations monitored by this application:

### Porkkala Area (Key Maritime Stations)
- **101023** - Emäsalo (Porvoo area, closest to Porkkala)
- **101022** - Kalbådagrund (Porkkala lighthouse area)
- **100996** - Harmaja (Helsinki lighthouse, maritime reference)
- **151028** - Vuosaari (Helsinki harbor)
- **105392** - Itätoukki (Sipoo area)

### Coastal & Maritime Stations
- **100969** - Bågaskär (Inkoo coastal)
- **100965** - Jussarö (Raasepori maritime)
- **100946** - Tulliniemi (Hanko coastal)
- **100932** - Russarö (Hanko southern coast)
- **100945** - Vänö (Kemiönsaari archipelago)
- **100908** - Utö (Southern archipelago, HELCOM station)

### Northern Coastal
- **101267** - Tahkoluoto (Pori harbor)
- **101661** - Tankar (Kokkola west coast)
- **101673** - Ulkokalla (Kalajoki northern coast)
- **101784** - Marjaniemi (Hailuoto)
- **101794** - Vihreäsaari (Oulu harbor)

## API Reference

### Base URL
```
https://opendata.fmi.fi/wfs
```

### Common Parameters
- `service=WFS` - Web Feature Service
- `version=2.0.0` - WFS version
- `request=getFeature` - Request type
- `storedquery_id` - Specific query identifier
- `starttime` - Start time (ISO 8601 format)
- `endtime` - End time (ISO 8601 format)
- `fmisid` - Station ID(s), can be repeated for multiple stations
- `parameters` - Comma-separated parameter list
- `bbox` - Bounding box for geographic queries

### Response Format
FMI returns XML in WFS (Web Feature Service) format with complex nested structures. The parsers in this package handle the complexity and provide clean Go structs.

## Error Handling

The package handles several types of errors:

1. **HTTP Errors** - Network issues, timeouts
2. **FMI API Errors** - Invalid parameters, no data available
3. **XML Parsing Errors** - Malformed or unexpected response format
4. **Data Validation Errors** - Invalid coordinates, missing required fields

## Contributing

When adding new functionality:

1. **Create a new functional package** (e.g., `lightning/`, `radar/`)
2. **Include models, parser, query, and tests** in the same directory
3. **Add test data** in the package's `testdata/` directory
4. **Update this README** with the new functionality
5. **Follow the existing patterns** for consistency

## Dependencies

- **Standard library only** - No external dependencies for core functionality
- **net/http** - HTTP client operations
- **encoding/xml** - XML parsing
- **compress/gzip** - Optional response compression

## License

This package is part of the WindZ Monitor application and follows the same license terms.