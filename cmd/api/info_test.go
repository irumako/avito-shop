package main

import (
	"avito-shop/internal/data"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestGetInfoHandler(t *testing.T) {
	// Create mocks for the models we already have.
	mockWallets := new(data.MockWalletModel)
	mockOrders := new(data.MockOrderModel)
	mockProducts := new(data.MockProductModel)
	mockTransactions := new(data.MockTransactionModel)
	mockUsers := new(data.MockUserModel)

	models := data.Models{
		Wallets:      mockWallets,
		Orders:       mockOrders,
		Products:     mockProducts,
		Users:        mockUsers,
		Transactions: mockTransactions,
	}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	app := &testApp{
		application: &application{
			logger: logger,
			models: models,
		},
	}

	testUser := &data.User{
		ID:       "user123",
		Username: "testuser",
	}

	tests := []struct {
		name             string
		setupMocks       func()
		expectedStatus   int
		expectedEnvelope map[string]interface{}
	}{
		{
			name: "Wallet not found",
			setupMocks: func() {
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(nil, sql.ErrNoRows)
			},
			expectedStatus: http.StatusNotFound,
			// Your notFoundResponse returns an envelope with key "errors" and message "Could not be found"
			expectedEnvelope: map[string]interface{}{
				"errors": "Could not be found",
			},
		},
		{
			name: "Successful info with no orders and no transactions",
			setupMocks: func() {
				// Return a valid wallet.
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   1000.0,
					UserId:    testUser.ID,
					UpdatedAt: time.Now(),
				}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				// Orders: return an empty slice.
				mockOrders.ExpectedCalls = nil
				mockOrders.On("GetByWalletId", wallet.ID).Return([]data.Order{}, nil)

				// Transactions: simulate "not found" errors.
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("GetByReceiverWalletId", wallet.ID).Return([]data.Transaction{}, data.ErrRecordNotFound)
				mockTransactions.On("GetBySenderWalletId", wallet.ID).Return([]data.Transaction{}, data.ErrRecordNotFound)
			},
			expectedStatus: http.StatusOK,
			expectedEnvelope: map[string]interface{}{
				"coins":     1000.0,
				"inventory": []interface{}{},
				"coinHistory": map[string]interface{}{
					"received": []interface{}{},
					"sent":     []interface{}{},
				},
			},
		},
		{
			name: "Successful info with orders and transactions",
			setupMocks: func() {
				// Wallet found.
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   1500.0,
					UserId:    testUser.ID,
					UpdatedAt: time.Now(),
				}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				// Orders: return one order.
				order := data.Order{
					ID:           "order1",
					WalletId:     wallet.ID,
					ProductId:    "prod1",
					PurchaseCost: 500.0,
					PurchasedAt:  time.Now(),
				}
				mockOrders.ExpectedCalls = nil
				mockOrders.On("GetByWalletId", wallet.ID).Return([]data.Order{order}, nil)

				// Product: return a product for the order.
				product := &data.Product{
					ID:   "prod1",
					Name: "Laptop",
					Cost: 500.0,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("Get", order.ProductId).Return(product, nil)

				// Transactions: simulate one received transaction.
				// (For a received transaction, the sender wallet ID is "walletSender")
				txReceived := data.Transaction{
					SenderWalletId:   "walletSender",
					ReceiverWalletId: wallet.ID,
					Amount:           200.0,
				}
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("GetByReceiverWalletId", wallet.ID).Return([]data.Transaction{txReceived}, nil)

				// And one sent transaction.
				txSent := data.Transaction{
					SenderWalletId:   wallet.ID,
					ReceiverWalletId: "walletReceiver",
					Amount:           150.0,
				}
				mockTransactions.On("GetBySenderWalletId", wallet.ID).Return([]data.Transaction{txSent}, nil)

				// For the received transaction, look up the sender's wallet.
				walletSender := &data.Wallet{
					ID:        "walletSender",
					UserId:    "userSender",
					Balance:   0,
					UpdatedAt: time.Now(),
				}
				// Assuming Wallets.Get is available.
				mockWallets.On("Get", "walletSender").Return(walletSender, nil)

				// For the sent transaction, look up the receiver's wallet.
				walletReceiver := &data.Wallet{
					ID:        "walletReceiver",
					UserId:    "userReceiver",
					Balance:   0,
					UpdatedAt: time.Now(),
				}
				mockWallets.On("Get", "walletReceiver").Return(walletReceiver, nil)

				// Now, for Users: for walletSender, return a user with Username "Alice".
				mockUsers.ExpectedCalls = nil
				mockUsers.On("Get", "userSender").Return(&data.User{
					ID:       "userSender",
					Username: "Alice",
				}, nil)
				// For walletReceiver, return a user with Username "Bob".
				mockUsers.On("Get", "userReceiver").Return(&data.User{
					ID:       "userReceiver",
					Username: "Bob",
				}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedEnvelope: map[string]interface{}{
				"coins": 1500.0,
				"inventory": []interface{}{
					map[string]interface{}{
						"type":     "Laptop",
						"quantity": float64(1),
					},
				},
				"coinHistory": map[string]interface{}{
					"received": []interface{}{
						map[string]interface{}{
							"fromUser": "Alice",
							"amount":   200.0,
						},
					},
					"sent": []interface{}{
						map[string]interface{}{
							"toUser": "Bob",
							"amount": 150.0,
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations.
			mockWallets.ExpectedCalls = nil
			mockOrders.ExpectedCalls = nil
			mockProducts.ExpectedCalls = nil
			mockTransactions.ExpectedCalls = nil
			mockUsers.ExpectedCalls = nil

			tt.setupMocks()

			// Create a new HTTP request.
			req := httptest.NewRequest("GET", "/info", nil)
			// Insert the dummy authenticated user in the context using the key "user".
			req = req.WithContext(context.WithValue(req.Context(), userContextKey, testUser))
			// (If your routing requires httprouter.Params, set it as well.)
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, httprouter.Params{}))

			rr := httptest.NewRecorder()

			// Call the handler.
			app.getInfoHandler(rr, req)

			// Check the status code.
			assert.Equal(t, tt.expectedStatus, rr.Code, tt.name)

			// Unmarshal the JSON response.
			var got map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &got)
			assert.NoError(t, err, tt.name)

			// Compare the envelope.
			assert.Equal(t, tt.expectedEnvelope, got, tt.name)

			// Assert that all mock expectations were met.
			mockWallets.AssertExpectations(t)
			mockOrders.AssertExpectations(t)
			mockProducts.AssertExpectations(t)
			mockTransactions.AssertExpectations(t)
			mockUsers.AssertExpectations(t)
		})
	}
}
