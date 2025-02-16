package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckHandler(t *testing.T) {
	app := &application{}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthcheck", nil)

	app.healthcheckHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var got map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &got)
	assert.NoError(t, err)

	expected := map[string]interface{}{
		"status": "available",
		"system_info": map[string]interface{}{
			"version": version,
		},
	}

	assert.Equal(t, expected, got)
}
