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

	router.HandlerFunc(http.MethodPost, "/api/buy/:item", app.requireAuthenticatedUser(app.buyProductHandler))
	router.HandlerFunc(http.MethodGet, "/api/auth", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.authenticate(router))
}
