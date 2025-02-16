package main

import (
	"avito-shop/internal/data"
	"errors"
	"net/http"
)

func (app *application) getInfoHandler(w http.ResponseWriter, r *http.Request) {

	user := app.contextGetUser(r)
	app.logger.Println("info")

	wallet, err := app.models.Wallets.GetByUserId(user.ID)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	app.logger.Println("wallet")
	orders, err := app.models.Orders.GetByWalletId(wallet.ID)
	if err != nil {
		if !errors.Is(err, data.ErrRecordNotFound) {
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	app.logger.Println("orders")

	type item struct {
		Type     string `json:"type"`
		Quantity int    `json:"quantity"`
	}

	var items = make(map[string]int)
	for _, order := range orders {
		product, err := app.models.Products.Get(order.ProductId)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		items[product.Name]++
	}

	var inventory = make([]item, 0)
	for k, v := range items {
		inventory = append(inventory, item{Type: k, Quantity: v})
	}

	type fromTransaction struct {
		FromUser string  `json:"fromUser"`
		Amount   float64 `json:"amount"`
	}

	var fromTransactions = make([]fromTransaction, 0)

	receivedTransactions, err := app.models.Transactions.GetByReceiverWalletId(wallet.ID)
	if err != nil {
		if !errors.Is(err, data.ErrRecordNotFound) {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	for _, transaction := range receivedTransactions {
		walletSender, err := app.models.Wallets.Get(transaction.SenderWalletId)
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}

		sender, err := app.models.Users.Get(walletSender.UserId)
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}

		fromTransactions = append(fromTransactions,
			fromTransaction{FromUser: sender.Username, Amount: transaction.Amount})
	}

	type toTransaction struct {
		ToUser string  `json:"toUser"`
		Amount float64 `json:"amount"`
	}

	var toTransactions = make([]toTransaction, 0)

	sendTransactions, err := app.models.Transactions.GetBySenderWalletId(wallet.ID)
	if err != nil {
		if !errors.Is(err, data.ErrRecordNotFound) {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	for _, transaction := range sendTransactions {
		walletReceive, err := app.models.Wallets.Get(transaction.ReceiverWalletId)
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}
		sender, err := app.models.Users.Get(walletReceive.UserId)
		if err != nil {
			app.notFoundResponse(w, r)
			return
		}

		toTransactions = append(toTransactions,
			toTransaction{ToUser: sender.Username, Amount: transaction.Amount})
	}

	env := envelope{
		"coins":     wallet.Balance,
		"inventory": inventory,
		"coinHistory": map[string]interface{}{
			"received": fromTransactions,
			"sent":     toTransactions,
		},
	}

	err = app.writeJSON(w, http.StatusOK, env, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
