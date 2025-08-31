package fmi

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

// StationGroupingParser collects all data for each station before processing
type StationGroupingParser struct {
	decoder     *xml.Decoder
	callbacks   WindDataCallbacks
	stats       ProcessingStats
	stationData map[string]*StationWindData // stationID -> accumulated data
	startTime   time.Time
}

// WindDataCallbacks defines callbacks for streaming wind data processing
type WindDataCallbacks struct {
	// OnStationData - called when all data for a station is complete
	OnStationData func(stationData StationWindData) error

	// OnError called when parsing or processing errors occur
	OnError func(err error)

	// OnStart called when streaming begins (optional)
	OnStart func()

	// OnComplete called when streaming finishes (optional)
	OnComplete func(stats ProcessingStats)
}

// NewStationGroupingParser creates a parser that groups data by station
func NewStationGroupingParser(r io.Reader, callbacks WindDataCallbacks) *StationGroupingParser {
	return &StationGroupingParser{
		decoder:     xml.NewDecoder(r),
		callbacks:   callbacks,
		stationData: make(map[string]*StationWindData),
		startTime:   time.Now(),
	}
}

// Parse processes the XML stream and groups data by station
func (p *StationGroupingParser) Parse() error {
	// Notify start
	if p.callbacks.OnStart != nil {
		p.callbacks.OnStart()
	}

	// Parse the entire feature collection
	var fc FeatureCollection
	if err := p.decoder.Decode(&fc); err != nil {
		p.handleError(fmt.Errorf("failed to parse XML: %w", err))
		return err
	}

	// Process each member (station-parameter combination)
	for _, member := range fc.Members {
		if err := p.processFeatureMember(member); err != nil {
			p.handleError(fmt.Errorf("failed to process member: %w", err))
			continue
		}
	}

	// Finalize and notify callbacks for each station
	p.finalizeAndNotify()

	return nil
}

// processFeatureMember processes a single feature member (station-parameter observation set)
func (p *StationGroupingParser) processFeatureMember(member FeatureMember) error {
	obs := member.GridSeriesObservation

	// Handle single station case (older format compatibility)
	// For single station responses, extract from the first member
	if len(obs.SpatialSamplingFeature.SampledFeature.Members) == 0 {
		return fmt.Errorf("no station data found in observation")
	}

	// Get first location (for single station queries)
	firstLocation := obs.SpatialSamplingFeature.SampledFeature.Members[0].Location

	// Extract station information
	stationID := firstLocation.Identifier.Value
	if stationID == "" {
		return fmt.Errorf("no station ID found in observation")
	}

	stationName := ""
	for _, name := range firstLocation.Names {
		if name.CodeSpace == "http://xml.fmi.fi/namespace/locationcode/name" {
			stationName = name.Value
			break
		}
	}

	// Extract coordinates
	coords, err := p.extractCoordinates(obs.SpatialSamplingFeature.Shape.MultiPoint)
	if err != nil {
		return fmt.Errorf("failed to extract coordinates: %w", err)
	}

	// Get or create station data container
	stationData, exists := p.stationData[stationID]
	if !exists {
		stationData = &StationWindData{
			StationID:    stationID,
			StationName:  stationName,
			Location:     coords,
			Observations: make([]WindObservation, 0),
			Metadata:     make(map[string]string),
		}
		stationData.Location.Region = firstLocation.Region
		p.stationData[stationID] = stationData
		p.stats.StationCount++
	}

	// Parse the observation data
	observations, err := p.parseObservations(obs)
	if err != nil {
		return fmt.Errorf("failed to parse observations for station %s: %w", stationID, err)
	}

	// Check if this is a multi-parameter query
	parameters := p.extractParametersFromURL(obs.ObservedProperty.Href)

	if len(parameters) > 1 {
		// Multi-parameter data: observations already contain all parameters, merge directly
		p.mergeMultiParameterObservations(stationData, observations)
	} else {
		// Single-parameter data: use the original merging logic
		parameter := WindSpeedMS // Default
		if len(parameters) > 0 {
			parameter = parameters[0]
		}
		p.mergeObservations(stationData, observations, parameter)
	}

	p.stats.TotalObservations += len(observations)

	return nil
}

// extractCoordinates extracts coordinates from multipoint
func (p *StationGroupingParser) extractCoordinates(multiPoint MultiPoint) (Coordinates, error) {
	if len(multiPoint.PointMembers) == 0 {
		return Coordinates{}, fmt.Errorf("no point members found")
	}

	// Get first point's position
	posStr := multiPoint.PointMembers[0].Point.Pos
	parts := strings.Fields(posStr)
	if len(parts) < 2 {
		return Coordinates{}, fmt.Errorf("invalid position format: %s", posStr)
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

// extractParameter extracts the parameter type from the observed property URL
func (p *StationGroupingParser) extractParameter(href string) WindParameter {
	if strings.Contains(href, "windspeedms") || strings.Contains(href, "ws_10min") {
		return WindSpeedMS
	} else if strings.Contains(href, "windgust") || strings.Contains(href, "wg_10min") {
		return WindGustMS
	} else if strings.Contains(href, "winddirection") || strings.Contains(href, "wd_10min") {
		return WindDirection
	}
	return WindParameter("")
}

// extractParametersFromURL extracts all parameters from multi-parameter URLs
func (p *StationGroupingParser) extractParametersFromURL(href string) []WindParameter {
	var params []WindParameter

	// Handle multi-parameter URLs like "param=windspeedms,windgust,winddirection"
	if strings.Contains(href, "param=") {
		// Extract the param value
		parts := strings.Split(href, "param=")
		if len(parts) > 1 {
			paramPart := parts[1]
			// Get everything up to the next & or end of string
			if ampIndex := strings.Index(paramPart, "&"); ampIndex != -1 {
				paramPart = paramPart[:ampIndex]
			}

			// Split by comma for multi-parameter queries
			paramNames := strings.Split(paramPart, ",")
			for _, paramName := range paramNames {
				paramName = strings.TrimSpace(paramName)
				if param := p.extractParameterByName(paramName); param != "" {
					params = append(params, param)
				}
			}
		}
	}

	// Fallback to single parameter extraction if no multi-parameter found
	if len(params) == 0 {
		if param := p.extractParameter(href); param != "" {
			params = append(params, param)
		}
	}

	return params
}

// extractParameterByName converts parameter names to WindParameter types
func (p *StationGroupingParser) extractParameterByName(paramName string) WindParameter {
	switch paramName {
	case "windspeedms", "ws_10min":
		return WindSpeedMS
	case "windgust", "wg_10min":
		return WindGustMS
	case "winddirection", "wd_10min":
		return WindDirection
	default:
		return WindParameter("")
	}
}

// extractParametersFromFields extracts parameter mapping from rangeType fields
func (p *StationGroupingParser) extractParametersFromFields(fields []Field) []WindParameter {
	var params []WindParameter

	for _, field := range fields {
		// Check field name first
		if param := p.extractParameterByName(field.Name); param != "" {
			params = append(params, param)
		} else if param := p.extractParameter(field.Href); param != "" {
			// Fallback to href-based extraction
			params = append(params, param)
		}
	}

	return params
}

// parseObservations parses the grid coverage data into observations
func (p *StationGroupingParser) parseObservations(obs GridSeriesObservation) ([]WindObservation, error) {
	// Try MultiPointCoverage first, then fall back to RectifiedGridCoverage
	var positionsStr, dataStr string
	var rangeFields []Field

	if obs.Result.MultiPointCoverage.GmlID != "" {
		// Use MultiPointCoverage structure
		positionsStr = strings.TrimSpace(obs.Result.MultiPointCoverage.DomainSet.SimpleMultiPoint.Positions)
		dataStr = strings.TrimSpace(obs.Result.MultiPointCoverage.RangeSet.DataBlock.DoubleOrNilReasonTupleList)
		rangeFields = obs.Result.MultiPointCoverage.RangeType.DataRecord.Fields
	} else {
		// Fall back to RectifiedGridCoverage structure
		positionsStr = strings.TrimSpace(obs.Result.RectifiedGridCoverage.DomainSet.SimpleMultiPoint.Positions)
		dataStr = strings.TrimSpace(obs.Result.RectifiedGridCoverage.RangeSet.DataBlock.DoubleOrNilReasonTupleList)
		rangeFields = obs.Result.RectifiedGridCoverage.RangeType.DataRecord.Fields
	}

	if positionsStr == "" {
		return nil, fmt.Errorf("no position data in observation")
	}

	if dataStr == "" {
		return nil, fmt.Errorf("no data in observation")
	}

	// Split position data into individual position records (lat lon time)
	positionFields := strings.Fields(positionsStr)

	// Split data values
	dataValues := strings.Fields(dataStr)

	// Each position record has 3 fields: lat, lon, time
	numPositions := len(positionFields) / 3
	if numPositions == 0 {
		return nil, fmt.Errorf("no valid positions found")
	}

	// Extract parameter mapping from range fields
	var parameterMapping []WindParameter
	if len(rangeFields) > 0 {
		parameterMapping = p.extractParametersFromFields(rangeFields)
	}

	// Determine how many parameters we have from the range type
	paramCount := len(rangeFields)
	if paramCount == 0 {
		paramCount = 1 // Default to 1 parameter for single-parameter queries
	}

	// Check if we have the right amount of data values
	expectedDataValues := numPositions * paramCount
	if len(dataValues) != expectedDataValues {
		return nil, fmt.Errorf("data mismatch: %d positions with %d parameters = %d expected values, but got %d data values",
			numPositions, paramCount, expectedDataValues, len(dataValues))
	}

	var observations []WindObservation

	for i := 0; i < numPositions; i++ {
		// Extract timestamp from position (index 2 of each 3-field record)
		timestampStr := positionFields[i*3+2]
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			p.handleError(fmt.Errorf("failed to parse timestamp: %w", err))
			continue
		}

		// Create observation
		observation := WindObservation{
			Timestamp: time.Unix(timestamp, 0),
		}

		// Parse parameter values for this position using proper parameter mapping
		for j := 0; j < paramCount; j++ {
			valueIndex := i*paramCount + j
			if valueIndex >= len(dataValues) {
				break
			}

			valueStr := dataValues[valueIndex]
			if valueStr != "NaN" {
				value, err := strconv.ParseFloat(valueStr, 64)
				if err == nil {
					// Assign value based on actual parameter mapping from rangeFields
					var param WindParameter
					if j < len(parameterMapping) {
						param = parameterMapping[j]
					} else if paramCount == 1 {
						// For single parameter queries, use the URL-based extraction
						param = WindSpeedMS // Default assumption for single param
					}

					// Assign to correct field based on parameter type
					switch param {
					case WindSpeedMS:
						observation.WindSpeed = &value
					case WindGustMS:
						observation.WindGust = &value
					case WindDirection:
						observation.WindDirection = &value
					}
				}
			}
		}

		observations = append(observations, observation)
	}

	p.stats.ProcessedObservations += len(observations)

	return observations, nil
}

// mergeObservations merges new observations with existing station data
func (p *StationGroupingParser) mergeObservations(stationData *StationWindData, newObs []WindObservation, parameter WindParameter) {
	// Create timestamp index for existing observations
	obsIndex := make(map[time.Time]*WindObservation)
	for i := range stationData.Observations {
		obs := &stationData.Observations[i]
		obsIndex[obs.Timestamp] = obs
	}

	// Merge new observations based on parameter type
	for _, newOb := range newObs {
		existingObs, exists := obsIndex[newOb.Timestamp]
		if !exists {
			// Create new observation
			existingObs = &WindObservation{Timestamp: newOb.Timestamp}
			stationData.Observations = append(stationData.Observations, *existingObs)
			existingObs = &stationData.Observations[len(stationData.Observations)-1]
			obsIndex[newOb.Timestamp] = existingObs
		}

		// Assign value based on parameter type
		switch parameter {
		case WindSpeedMS:
			if newOb.WindSpeed != nil {
				existingObs.WindSpeed = newOb.WindSpeed
			}
		case WindGustMS:
			if newOb.WindSpeed != nil { // Data comes as single value
				existingObs.WindGust = newOb.WindSpeed
			}
		case WindDirection:
			if newOb.WindSpeed != nil { // Data comes as single value
				existingObs.WindDirection = newOb.WindSpeed
			}
		}
	}
}

// mergeMultiParameterObservations merges observations that already contain all parameters
func (p *StationGroupingParser) mergeMultiParameterObservations(stationData *StationWindData, newObs []WindObservation) {
	// Create timestamp index for existing observations
	obsIndex := make(map[time.Time]*WindObservation)
	for i := range stationData.Observations {
		obs := &stationData.Observations[i]
		obsIndex[obs.Timestamp] = obs
	}

	// Merge new observations with all their parameters
	for _, newOb := range newObs {
		existingObs, exists := obsIndex[newOb.Timestamp]
		if !exists {
			// Create new observation - copy all parameters from newOb
			newObsCopy := WindObservation{
				Timestamp:     newOb.Timestamp,
				WindSpeed:     newOb.WindSpeed,
				WindGust:      newOb.WindGust,
				WindDirection: newOb.WindDirection,
				Quality:       newOb.Quality,
			}
			stationData.Observations = append(stationData.Observations, newObsCopy)
			existingObs = &stationData.Observations[len(stationData.Observations)-1]
			obsIndex[newOb.Timestamp] = existingObs
		} else {
			// Merge all parameters from newOb into existingObs
			if newOb.WindSpeed != nil {
				existingObs.WindSpeed = newOb.WindSpeed
			}
			if newOb.WindGust != nil {
				existingObs.WindGust = newOb.WindGust
			}
			if newOb.WindDirection != nil {
				existingObs.WindDirection = newOb.WindDirection
			}
			if newOb.Quality != "" {
				existingObs.Quality = newOb.Quality
			}
		}
	}
}

// finalizeAndNotify sorts observations and calls callbacks for each station
func (p *StationGroupingParser) finalizeAndNotify() {
	for _, stationData := range p.stationData {
		// Sort observations by timestamp
		sort.Slice(stationData.Observations, func(i, j int) bool {
			return stationData.Observations[i].Timestamp.Before(stationData.Observations[j].Timestamp)
		})

		// Notify station data complete
		if p.callbacks.OnStationData != nil {
			if err := p.callbacks.OnStationData(*stationData); err != nil {
				p.handleError(fmt.Errorf("station data callback failed for %s: %w", stationData.StationID, err))
			}
		}

	}

	// Calculate final stats
	p.stats.Duration = time.Since(p.startTime)

	// Notify completion
	if p.callbacks.OnComplete != nil {
		p.callbacks.OnComplete(p.stats)
	}
}

// handleError handles parsing errors
func (p *StationGroupingParser) handleError(err error) {
	p.stats.ErrorCount++
	if p.callbacks.OnError != nil {
		p.callbacks.OnError(err)
	} else {
		log.Printf("Parser error: %v", err)
	}
}
