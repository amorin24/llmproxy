package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func CreateTestRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	var bodyReader io.Reader
	
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewBuffer(bodyBytes)
	}
	
	req, err := http.NewRequest(method, path, bodyReader)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	return req
}

func PerformRequest(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func DecodeResponse(t *testing.T, rr *httptest.ResponseRecorder, v interface{}) {
	if err := json.NewDecoder(rr.Body).Decode(v); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
}

func AssertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	if status := rr.Code; status != expected {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, expected)
	}
}

func AssertHeader(t *testing.T, rr *httptest.ResponseRecorder, header, expected string) {
	if value := rr.Header().Get(header); value != expected {
		t.Errorf("Handler returned wrong header value: got %v want %v", value, expected)
	}
}

func AssertHeaderContains(t *testing.T, rr *httptest.ResponseRecorder, header, expected string) {
	if value := rr.Header().Get(header); value == "" || !bytes.Contains([]byte(value), []byte(expected)) {
		t.Errorf("Handler header %s does not contain %s, got: %s", header, expected, value)
	}
}
