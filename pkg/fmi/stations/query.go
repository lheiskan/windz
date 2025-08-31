// @vibe: 🤖 -- ai
package stations

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Query handles FMI stations API queries
type Query struct {
	baseURL    string
	httpClient HTTPClient
}

// NewQuery creates a new stations query handler
func NewQuery(baseURL string, client HTTPClient) *Query {
	return &Query{
		baseURL:    baseURL,
		httpClient: client,
	}
}

// Execute performs the query and returns parsed stations
func (q *Query) Execute(req Request) (*Response, error) {
	// Build query URL
	requestURL, err := q.buildURL(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create HTTP request
	httpReq, err := q.createHTTPRequest(requestURL, req.UseGzip)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Execute HTTP request
	resp, err := q.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, q.parseHTTPError(resp)
	}

	// Parse the response
	parser := NewParser()
	return parser.Parse(resp.Body)
}

// ExecuteWithParser executes query and uses provided parser
func (q *Query) ExecuteWithParser(req Request, parser *Parser) (*Response, error) {
	// Build query URL
	requestURL, err := q.buildURL(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create HTTP request
	httpReq, err := q.createHTTPRequest(requestURL, req.UseGzip)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Execute HTTP request
	resp, err := q.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, q.parseHTTPError(resp)
	}

	// Parse using provided parser
	return parser.Parse(resp.Body)
}

func (q *Query) buildURL(req Request) (string, error) {
	params := url.Values{}
	params.Set("service", "WFS")
	params.Set("version", "2.0.0")
	params.Set("request", "getFeature")
	params.Set("storedquery_id", "fmi::ef::stations")

	if req.BBox != nil {
		params.Set("bbox", req.BBox.String())
	}

	return fmt.Sprintf("%s?%s", q.baseURL, params.Encode()), nil
}

func (q *Query) createHTTPRequest(url string, useGzip bool) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Request gzip encoding if specified
	if useGzip {
		req.Header.Set("Accept-Encoding", "gzip")
	}

	return req, nil
}

func (q *Query) parseHTTPError(resp *http.Response) error {
	// Try to read the response body for error details
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	// Try to parse as FMI error (basic implementation)
	if len(body) > 0 {
		bodyStr := string(body)
		if contains := func(s, substr string) bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}; contains(bodyStr, "ExceptionText") {
			return fmt.Errorf("FMI API error: %s", bodyStr)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
}
