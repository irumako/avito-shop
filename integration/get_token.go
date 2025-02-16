package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type unexpectedStatusError struct {
	expected int
	got      int
}

func (e *unexpectedStatusError) Error() string {
	return "unexpected HTTP status: expected " + http.StatusText(e.expected) + ", got " + http.StatusText(e.got)
}

type unexpectedResponseError struct {
	msg string
}

func (e *unexpectedResponseError) Error() string {
	return "unexpected response: " + e.msg
}

func getToken(client *http.Client, authURL string, payload interface{}, expectedStatus int) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(authURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		return "", &unexpectedStatusError{expected: expectedStatus, got: resp.StatusCode}
	}
	var tokenResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}
	if tokenResp.Token == "" {
		return "", &unexpectedResponseError{"empty token"}
	}
	return tokenResp.Token, nil
}
