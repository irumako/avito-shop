package main

import (
	"avito-shop/internal/data"
	"errors"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) buyProductHandler(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	productName := params.ByName("item")
	if productName == "" {
		app.notFoundResponse(w, r)
		return
	}

	product, err := app.models.Products.GetByName(productName)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user := app.contextGetUser(r)

	wallet, err := app.models.Wallets.GetByUserId(user.ID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	order := &data.Order{
		WalletId:     wallet.ID,
		ProductId:    product.ID,
		PurchaseCost: product.Cost,
	}

	err = app.models.Orders.BuyProductTX(order)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrInsufficientFunds):
			app.insufficientFundsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
}
