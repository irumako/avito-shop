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
	mockWallets := new(data.MockWalletModel)
	mockUsers := new(data.MockUserModel)
	mockTransactions := new(data.MockTransactionModel)

	models := data.Models{
		Wallets:      mockWallets,
		Users:        mockUsers,
		Transactions: mockTransactions,
	}

	app := &application{
		models: models,
		logger: log.New(io.Discard, "", 0),
	}

	testUser := &data.User{
		ID:       "sender1",
		Username: "testuser",
	}

	tests := []struct {
		name             string
		requestBody      string
		setupMocks       func()
		expectedStatus   int
		expectedResponse interface{}
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
			name:             "Validation error (empty toUser and zero amount)",
			requestBody:      `{"toUser": "", "amount": 0}`,
			setupMocks:       func() {},
			expectedStatus:   http.StatusUnprocessableEntity,
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
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
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
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				mockWallets.On("GetByUserId", "receiver1").Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus:   http.StatusNotFound,
			expectedResponse: map[string]interface{}{"errors": "Could not be found"},
		},
		{
			name:        "Insufficient funds",
			requestBody: `{"toUser": "receiver", "amount": 100}`,
			setupMocks: func() {
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 50}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
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
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
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
				wallet := &data.Wallet{ID: "wallet_sender", UserId: "sender1", Balance: 1000}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", "sender1").Return(wallet, nil)
				receiver := &data.User{ID: "receiver1", Username: "receiver"}
				mockUsers.ExpectedCalls = nil
				mockUsers.On("GetByUsername", "receiver").Return(receiver, nil)
				receiverWallet := &data.Wallet{ID: "wallet_receiver", UserId: "receiver1", Balance: 500}
				mockWallets.On("GetByUserId", "receiver1").Return(receiverWallet, nil)
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("SendCoinTX", mock.MatchedBy(func(tx *data.Transaction) bool {
					return tx.SenderWalletId == wallet.ID &&
						tx.ReceiverWalletId == receiverWallet.ID &&
						tx.Amount == 100
				})).Return(nil)
			},
			expectedStatus:   http.StatusOK,
			expectedResponse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockWallets.ExpectedCalls = nil
			mockUsers.ExpectedCalls = nil
			mockTransactions.ExpectedCalls = nil

			tt.setupMocks()

			req := httptest.NewRequest("POST", "/send", nil)
			req.Body = io.NopCloser(strings.NewReader(tt.requestBody))
			req = req.WithContext(context.WithValue(req.Context(), userContextKey, testUser))

			rr := httptest.NewRecorder()
			app.sendCoinHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.name)

			if tt.expectedResponse != nil {
				var got map[string]interface{}
				err := json.Unmarshal(rr.Body.Bytes(), &got)
				assert.NoError(t, err, tt.name)
				if tt.expectedResponse == "validation" {
					_, ok := got["errors"]
					assert.True(t, ok, tt.name)
				} else {
					assert.Equal(t, tt.expectedResponse, got, tt.name)
				}
			}
		})
	}
}
