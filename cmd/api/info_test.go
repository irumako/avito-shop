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
			expectedEnvelope: map[string]interface{}{
				"errors": "Could not be found",
			},
		},
		{
			name: "Successful info with no orders and no transactions",
			setupMocks: func() {
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   1000.0,
					UserId:    testUser.ID,
					UpdatedAt: time.Now(),
				}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				mockOrders.ExpectedCalls = nil
				mockOrders.On("GetByWalletId", wallet.ID).Return([]data.Order{}, nil)

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
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   1500.0,
					UserId:    testUser.ID,
					UpdatedAt: time.Now(),
				}
				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				order := data.Order{
					ID:           "order1",
					WalletId:     wallet.ID,
					ProductId:    "prod1",
					PurchaseCost: 500.0,
					PurchasedAt:  time.Now(),
				}
				mockOrders.ExpectedCalls = nil
				mockOrders.On("GetByWalletId", wallet.ID).Return([]data.Order{order}, nil)

				product := &data.Product{
					ID:   "prod1",
					Name: "Laptop",
					Cost: 500.0,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("Get", order.ProductId).Return(product, nil)

				txReceived := data.Transaction{
					SenderWalletId:   "walletSender",
					ReceiverWalletId: wallet.ID,
					Amount:           200.0,
				}
				mockTransactions.ExpectedCalls = nil
				mockTransactions.On("GetByReceiverWalletId", wallet.ID).Return([]data.Transaction{txReceived}, nil)

				txSent := data.Transaction{
					SenderWalletId:   wallet.ID,
					ReceiverWalletId: "walletReceiver",
					Amount:           150.0,
				}
				mockTransactions.On("GetBySenderWalletId", wallet.ID).Return([]data.Transaction{txSent}, nil)

				walletSender := &data.Wallet{
					ID:        "walletSender",
					UserId:    "userSender",
					Balance:   0,
					UpdatedAt: time.Now(),
				}

				mockWallets.On("Get", "walletSender").Return(walletSender, nil)

				walletReceiver := &data.Wallet{
					ID:        "walletReceiver",
					UserId:    "userReceiver",
					Balance:   0,
					UpdatedAt: time.Now(),
				}
				mockWallets.On("Get", "walletReceiver").Return(walletReceiver, nil)

				mockUsers.ExpectedCalls = nil
				mockUsers.On("Get", "userSender").Return(&data.User{
					ID:       "userSender",
					Username: "Alice",
				}, nil)
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
			mockWallets.ExpectedCalls = nil
			mockOrders.ExpectedCalls = nil
			mockProducts.ExpectedCalls = nil
			mockTransactions.ExpectedCalls = nil
			mockUsers.ExpectedCalls = nil

			tt.setupMocks()

			req := httptest.NewRequest("GET", "/info", nil)
			req = req.WithContext(context.WithValue(req.Context(), userContextKey, testUser))
			req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, httprouter.Params{}))

			rr := httptest.NewRecorder()

			app.getInfoHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code, tt.name)

			var got map[string]interface{}
			err := json.Unmarshal(rr.Body.Bytes(), &got)
			assert.NoError(t, err, tt.name)

			assert.Equal(t, tt.expectedEnvelope, got, tt.name)

			mockWallets.AssertExpectations(t)
			mockOrders.AssertExpectations(t)
			mockProducts.AssertExpectations(t)
			mockTransactions.AssertExpectations(t)
			mockUsers.AssertExpectations(t)
		})
	}
}
