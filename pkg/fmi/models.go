package fmi

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// StationWindData represents all wind observations for a single station
type StationWindData struct {
	StationID    string            `json:"station_id"`
	StationName  string            `json:"station_name"`
	Location     Coordinates       `json:"coordinates"`
	Observations []WindObservation `json:"observations"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// WindObservation represents a single timestamped wind measurement
type WindObservation struct {
	Timestamp     time.Time `json:"timestamp"`
	WindSpeed     *float64  `json:"wind_speed_ms,omitempty"`
	WindGust      *float64  `json:"wind_gust_ms,omitempty"`
	WindDirection *float64  `json:"wind_direction_deg,omitempty"`
	Quality       string    `json:"quality,omitempty"`
}

// WindReading represents a single wind observation with station context
type WindReading struct {
	StationID     string      `json:"station_id"`
	StationName   string      `json:"station_name"`
	Network       string      `json:"network,omitempty"`
	Timestamp     time.Time   `json:"timestamp"`
	Location      Coordinates `json:"coordinates"`
	WindSpeed     *float64    `json:"wind_speed_ms,omitempty"`
	WindGust      *float64    `json:"wind_gust_ms,omitempty"`
	WindDirection *float64    `json:"wind_direction_deg,omitempty"`
	Quality       string      `json:"quality,omitempty"`
}

// Coordinates represents geographic location
type Coordinates struct {
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Region string  `json:"region,omitempty"`
}


// ProcessingStats provides summary of streaming operation
type ProcessingStats struct {
	TotalObservations     int           `json:"total_observations"`
	ProcessedObservations int           `json:"processed_observations"`
	SkippedObservations   int           `json:"skipped_observations"`
	StationCount          int           `json:"station_count"`
	ErrorCount            int           `json:"error_count"`
	Duration              time.Duration `json:"duration"`
	BytesProcessed        int64         `json:"bytes_processed"`
}

// WindParameter represents wind measurement parameters
type WindParameter string

const (
	WindSpeedMS   WindParameter = "windspeedms"
	WindGustMS    WindParameter = "windgust"
	WindDirection WindParameter = "winddirection"
)

// Legacy types for backward compatibility with existing client.go
// TODO: Remove when client.go is updated to use observations and stations packages

// BBox represents a geographic bounding box
type BBox struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// String returns the bounding box as a comma-separated string for API queries
func (b BBox) String() string {
	return fmt.Sprintf("%.2f,%.2f,%.2f,%.2f", b.MinLon, b.MinLat, b.MaxLon, b.MaxLat)
}

// Predefined bounding boxes for convenience
var (
	FinlandBBox         = BBox{19.08, 59.45, 31.59, 70.09} // All Finland
	SouthernFinlandBBox = BBox{19.5, 59.7, 31.6, 61.8}     // Southern Finland
	CentralFinlandBBox  = BBox{22.0, 61.8, 31.0, 65.0}     // Central Finland
	NorthernFinlandBBox = BBox{20.0, 65.0, 31.6, 70.1}     // Northern Finland
)

// Station represents a weather station (legacy compatibility)
type Station struct {
	ID           string            `json:"id"`
	FMISID       string            `json:"fmisid"`
	Name         string            `json:"name"`
	Location     Coordinates       `json:"coordinates"`
	StartDate    time.Time         `json:"start_date"`
	EndDate      *time.Time        `json:"end_date,omitempty"`
	Network      string            `json:"network"`
	Capabilities []string          `json:"capabilities"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// WindDataCallbacks defines callbacks for streaming wind data processing (legacy)
type WindDataCallbacks struct {
	// OnStationData - called when all data for a station is complete
	OnStationData func(stationData StationWindData) error

	// OnError called when parsing or processing errors occur
	OnError func(err error)

	// OnProgress called periodically during processing
	OnProgress func(processed, total int)

	// OnComplete called when streaming is complete with final stats
	OnComplete func(stats ProcessingStats)
}


// Legacy XML parsing types for stations (TODO: Remove when client.go updated)

// WFSStationResponse represents the root WFS response for station queries
type WFSStationResponse struct {
	XMLName xml.Name           `xml:"FeatureCollection"`
	Members []WFSStationMember `xml:"member"`
}

// WFSStationMember represents a single station in the WFS response
type WFSStationMember struct {
	XMLName            xml.Name           `xml:"member"`
	MonitoringFacility MonitoringFacility `xml:"EnvironmentalMonitoringFacility"`
}

// MonitoringFacility represents the environmental monitoring facility (station)
type MonitoringFacility struct {
	XMLName    xml.Name      `xml:"EnvironmentalMonitoringFacility"`
	ID         string        `xml:"id,attr"`
	Identifier GMLIdentifier `xml:"identifier"`
	Names      []GMLName     `xml:"name"`
	StartDate  string        `xml:"operationalActivityPeriod>OperationalActivityPeriod>activityTime>TimePeriod>beginPosition"`
	Geometry   WFSGeometry   `xml:"representativePoint"`
	BelongsTo  []BelongsTo   `xml:"belongsTo"`
}

// GMLIdentifier represents an identifier element with codeSpace attribute
type GMLIdentifier struct {
	XMLName   xml.Name `xml:"identifier"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// GMLName represents a name element with codeSpace attribute
type GMLName struct {
	XMLName   xml.Name `xml:"name"`
	CodeSpace string   `xml:"codeSpace,attr"`
	Value     string   `xml:",chardata"`
}

// WFSGeometry represents the geographic location of the station
type WFSGeometry struct {
	XMLName xml.Name `xml:"representativePoint"`
	Point   WFSPoint `xml:"Point"`
}

// WFSPoint represents a geographic point in WFS
type WFSPoint struct {
	XMLName     xml.Name `xml:"Point"`
	ID          string   `xml:"id,attr"`
	SrsName     string   `xml:"srsName,attr"`
	Coordinates string   `xml:"pos"`
}

// BelongsTo represents the network(s) the station belongs to
type BelongsTo struct {
	XMLName xml.Name `xml:"belongsTo"`
	Title   string   `xml:"title,attr"`
	Href    string   `xml:"href,attr"`
}

// Legacy parsing functions (TODO: Remove when client.go updated)

// parseStationXML parses the XML response from FMI stations query
func parseStationXML(reader io.Reader, response *WFSStationResponse) error {
	decoder := xml.NewDecoder(reader)
	return decoder.Decode(response)
}

// convertToStationModel converts FMI XML data to our Station model
func convertToStationModel(member WFSStationMember) *Station {
	if member.MonitoringFacility.ID == "" {
		return nil
	}

	// Parse coordinates
	coords := parseCoordinates(member.MonitoringFacility.Geometry.Point.Coordinates)
	if len(coords) < 2 || !isValidCoordinate(coords[0], coords[1]) {
		return nil
	}

	// Parse start date
	startDate, _ := time.Parse(time.RFC3339, member.MonitoringFacility.StartDate)

	// Extract station name and FMIS ID
	stationName := extractStationName(member.MonitoringFacility.Names)
	fmisID := extractFMISID(member.MonitoringFacility.Identifier)

	return &Station{
		ID:     member.MonitoringFacility.ID,
		FMISID: fmisID,
		Name:   stationName,
		Location: Coordinates{
			Lat: coords[0], // Latitude is first in coordinate pair (FMI uses "Lat Long" order)
			Lon: coords[1], // Longitude is second in coordinate pair
		},
		StartDate:    startDate,
		Network:      extractNetwork(member.MonitoringFacility.BelongsTo),
		Capabilities: GetDefaultWindCapabilities(),
		Metadata:     make(map[string]string),
	}
}

// parseCoordinates parses coordinate string from FMI XML
func parseCoordinates(coordStr string) []float64 {
	if coordStr == "" {
		return nil
	}

	parts := strings.Fields(strings.TrimSpace(coordStr))
	if len(parts) < 2 {
		return nil
	}

	var coords []float64
	for _, part := range parts {
		if val, err := strconv.ParseFloat(part, 64); err == nil {
			coords = append(coords, val)
		}
	}

	return coords
}

// isValidCoordinate checks if latitude and longitude values are reasonable
func isValidCoordinate(lat, lon float64) bool {
	// Finland's approximate coordinate bounds
	return lat >= 59.0 && lat <= 71.0 && lon >= 19.0 && lon <= 32.0
}

// extractStationName extracts the human-readable station name from GML names
func extractStationName(names []GMLName) string {
	// Look for Finnish name first, then fallback to any available name
	for _, name := range names {
		if name.CodeSpace == "http://xml.fmi.fi/namespace/locationcode/name" ||
			strings.Contains(strings.ToLower(name.CodeSpace), "name") {
			return strings.TrimSpace(name.Value)
		}
	}

	// If no specific name found, use the first available
	if len(names) > 0 {
		return strings.TrimSpace(names[0].Value)
	}

	return ""
}

// extractFMISID extracts the FMIS ID from the identifier
func extractFMISID(identifier GMLIdentifier) string {
	// FMIS ID is usually in the identifier field
	if identifier.CodeSpace == "http://xml.fmi.fi/namespace/stationcode/fmisid" {
		return strings.TrimSpace(identifier.Value)
	}

	// Fallback: try to extract from the value if it looks like a numeric ID
	value := strings.TrimSpace(identifier.Value)
	if value != "" && isNumeric(value) {
		return value
	}

	return ""
}

// extractNetwork extracts the network name from BelongsTo elements
func extractNetwork(belongsTo []BelongsTo) string {
	// Look for network information in the belongsTo elements
	for _, bt := range belongsTo {
		if bt.Title != "" {
			return strings.TrimSpace(bt.Title)
		}
	}

	return "Unknown"
}

// isNumeric checks if a string contains only numeric characters
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// GetDefaultWindCapabilities returns a default set of wind measurement capabilities
// This is used when we can't determine actual capabilities from API responses
func GetDefaultWindCapabilities() []string {
	return []string{
		"WS_PT1H_AVG", // Wind speed (hourly average)
		"WD_PT1H_AVG", // Wind direction (hourly average)
		"WG_PT1H_MAX", // Wind gust (hourly maximum)
	}
}

// Legacy parser stubs (TODO: Remove when client.go is updated)

// DeprecatedParser is a stub for deprecated parsing functionality
type DeprecatedParser struct {
	callbacks WindDataCallbacks
}

// Parse is a deprecated method - use observations package instead
func (p *DeprecatedParser) Parse() error {
	if p.callbacks.OnError != nil {
		p.callbacks.OnError(fmt.Errorf("DeprecatedParser.Parse is deprecated - use pkg/fmi/observations package instead"))
	}
	return fmt.Errorf("DeprecatedParser.Parse is deprecated - use pkg/fmi/observations package instead")
}

// NewStationGroupingParser creates a deprecated parser - use observations package instead
func NewStationGroupingParser(reader io.Reader, callbacks WindDataCallbacks) *DeprecatedParser {
	return &DeprecatedParser{callbacks: callbacks}
}

// ParseMultiStationResponseWithGzip is deprecated - use observations package instead
func ParseMultiStationResponseWithGzip(reader io.Reader, isGzipped bool) ([]StationWindData, error) {
	return nil, fmt.Errorf("ParseMultiStationResponseWithGzip is deprecated - use pkg/fmi/observations package instead")
}

