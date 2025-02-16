package main

import (
	"avito-shop/internal/data"
	"context"
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type testApp struct{ *application }

func TestBuyProductHandler(t *testing.T) {
	mockProducts := new(data.MockProductModel)
	mockWallets := new(data.MockWalletModel)
	mockOrders := new(data.MockOrderModel)

	models := data.Models{
		Products: mockProducts,
		Wallets:  mockWallets,
		Orders:   mockOrders,
	}
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	app := &testApp{
		application: &application{
			logger: logger,
			models: models,
		},
	}

	testUser := &data.User{ID: "user123"}

	// Тестовые случаи
	tests := []struct {
		name           string
		params         httprouter.Params
		setupMocks     func()
		expectedStatus int
	}{
		{
			name: "Empty item parameter",
			params: httprouter.Params{
				{Key: "item", Value: ""},
			},
			setupMocks: func() {
				// No mocks are needed – the handler returns immediately.
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Product not found",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Product retrieval error",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(nil, errors.New("db error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Wallet not found",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				product := &data.Product{
					ID:   "prod123",
					Name: "laptop",
					Cost: 1000.0,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(product, nil)

				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(nil, data.ErrRecordNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Insufficient funds",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				product := &data.Product{
					ID:   "prod123",
					Name: "laptop",
					Cost: 1000.0,
				}
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   500.0,
					UpdatedAt: time.Now(),
					UserId:    testUser.ID,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(product, nil)

				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				mockOrders.ExpectedCalls = nil
				mockOrders.On("BuyProductTX", mock.MatchedBy(func(o *data.Order) bool {
					return o.WalletId == wallet.ID && o.ProductId == product.ID && o.PurchaseCost == product.Cost
				})).Return(data.ErrInsufficientFunds)
			},
			expectedStatus: http.StatusPaymentRequired,
		},
		{
			name: "Order creation error",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				product := &data.Product{
					ID:   "prod123",
					Name: "laptop",
					Cost: 1000.0,
				}
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   1500.0,
					UpdatedAt: time.Now(),
					UserId:    testUser.ID,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(product, nil)

				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				mockOrders.ExpectedCalls = nil
				mockOrders.On("BuyProductTX", mock.MatchedBy(func(o *data.Order) bool {
					return o.WalletId == wallet.ID && o.ProductId == product.ID && o.PurchaseCost == product.Cost
				})).Return(errors.New("order error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "Successful purchase",
			params: httprouter.Params{
				{Key: "item", Value: "laptop"},
			},
			setupMocks: func() {
				product := &data.Product{
					ID:   "prod123",
					Name: "laptop",
					Cost: 1000.0,
				}
				wallet := &data.Wallet{
					ID:        "wallet123",
					Balance:   2000.0,
					UpdatedAt: time.Now(),
					UserId:    testUser.ID,
				}
				mockProducts.ExpectedCalls = nil
				mockProducts.On("GetByName", "laptop").Return(product, nil)

				mockWallets.ExpectedCalls = nil
				mockWallets.On("GetByUserId", testUser.ID).Return(wallet, nil)

				mockOrders.ExpectedCalls = nil
				mockOrders.On("BuyProductTX", mock.MatchedBy(func(o *data.Order) bool {
					return o.WalletId == wallet.ID && o.ProductId == product.ID && o.PurchaseCost == product.Cost
				})).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}
	// Прогоняем тесты
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			req := httptest.NewRequest("GET", "/buy/item", nil)
			ctx := req.Context()
			ctx = context.WithValue(ctx, userContextKey, testUser)
			ctx = context.WithValue(ctx, httprouter.ParamsKey, tt.params)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			app.buyProductHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			mockProducts.AssertExpectations(t)
			mockWallets.AssertExpectations(t)
			mockOrders.AssertExpectations(t)
		})
	}
}
