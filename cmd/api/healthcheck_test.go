package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckHandler(t *testing.T) {
	// Create a minimal application instance.
	//The writeJSON method from helpers.go will be used.
	app := &application{}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthcheck", nil)

	// Call the healthcheckHandler.
	app.healthcheckHandler(rr, req)

	// Check that the status code is 200 OK.
	assert.Equal(t, http.StatusOK, rr.Code)

	// Unmarshal the JSON response.
	var got map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &got)
	assert.NoError(t, err)

	// Define the expected response envelope.
	expected := map[string]interface{}{
		"status": "available",
		"system_info": map[string]interface{}{
			"version": version,
		},
	}

	// Compare the expected envelope with the actual response.
	assert.Equal(t, expected, got)
}
