// @vibe: 🤖 -- ai
package stations

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"time"
)

// Parser handles parsing of FMI station XML responses
type Parser struct{}

// NewParser creates a new stations parser
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a stations XML response
func (p *Parser) Parse(reader io.Reader) (*Response, error) {
	return p.parseXML(reader)
}

// ParseXML parses the XML content directly
func (p *Parser) ParseXML(reader io.Reader) (*Response, error) {
	return p.parseXML(reader)
}

func (p *Parser) parseXML(reader io.Reader) (*Response, error) {
	var wfsResponse WFSStationResponse
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&wfsResponse); err != nil {
		return nil, err
	}

	// Convert to our Station model
	stations := make([]Station, 0, len(wfsResponse.Members))
	for _, member := range wfsResponse.Members {
		if station := convertToStationModel(member); station != nil {
			stations = append(stations, *station)
		}
	}

	return &Response{
		Stations: stations,
		Count:    len(stations),
	}, nil
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
