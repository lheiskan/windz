package fmi

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// MultiStationParser handles parsing of multi-station FMI XML responses
type MultiStationParser struct {
	// Maps coordinate key to station ID
	coordToStation map[string]string

	// Station metadata indexed by station ID
	stations map[string]*StationMetadata

	// Wind parameter indices in the data tuples
	paramIndices map[WindParameter]int
}

// StationMetadata holds station information from XML
type StationMetadata struct {
	ID     string
	Name   string
	Region string
	Lat    float64
	Lon    float64
	WMO    string
	GeoID  string
}

// ParseMultiStationResponse parses XML response containing multiple stations
func ParseMultiStationResponse(reader io.Reader) ([]StationWindData, error) {
	parser := &MultiStationParser{
		coordToStation: make(map[string]string),
		stations:       make(map[string]*StationMetadata),
	}

	return parser.parse(reader)
}

// ParseMultiStationResponseWithGzip handles both gzipped and plain XML
func ParseMultiStationResponseWithGzip(reader io.Reader, isGzipped bool) ([]StationWindData, error) {
	var xmlReader io.Reader = reader

	if isGzipped {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzReader.Close()
		xmlReader = gzReader
	}

	return ParseMultiStationResponse(xmlReader)
}

func (p *MultiStationParser) parse(reader io.Reader) ([]StationWindData, error) {
	// Parse the XML structure
	var fc FeatureCollection
	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&fc); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	if len(fc.Members) == 0 {
		return nil, fmt.Errorf("no observation data in response")
	}

	// Process the first member (multi-station observations are in one member)
	member := fc.Members[0]
	obs := member.GridSeriesObservation

	// Extract station metadata from LocationCollection
	if err := p.extractStationMetadata(obs.SpatialSamplingFeature); err != nil {
		return nil, fmt.Errorf("failed to extract station metadata: %w", err)
	}

	// Extract wind parameters from the observed property URL
	p.extractParameterIndices(obs.ObservedProperty.Href)

	// Parse positions and data
	positions, err := p.parsePositions(obs.Result.MultiPointCoverage.DomainSet.SimpleMultiPoint.Positions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse positions: %w", err)
	}

	dataValues, err := p.parseDataValues(obs.Result.MultiPointCoverage.RangeSet.DataBlock.DoubleOrNilReasonTupleList)
	if err != nil {
		return nil, fmt.Errorf("failed to parse data values: %w", err)
	}

	// Validate data consistency
	if len(positions) != len(dataValues) {
		return nil, fmt.Errorf("position count (%d) doesn't match data count (%d)",
			len(positions), len(dataValues))
	}

	// Group observations by station
	stationObservations := make(map[string][]WindObservation)

	for i, pos := range positions {
		stationID := p.getStationIDForCoordinate(pos.Lat, pos.Lon)
		if stationID == "" {
			continue // Skip unknown coordinates
		}

		obs := p.createWindObservation(pos.Timestamp, dataValues[i])

		if stationObservations[stationID] == nil {
			stationObservations[stationID] = make([]WindObservation, 0)
		}
		stationObservations[stationID] = append(stationObservations[stationID], obs)
	}

	// Build final result
	var result []StationWindData
	for stationID, observations := range stationObservations {
		metadata := p.stations[stationID]
		if metadata == nil {
			continue
		}

		stationData := StationWindData{
			StationID:   stationID,
			StationName: metadata.Name,
			Location: Coordinates{
				Lat:    metadata.Lat,
				Lon:    metadata.Lon,
				Region: metadata.Region,
			},
			Observations: observations,
			Metadata: map[string]string{
				"wmo":   metadata.WMO,
				"geoid": metadata.GeoID,
			},
		}

		result = append(result, stationData)
	}

	return result, nil
}

func (p *MultiStationParser) extractStationMetadata(feature SpatialSamplingFeature) error {
	// Extract stations from LocationCollection
	for _, member := range feature.SampledFeature.Members {
		loc := member.Location

		// Extract station ID from identifier
		stationID := loc.Identifier.Value
		if stationID == "" {
			continue
		}

		metadata := &StationMetadata{
			ID:     stationID,
			Region: loc.Region,
		}

		// Extract various names
		for _, name := range loc.Names {
			switch name.CodeSpace {
			case "http://xml.fmi.fi/namespace/locationcode/name":
				metadata.Name = name.Value
			case "http://xml.fmi.fi/namespace/locationcode/wmo":
				metadata.WMO = name.Value
			case "http://xml.fmi.fi/namespace/locationcode/geoid":
				metadata.GeoID = name.Value
			}
		}

		p.stations[stationID] = metadata
	}

	// Extract coordinates from MultiPoint
	for _, pointMember := range feature.Shape.MultiPoint.PointMembers {
		coords, err := parseCoordinateString(pointMember.Point.Pos)
		if err != nil {
			continue
		}

		// Find matching station by name
		stationName := pointMember.Point.Name
		for id, metadata := range p.stations {
			if metadata.Name == stationName {
				metadata.Lat = coords.Lat
				metadata.Lon = coords.Lon

				// Create coordinate key for fast lookup
				coordKey := formatCoordinateKey(coords.Lat, coords.Lon)
				p.coordToStation[coordKey] = id
				break
			}
		}
	}

	return nil
}

func (p *MultiStationParser) extractParameterIndices(url string) {
	// Default parameter order from URL
	params := extractParametersFromURL(url)

	p.paramIndices = make(map[WindParameter]int)
	for i, param := range params {
		p.paramIndices[param] = i
	}

	// Default if not specified
	if len(p.paramIndices) == 0 {
		p.paramIndices[WindSpeedMS] = 0
		p.paramIndices[WindGustMS] = 1
		p.paramIndices[WindDirection] = 2
	}
}

func (p *MultiStationParser) parsePositions(positionsStr string) ([]PositionEntry, error) {
	var positions []PositionEntry

	// Split by whitespace and parse in groups of 3 (lat, lon, timestamp)
	parts := strings.Fields(positionsStr)
	if len(parts)%3 != 0 {
		return nil, fmt.Errorf("invalid positions format: expected triplets, got %d values", len(parts))
	}

	for i := 0; i < len(parts); i += 3 {
		lat, err := strconv.ParseFloat(parts[i], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude at position %d: %w", i, err)
		}

		lon, err := strconv.ParseFloat(parts[i+1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude at position %d: %w", i+1, err)
		}

		unixTime, err := strconv.ParseInt(parts[i+2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp at position %d: %w", i+2, err)
		}

		positions = append(positions, PositionEntry{
			Lat:       lat,
			Lon:       lon,
			Timestamp: time.Unix(unixTime, 0),
		})
	}

	return positions, nil
}

func (p *MultiStationParser) parseDataValues(dataStr string) ([][]float64, error) {
	var dataValues [][]float64

	// Each line contains values for one observation
	lines := strings.Split(strings.TrimSpace(dataStr), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		values := make([]float64, len(parts))

		for i, part := range parts {
			// Handle NaN values
			if part == "NaN" {
				values[i] = 0 // Will be converted to nil pointer later
				continue
			}

			val, err := strconv.ParseFloat(part, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid data value '%s': %w", part, err)
			}
			values[i] = val
		}

		dataValues = append(dataValues, values)
	}

	return dataValues, nil
}

func (p *MultiStationParser) getStationIDForCoordinate(lat, lon float64) string {
	coordKey := formatCoordinateKey(lat, lon)
	return p.coordToStation[coordKey]
}

func (p *MultiStationParser) createWindObservation(timestamp time.Time, values []float64) WindObservation {
	obs := WindObservation{
		Timestamp: timestamp,
	}

	// Map values to parameters based on indices
	if idx, ok := p.paramIndices[WindSpeedMS]; ok && idx < len(values) {
		if values[idx] > 0 { // Skip NaN/0 values
			val := values[idx]
			obs.WindSpeed = &val
		}
	}

	if idx, ok := p.paramIndices[WindGustMS]; ok && idx < len(values) {
		if values[idx] > 0 {
			val := values[idx]
			obs.WindGust = &val
		}
	}

	if idx, ok := p.paramIndices[WindDirection]; ok && idx < len(values) {
		if values[idx] >= 0 { // Direction can be 0
			val := values[idx]
			obs.WindDirection = &val
		}
	}

	return obs
}

// PositionEntry represents a position with timestamp
type PositionEntry struct {
	Lat       float64
	Lon       float64
	Timestamp time.Time
}

// Helper functions

func formatCoordinateKey(lat, lon float64) string {
	// Format with 5 decimal places for coordinate matching
	return fmt.Sprintf("%.5f,%.5f", lat, lon)
}

func parseCoordinateString(coordStr string) (Coordinates, error) {
	parts := strings.Fields(coordStr)
	if len(parts) < 2 {
		return Coordinates{}, fmt.Errorf("invalid coordinate string: %s", coordStr)
	}

	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("invalid latitude: %w", err)
	}

	lon, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return Coordinates{}, fmt.Errorf("invalid longitude: %w", err)
	}

	return Coordinates{Lat: lat, Lon: lon}, nil
}

func extractParametersFromURL(url string) []WindParameter {
	// Extract parameter list from URL query string
	if !strings.Contains(url, "param=") {
		return nil
	}

	parts := strings.Split(url, "param=")
	if len(parts) < 2 {
		return nil
	}

	paramStr := parts[1]
	if ampIdx := strings.Index(paramStr, "&"); ampIdx >= 0 {
		paramStr = paramStr[:ampIdx]
	}

	var params []WindParameter
	for _, p := range strings.Split(paramStr, ",") {
		switch strings.TrimSpace(p) {
		case "windspeedms":
			params = append(params, WindSpeedMS)
		case "windgust":
			params = append(params, WindGustMS)
		case "winddirection":
			params = append(params, WindDirection)
		}
	}

	return params
}
