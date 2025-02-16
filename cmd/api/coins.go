package main

import (
	"avito-shop/internal/data"
	"avito-shop/internal/validator"
	"errors"
	"net/http"
)

func (app *application) sendCoinHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ToUser string  `json:"toUser"`
		Amount float64 `json:"amount"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := validator.New()
	data.ValidateUsername(v, input.ToUser)
	data.ValidateAmount(v, input.Amount)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user := app.contextGetUser(r)
	wallet, err := app.models.Wallets.GetByUserId(user.ID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	receiver, err := app.models.Users.GetByUsername(input.ToUser)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	receiveWallet, err := app.models.Wallets.GetByUserId(receiver.ID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	transaction := &data.Transaction{
		SenderWalletId:   wallet.ID,
		ReceiverWalletId: receiveWallet.ID,
		Amount:           input.Amount,
	}

	err = app.models.Transactions.SendCoinTX(transaction)
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
