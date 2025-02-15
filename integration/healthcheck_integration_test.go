package integration

import (
	"io"
	"net/http"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	url := "http://localhost:8080/healthcheck"

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("Error sending GET request to %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	expectedBody := "{\"status\":\"available\",\"system_info\":{\"version\":\"1.0.0\"}}\n"
	if string(body) != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, string(body))
	}
}
