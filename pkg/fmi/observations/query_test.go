package observations

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// MockHTTPClient for testing
type MockHTTPClient struct {
	Response *http.Response
	Error    error
	Requests []*http.Request
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.Requests = append(m.Requests, req)
	return m.Response, m.Error
}

func TestQueryBuildURL(t *testing.T) {
	query := NewQuery("https://opendata.fmi.fi/wfs", nil)

	startTime := time.Date(2025, 8, 31, 8, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 8, 31, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		req         Request
		expectParts map[string]string
	}{
		{
			name: "Multi_Station_Request",
			req: Request{
				StartTime:  startTime,
				EndTime:    endTime,
				StationIDs: []string{"100996", "101023", "151028"},
				Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
				UseGzip:    true,
			},
			expectParts: map[string]string{
				"service":        "WFS",
				"version":        "2.0.0",
				"request":        "getFeature",
				"storedquery_id": "fmi::observations::weather::multipointcoverage",
				"starttime":      "2025-08-31T08:00:00Z",
				"endtime":        "2025-08-31T10:00:00Z",
				"parameters":     "windspeedms,windgust,winddirection",
			},
		},
		{
			name: "Single_Station_With_BBox",
			req: Request{
				StartTime: startTime,
				EndTime:   endTime,
				BBox: &BBox{
					MinLon: 24.0,
					MinLat: 60.0,
					MaxLon: 25.0,
					MaxLat: 61.0,
				},
				Parameters: []WindParameter{WindSpeedMS},
			},
			expectParts: map[string]string{
				"bbox":       "24.00,60.00,25.00,61.00",
				"parameters": "windspeedms",
			},
		},
		{
			name: "Default_Parameters",
			req: Request{
				StartTime:  startTime,
				EndTime:    endTime,
				StationIDs: []string{"100996"},
			},
			expectParts: map[string]string{
				"parameters": "windspeedms,windgust,winddirection",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urlStr, err := query.buildURL(tt.req)
			if err != nil {
				t.Fatalf("buildURL failed: %v", err)
			}

			parsedURL, err := url.Parse(urlStr)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}

			params := parsedURL.Query()

			// Check expected parameters
			for key, expectedValue := range tt.expectParts {
				actualValue := params.Get(key)
				if actualValue != expectedValue {
					t.Errorf("Parameter %s: expected '%s', got '%s'", key, expectedValue, actualValue)
				}
			}

			// Check multi-value parameters (fmisid)
			if len(tt.req.StationIDs) > 0 {
				fmisids := params["fmisid"]
				if len(fmisids) != len(tt.req.StationIDs) {
					t.Errorf("Expected %d fmisid parameters, got %d", len(tt.req.StationIDs), len(fmisids))
				}

				for i, expectedID := range tt.req.StationIDs {
					if i < len(fmisids) && fmisids[i] != expectedID {
						t.Errorf("fmisid[%d]: expected '%s', got '%s'", i, expectedID, fmisids[i])
					}
				}
			}
		})
	}
}

func TestQueryCreateHTTPRequest(t *testing.T) {
	query := NewQuery("https://opendata.fmi.fi/wfs", nil)

	tests := []struct {
		name     string
		url      string
		useGzip  bool
		checkReq func(t *testing.T, req *http.Request)
	}{
		{
			name:    "Basic_Request",
			url:     "https://opendata.fmi.fi/wfs?service=WFS&version=2.0.0",
			useGzip: false,
			checkReq: func(t *testing.T, req *http.Request) {
				if req.Method != "GET" {
					t.Errorf("Expected GET method, got %s", req.Method)
				}

				if req.Header.Get("Accept-Encoding") != "" {
					t.Errorf("Expected no Accept-Encoding header, got '%s'", req.Header.Get("Accept-Encoding"))
				}
			},
		},
		{
			name:    "Gzip_Request",
			url:     "https://opendata.fmi.fi/wfs?service=WFS&version=2.0.0",
			useGzip: true,
			checkReq: func(t *testing.T, req *http.Request) {
				if req.Header.Get("Accept-Encoding") != "gzip" {
					t.Errorf("Expected 'gzip' Accept-Encoding header, got '%s'", req.Header.Get("Accept-Encoding"))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := query.createHTTPRequest(tt.url, tt.useGzip)
			if err != nil {
				t.Fatalf("createHTTPRequest failed: %v", err)
			}

			if req.URL.String() != tt.url {
				t.Errorf("Expected URL '%s', got '%s'", tt.url, req.URL.String())
			}

			tt.checkReq(t, req)
		})
	}
}

func TestQueryExecuteSuccess(t *testing.T) {
	// Load test XML data
	xmlData, err := os.ReadFile("testdata/test_three_station_response.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Create mock HTTP client
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(xmlData)),
	}

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)

	req := Request{
		StartTime:  time.Now().Add(-2 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996", "101023", "151028"},
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
		UseGzip:    false,
	}

	response, err := query.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we got data
	if len(response.Stations) == 0 {
		t.Error("Expected stations in response")
	}

	// Check that HTTP request was made
	if len(mockClient.Requests) != 1 {
		t.Errorf("Expected 1 HTTP request, got %d", len(mockClient.Requests))
	}

	// Verify the request URL contains expected parameters
	if len(mockClient.Requests) > 0 {
		reqURL := mockClient.Requests[0].URL
		params := reqURL.Query()

		if params.Get("service") != "WFS" {
			t.Error("Expected service=WFS in request")
		}

		if params.Get("storedquery_id") != "fmi::observations::weather::multipointcoverage" {
			t.Error("Expected correct storedquery_id")
		}

		fmisids := params["fmisid"]
		if len(fmisids) != 3 {
			t.Errorf("Expected 3 fmisid parameters, got %d", len(fmisids))
		}
	}
}

func TestQueryExecuteHTTPError(t *testing.T) {
	// Create mock HTTP client that returns error
	mockClient := &MockHTTPClient{
		Response: nil,
		Error:    http.ErrHandlerTimeout,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)

	req := Request{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996"},
	}

	_, err := query.Execute(req)
	if err == nil {
		t.Error("Expected error for HTTP failure")
	}

	if !strings.Contains(err.Error(), "failed to execute request") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestQueryExecuteHTTPStatusError(t *testing.T) {
	// Create mock HTTP client that returns 404
	errorBody := "Not Found"
	mockResp := &http.Response{
		StatusCode: 404,
		Status:     "404 Not Found",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(errorBody)),
	}

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)

	req := Request{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996"},
	}

	_, err := query.Execute(req)
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected 404 in error message, got: %v", err)
	}
}

func TestQueryExecuteWithGzipResponse(t *testing.T) {
	// Load test XML data
	xmlData, err := os.ReadFile("testdata/test_three_station_response.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Create mock HTTP response with gzip header
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(xmlData)),
	}
	mockResp.Header.Set("Content-Encoding", "gzip")

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)

	req := Request{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996"},
		UseGzip:    true,
	}

	// This should work even though we're not actually sending gzipped data
	// The parser will try to decompress but fail gracefully for non-gzipped content
	_, err = query.Execute(req)

	// We expect this to fail during gzip decompression since we're sending plain XML
	if err == nil {
		t.Error("Expected error due to fake gzip header")
	}

	// Check that gzip header was requested
	if len(mockClient.Requests) > 0 {
		req := mockClient.Requests[0]
		if req.Header.Get("Accept-Encoding") != "gzip" {
			t.Error("Expected gzip Accept-Encoding header")
		}
	}
}

func TestQueryExecuteWithCustomParser(t *testing.T) {
	// Load test XML data
	xmlData, err := os.ReadFile("testdata/test_three_station_response.xml")
	if err != nil {
		t.Fatalf("Failed to read test XML file: %v", err)
	}

	// Create mock HTTP client
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(xmlData)),
	}

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)
	customParser := NewParser()

	req := Request{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996"},
	}

	response, err := query.ExecuteWithParser(req, customParser)
	if err != nil {
		t.Fatalf("ExecuteWithParser failed: %v", err)
	}

	if len(response.Stations) == 0 {
		t.Error("Expected stations in response")
	}
}

// Integration-style test (would require real API access)
func TestQueryIntegration(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// This would test against the real FMI API
	client := &http.Client{Timeout: 30 * time.Second}
	query := NewQuery("https://opendata.fmi.fi/wfs", client)

	req := Request{
		StartTime:  time.Now().Add(-1 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996"}, // Just Harmaja
		UseGzip:    true,
	}

	response, err := query.Execute(req)
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	t.Logf("Got %d stations with %d total observations",
		len(response.Stations), response.Stats.TotalObservations)
}

// Benchmark query building
func BenchmarkQueryBuildURL(b *testing.B) {
	query := NewQuery("https://opendata.fmi.fi/wfs", nil)

	req := Request{
		StartTime:  time.Now().Add(-2 * time.Hour),
		EndTime:    time.Now(),
		StationIDs: []string{"100996", "101023", "151028"},
		Parameters: []WindParameter{WindSpeedMS, WindGustMS, WindDirection},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := query.buildURL(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
