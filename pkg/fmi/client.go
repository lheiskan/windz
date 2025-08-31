package fmi

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

// FMI OWS Exception structures for parsing error responses
type ExceptionReport struct {
	XMLName    xml.Name    `xml:"ExceptionReport"`
	Exceptions []Exception `xml:"Exception"`
}

type Exception struct {
	XMLName       xml.Name `xml:"Exception"`
	ExceptionCode string   `xml:"exceptionCode,attr"`
	ExceptionText []string `xml:"ExceptionText"`
}

// Client provides access to FMI Open Data API
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// NewClient creates a new FMI API client
func NewClient() *Client {
	return &Client{
		baseURL: "https://opendata.fmi.fi/wfs",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithHTTP creates a new FMI API client with custom HTTP client
func NewClientWithHTTP(httpClient *http.Client) *Client {
	return &Client{
		baseURL:    "https://opendata.fmi.fi/wfs",
		httpClient: httpClient,
	}
}

// WindDataRequest configures parameters for wind data queries
type WindDataRequest struct {
	// Time range for observations
	StartTime time.Time
	EndTime   time.Time

	// Geographic bounds (use predefined BBox constants or custom)
	BBox *BBox

	// Specific station IDs to query (optional, overrides BBox)
	StationID string

	// Multiple station IDs for multi-station queries
	StationIDs []string

	// Wind parameters to fetch
	Parameters []WindParameter

	// Max number of observations per station (optional)
	MaxObservations int

	// Request gzip compressed response
	UseGzip bool
}

// StreamWindDataByStation fetches wind data and streams results station by station
func (c *Client) StreamWindDataByStation(req WindDataRequest, callbacks WindDataCallbacks) error {
	// Build query parameters
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "2.0.0")
	params.Set("request", "getFeature")
	params.Set("storedquery_id", "fmi::observations::weather::multipointcoverage")

	// Set time range
	params.Set("starttime", req.StartTime.UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("endtime", req.EndTime.UTC().Format("2006-01-02T15:04:05Z"))

	if req.StationID != "" {
		params.Set("fmisid", req.StationID)
	} else if req.BBox != nil {
		// Query by bounding box
		params.Set("bbox", req.BBox.String())
	} else {
		// Default to all of Finland
		params.Set("bbox", FinlandBBox.String())
	}

	// Set parameters to fetch
	if len(req.Parameters) > 0 {
		paramStr := ""
		for i, param := range req.Parameters {
			if i > 0 {
				paramStr += ","
			}
			paramStr += string(param)
		}
		params.Set("parameters", paramStr)
	} else {
		// Default to all wind parameters
		params.Set("parameters", "windspeedms,windgust,winddirection")
	}

	// Set max observations if specified
	if req.MaxObservations > 0 {
		params.Set("maxlocations", fmt.Sprintf("%d", req.MaxObservations))
	}

	// Build request URL
	requestURL := c.baseURL + "?" + params.Encode()

	// Make HTTP request
	resp, err := c.httpClient.Get(requestURL)
	if err != nil {
		return fmt.Errorf("failed to make request to FMI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to parse FMI error response
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // Close the original body

		if fmiError, err := parseFMIError(io.NopCloser(bytes.NewReader(bodyBytes))); err == nil {
			return fmt.Errorf("%s\nRequest URL: %s", fmiError, requestURL)
		}

		// Fallback to generic error if parsing failed
		return fmt.Errorf("FMI API returned status %d: %s\nRequest URL: %s\nResponse: %s",
			resp.StatusCode, resp.Status, requestURL, string(bodyBytes))
	}

	// Log successful request for debugging
	log.Printf("FMI API request successful (status %d): %s", resp.StatusCode, requestURL)

	// Create streaming parser and process response
	parser := NewStationGroupingParser(resp.Body, callbacks)
	return parser.Parse()
}

// StreamWindDataAllStations is a convenience method to fetch all Finnish stations
func (c *Client) StreamWindDataAllStations(startTime, endTime time.Time, callbacks WindDataCallbacks) error {
	req := WindDataRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		BBox:       &FinlandBBox,
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
	}
	return c.StreamWindDataByStation(req, callbacks)
}

// StreamWindDataRegion fetches wind data for a specific region
func (c *Client) StreamWindDataRegion(region BBox, startTime, endTime time.Time, callbacks WindDataCallbacks) error {
	req := WindDataRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		BBox:       &region,
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
	}
	return c.StreamWindDataByStation(req, callbacks)
}

// StreamWindDataStations fetches wind data for specific stations
func (c *Client) StreamWindDataStations(stationID string, startTime, endTime time.Time, callbacks WindDataCallbacks) error {
	req := WindDataRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		StationID:  stationID,
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
	}
	return c.StreamWindDataByStation(req, callbacks)
}

// FetchMultiStationData fetches wind data for multiple stations
func (c *Client) FetchMultiStationData(req WindDataRequest) ([]StationWindData, error) {
	// Build query parameters
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "2.0.0")
	params.Set("request", "getFeature")
	params.Set("storedquery_id", "fmi::observations::weather::multipointcoverage")

	// Set time range
	params.Set("starttime", req.StartTime.UTC().Format("2006-01-02T15:04:05Z"))
	params.Set("endtime", req.EndTime.UTC().Format("2006-01-02T15:04:05Z"))

	// Add station IDs
	if len(req.StationIDs) > 0 {
		for _, stationID := range req.StationIDs {
			params.Add("fmisid", stationID)
		}
	} else if req.StationID != "" {
		params.Set("fmisid", req.StationID)
	} else if req.BBox != nil {
		params.Set("bbox", req.BBox.String())
	}

	// Set parameters to fetch
	if len(req.Parameters) > 0 {
		paramStr := ""
		for i, param := range req.Parameters {
			if i > 0 {
				paramStr += ","
			}
			paramStr += string(param)
		}
		params.Set("parameters", paramStr)
	} else {
		params.Set("parameters", "windspeedms,windgust,winddirection")
	}

	requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Create request
	httpReq, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Request gzip encoding if specified
	if req.UseGzip {
		httpReq.Header.Set("Accept-Encoding", "gzip")
	}

	// Make HTTP request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		if fmiError, err := parseFMIError(bytes.NewReader(bodyBytes)); err == nil {
			return nil, fmt.Errorf("%s", fmiError)
		}
		return nil, fmt.Errorf("FMI API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Check if response is gzipped
	isGzipped := resp.Header.Get("Content-Encoding") == "gzip"

	// Parse the response
	return ParseMultiStationResponseWithGzip(resp.Body, isGzipped)
}

// FetchStations retrieves available weather stations from FMI
func (c *Client) FetchStations(bbox *BBox) ([]Station, error) {
	return c.GetStations()
}

// GetStations fetches all weather stations from FMI WFS stations query
func (c *Client) GetStations() ([]Station, error) {
	// Build the WFS request URL for stations
	params := url.Values{
		"service":        {"WFS"},
		"version":        {"2.0.0"},
		"request":        {"getFeature"},
		"storedquery_id": {"fmi::ef::stations"},
	}

	requestURL := fmt.Sprintf("%s?%s", c.baseURL, params.Encode())

	// Make HTTP request
	resp, err := c.httpClient.Get(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to parse FMI error response
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // Close the original body

		if fmiError, err := parseFMIError(bytes.NewReader(bodyBytes)); err == nil {
			return nil, fmt.Errorf("%s\nRequest URL: %s", fmiError, requestURL)
		}

		// Fallback to generic error if parsing failed
		return nil, fmt.Errorf("FMI API returned status %d\nRequest URL: %s\nResponse: %s",
			resp.StatusCode, requestURL, string(bodyBytes))
	}

	// Parse XML response
	var wfsResponse WFSStationResponse
	if err := parseStationXML(resp.Body, &wfsResponse); err != nil {
		return nil, fmt.Errorf("failed to parse station XML response: %w", err)
	}

	// Convert to our Station model
	stations := make([]Station, 0, len(wfsResponse.Members))
	for _, member := range wfsResponse.Members {
		if station := convertToStationModel(member); station != nil {
			stations = append(stations, *station)
		}
	}

	return stations, nil
}

// TestConnection verifies connectivity to FMI API
func (c *Client) TestConnection() error {
	// Make a simple request to verify the API is accessible
	testTime := time.Now().Add(-24 * time.Hour)
	testBBox := BBox{24.0, 60.0, 25.0, 61.0} // Small area around Helsinki

	req := WindDataRequest{
		StartTime:       testTime,
		EndTime:         testTime.Add(time.Hour),
		BBox:            &testBBox,
		Parameters:      []WindParameter{WindSpeedMS},
		MaxObservations: 1,
	}

	callbacks := WindDataCallbacks{
		OnStationData: func(stationData StationWindData) error {
			return nil // Just test connectivity, ignore data
		},
		OnError: func(err error) {
			// Errors will be returned by StreamWindDataByStation
		},
	}

	return c.StreamWindDataByStation(req, callbacks)
}

// parseFMIError attempts to parse FMI XML error response
func parseFMIError(body io.Reader) (string, error) {
	var report ExceptionReport
	if err := xml.NewDecoder(body).Decode(&report); err != nil {
		return "", err
	}

	if len(report.Exceptions) == 0 {
		return "Unknown FMI API error", nil
	}

	exc := report.Exceptions[0]
	errorMsg := fmt.Sprintf("FMI API Error [%s]", exc.ExceptionCode)

	if len(exc.ExceptionText) > 0 {
		errorMsg += ": " + exc.ExceptionText[0]
		// Add additional error details if available
		if len(exc.ExceptionText) > 1 {
			for _, text := range exc.ExceptionText[1:] {
				errorMsg += " | " + text
			}
		}
	}

	return errorMsg, nil
}
