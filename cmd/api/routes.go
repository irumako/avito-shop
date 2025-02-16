package main

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/api/info", app.requireAuthenticatedUser(app.getInfoHandler))
	router.HandlerFunc(http.MethodPost, "/api/sendCoin", app.requireAuthenticatedUser(app.sendCoinHandler))
	router.HandlerFunc(http.MethodGet, "/api/buy/:item", app.requireAuthenticatedUser(app.buyProductHandler))
	router.HandlerFunc(http.MethodPost, "/api/auth", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.authenticate(router))
}
