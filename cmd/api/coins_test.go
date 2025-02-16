package main

import (
	"avito-shop/internal/data"
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendCoinHandler(t *testing.T) {
	// Create mocks for Wallets, Users, and Transactions.
	mockWallets := new(data.MockWalletModel)
	mockUsers := new(data.MockUserModel)
	mockTransactions := new(data.MockTransactionModel)

	models := data.Models{
		Wallets:      mockWallets,
		Users:        mockUsers,
		Transactions: mockTransactions,
	}

	// Create a minimal application instance with a logger.
	app := &application{
		models: models,
		logger: log.New(io.Discard, "", 0),
	}

	// Define a dummy authenticated sender.
	// Ensure that the type matches what contextGetUser expects.
	// For this test we assume that our application expects a *struct{ID string}
	// stored under the "user" key.
	testUser := &data.User{
		ID:       "sender1",
		Username: "testuser",
	}

	// Table-driven test cases.
	tests := []struct {
		name             string
		requestBody      string
		setupMocks       func()
		expectedStatus   int
		expectedResponse interface{} // nil for success or a map for error responses; for validation, we check for presence of an "errors" key.
	}{
		{
			name:           "Bad JSON input",
			requestBody:    "{invalid json",
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedResponse: map[string]interface{}{
				"errors": "body contains badly-formed JSON (at character 2)",
			},
		},
		{
			name:           "Validation error (empty toUser and zero amount)",
			requestBody:    `{"toUser": "", "amount": 0}`,
			setupMocks:     func() {},
			expectedStatus: http.StatusUnprocessableEntity,
			// In this case we simply check that the JSON envelope has an "errors" key.
			expectedResponse: "validation",
		},
		{
			name:        "Sender wallet not found",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
			expectedResponse: map[string]interface{}{
				"errors": "Could not be found",
			},
		},
		{
			name:        "Receiver not found",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				// Sender wallet exists.
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				// Receiver lookup fails.
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus:   http.StatusUnauthorized,
			expectedResponse: map[string]interface{}{"errors": "invalid authentication credentials"},
		},
		{
			name:        "Receiver wallet not found",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				// Sender wallet exists.
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				// Receiver exists.
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				// Receiver wallet lookup fails.
				mockWallets.On("GetByUserId", "receiver1").Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus:   http.StatusNotFound,
			expectedResponse: map[string]interface{}{"errors": "Could not be found"},
		},
		{
			name:        "Insufficient funds",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				// Sender wallet exists but has low balance.
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 50}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				// Receiver exists.
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				// Receiver wallet exists.
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
				// Transaction fails with insufficient funds.
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("SendCoinTX", mock.MatchedBy(func(tx *data.Transaction) bool {
					return tx.SenderWalletId == wallet.ID &&
						tx.ReceiverWalletId == receiverWallet.ID &&
						tx.Amount == 100
				})).Return(data.ErrInsufficientFunds)
			},
			expectedStatus:   http.StatusPaymentRequired,
			expectedResponse: map[string]interface{}{"errors": "insufficient funds"},
		},
		{
			name:        "Generic transaction error",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				// Sender wallet exists.
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				// Receiver exists.
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				// Receiver wallet exists.
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
				// Transaction fails with a generic error.
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("SendCoinTX", mock.MatchedBy(func(tx *data.Transaction) bool {
					return tx.SenderWalletId == wallet.ID &&
						tx.ReceiverWalletId == receiverWallet.ID &&
						tx.Amount == 100
				})).Return(errors.New("transaction error"))
			},
			expectedStatus:   http.StatusInternalServerError,
			expectedResponse: map[string]interface{}{"errors": "Server problem"},
		},
		{
			name:        "Success",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				// Sender wallet exists.
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				// Receiver exists.
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				// Receiver wallet exists.
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
				// Transaction succeeds.
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("SendCoinTX", mock.MatchedBy(func(tx *data.Transaction) bool {
					return tx.SenderWalletId == wallet.ID &&
						tx.ReceiverWalletId == receiverWallet.ID &&
						tx.Amount == 100
				})).Return(nil)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: nil, // No JSON response body on success.
		},
	}

	// Run each test case.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations.
			mockWallets.ExpectedCalls = nil
			mockUsers.ExpectedCalls = nil
			mockTransactions.ExpectedCalls = nil

			tt.setupMocks()

			// Create a new request with the JSON body.
			req := httptest.NewRequest("POST", "/send", nil)
			req.Body = io.NopCloser(strings.NewReader(tt.requestBody))
			// Inject the dummy authenticated sender into the request context.
			req = req.WithContext(context.WithValue(req.Context(), userContextKey, testUser))

			rr := httptest.NewRecorder()
			app.sendCoinHandler(rr, req)

			// Assert the HTTP status code.
			assert.Equal(t, tt.expectedStatus, rr.Code, tt.name)

			// If an error response is expected, unmarshal the response body and check.
			if tt.expectedResponse != nil {
				var got map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				assert.NoError(t, err, tt.name)
				if tt.expectedResponse == "validation" {
					// For validation errors, check that an "errors" key exists.
					_, ok := got["errors"]
					assert.True(t, ok, tt.name)
				} else {
					assert.Equal(t, tt.expectedResponse, got, tt.name)
				}
			}
		})
	}
}
