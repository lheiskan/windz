// @vibe: 🤖 -- ai
package stations

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
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

	tests := []struct {
		name        string
		req         Request
		expectParts map[string]string
	}{
		{
			name: "Basic_Request",
			req:  Request{},
			expectParts: map[string]string{
				"service":        "WFS",
				"version":        "2.0.0",
				"request":        "getFeature",
				"storedquery_id": "fmi::ef::stations",
			},
		},
		{
			name: "With_BBox",
			req: Request{
				BBox: &BBox{
					MinLon: 24.0,
					MinLat: 60.0,
					MaxLon: 25.0,
					MaxLat: 61.0,
				},
			},
			expectParts: map[string]string{
				"bbox": "24.00,60.00,25.00,61.00",
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
	// Test XML data
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<wfs:FeatureCollection xmlns:wfs="http://www.opengis.net/wfs/2.0"
                       xmlns:ef="http://inspire.ec.europa.eu/schemas/ef/4.0"
                       xmlns:gml="http://www.opengis.net/gml/3.2">
  <wfs:member>
    <ef:EnvironmentalMonitoringFacility gml:id="station-100996">
      <gml:identifier codeSpace="http://xml.fmi.fi/namespace/stationcode/fmisid">100996</gml:identifier>
      <gml:name codeSpace="http://xml.fmi.fi/namespace/locationcode/name">Helsinki Harmaja</gml:name>
      <ef:representativePoint>
        <gml:Point>
          <gml:pos>60.10512 24.97539</gml:pos>
        </gml:Point>
      </ef:representativePoint>
      <ef:belongsTo title="AWS"/>
    </ef:EnvironmentalMonitoringFacility>
  </wfs:member>
</wfs:FeatureCollection>`

	// Create mock HTTP client
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(testXML)),
	}

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)

	req := Request{
		BBox: &SouthernFinlandBBox,
	}

	response, err := query.Execute(req)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we got data
	if len(response.Stations) == 0 {
		t.Error("Expected stations in response")
	}

	if response.Count != 1 {
		t.Errorf("Expected count=1, got %d", response.Count)
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

		if params.Get("storedquery_id") != "fmi::ef::stations" {
			t.Error("Expected correct storedquery_id")
		}

		if params.Get("bbox") == "" {
			t.Error("Expected bbox parameter")
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

	req := Request{}

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

	req := Request{}

	_, err := query.Execute(req)
	if err == nil {
		t.Error("Expected error for HTTP 404")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected 404 in error message, got: %v", err)
	}
}

func TestQueryExecuteWithCustomParser(t *testing.T) {
	// Test XML data
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<wfs:FeatureCollection xmlns:wfs="http://www.opengis.net/wfs/2.0"
                       xmlns:ef="http://inspire.ec.europa.eu/schemas/ef/4.0"
                       xmlns:gml="http://www.opengis.net/gml/3.2">
  <wfs:member>
    <ef:EnvironmentalMonitoringFacility gml:id="station-100996">
      <gml:identifier codeSpace="http://xml.fmi.fi/namespace/stationcode/fmisid">100996</gml:identifier>
      <gml:name codeSpace="http://xml.fmi.fi/namespace/locationcode/name">Helsinki Harmaja</gml:name>
      <ef:representativePoint>
        <gml:Point>
          <gml:pos>60.10512 24.97539</gml:pos>
        </gml:Point>
      </ef:representativePoint>
      <ef:belongsTo title="AWS"/>
    </ef:EnvironmentalMonitoringFacility>
  </wfs:member>
</wfs:FeatureCollection>`

	// Create mock HTTP client
	mockResp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(testXML)),
	}

	mockClient := &MockHTTPClient{
		Response: mockResp,
		Error:    nil,
	}

	query := NewQuery("https://opendata.fmi.fi/wfs", mockClient)
	customParser := NewParser()

	req := Request{}

	response, err := query.ExecuteWithParser(req, customParser)
	if err != nil {
		t.Fatalf("ExecuteWithParser failed: %v", err)
	}

	if len(response.Stations) == 0 {
		t.Error("Expected stations in response")
	}
}

// Benchmark query building
func BenchmarkQueryBuildURL(b *testing.B) {
	query := NewQuery("https://opendata.fmi.fi/wfs", nil)

	req := Request{
		BBox: &SouthernFinlandBBox,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := query.buildURL(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}