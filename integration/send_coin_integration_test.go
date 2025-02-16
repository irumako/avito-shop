package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestSendCoin(t *testing.T) {
	client := &http.Client{}

	// Admin
	adminAuthURL := "http://localhost:8080/api/auth"
	adminPayload := map[string]string{
		"username": "admin",
		"password": "adminpassword",
	}
	adminToken, err := getToken(client, adminAuthURL, adminPayload, http.StatusOK)
	if err != nil {
		t.Fatalf("Admin auth failed: %v", err)
	}

	// Создание irumako
	irumakoAuthURL := "http://localhost:8080/api/auth"
	irumakoPayload := map[string]string{
		"username": "irumako",
		"password": "adminpassword",
	}
	irumakoToken, err := getToken(client, irumakoAuthURL, irumakoPayload, http.StatusOK)
	if err != nil {
		t.Fatalf("irumako auth failed: %v", err)
	}

	// Отправляем монеты admin -> irumako
	sendCoinURL := "http://localhost:8080/api/sendCoin"
	coinPayload := map[string]interface{}{
		"toUser": "irumako",
		"amount": 100,
	}
	coinPayloadBytes, err := json.Marshal(coinPayload)
	if err != nil {
		t.Fatalf("Failed to marshal sendCoin payload: %v", err)
	}
	req, err := http.NewRequest("POST", sendCoinURL, bytes.NewReader(coinPayloadBytes))
	if err != nil {
		t.Fatalf("Failed to create sendCoin request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("SendCoin request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected sendCoin status 200, got %d", resp.StatusCode)
	}

	// irumako info
	infoURL := "http://localhost:8080/api/info"
	req, err = http.NewRequest("GET", infoURL, nil)
	if err != nil {
		t.Fatalf("Failed to create info request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+irumakoToken)
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

	expectedBody := "{\"coinHistory\":{\"received\":[{\"fromUser\":\"admin\",\"amount\":100}],\"sent\":[]},\"coins\":1100,\"inventory\":[]}\n"
	if string(body) != expectedBody {
		t.Errorf("Expected response body %q, got %q", expectedBody, string(body))
	}
}
