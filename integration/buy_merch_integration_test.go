package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestBuyMerch(t *testing.T) {
	authURL := "http://localhost:8080/api/auth"
	authPayload := map[string]string{
		"username": "admin",
		"password": "adminpassword",
	}

	payloadBytes, err := json.Marshal(authPayload)
	if err != nil {
		t.Fatalf("Failed to marshal auth payload: %v", err)
	}

	resp, err := http.Post(authURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		t.Fatalf("Auth request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Fatalf("Expected auth status 201, got %d", resp.StatusCode)
	}

	var authResponse struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		t.Fatalf("Failed to decode auth response: %v", err)
	}
	if authResponse.Token == "" {
		t.Fatal("Empty token in auth response")
	}

	// Покупаем ручку
	buyURL := "http://localhost:8080/api/buy/pen"
	req, err := http.NewRequest("GET", buyURL, nil)
	if err != nil {
		t.Fatalf("Failed to create buy request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authResponse.Token)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Buy request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected buy status 200, got %d", resp.StatusCode)
	}

	// Проверяем изменения /info
	infoURL := "http://localhost:8080/api/info"
	req, err = http.NewRequest("GET", infoURL, nil)
	if err != nil {
		t.Fatalf("Failed to create info request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+authResponse.Token)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Info request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected info status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read info response body: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("Info response body is empty")
	}

	expectedBody := "{\"coinHistory\":{\"received\":[],\"sent\":[]},\"coins\":999990,\"inventory\":[{\"type\":\"pen\",\"quantity\":1}]}\n"
	if string(body) != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, string(body))
	}
}
