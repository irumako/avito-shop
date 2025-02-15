package main

import (
	"avito-shop/internal/data"
	"avito-shop/internal/validator"
	"errors"
	"github.com/pascaldekloe/jwt"
	"math/rand"
	"net/http"
	"time"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateUsername(v, input.Username)
	data.ValidatePasswordPlaintext(v, input.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	user, err := app.models.Users.GetByUsername(input.Username)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			user = &data.User{
				Username: input.Username,
			}

			err = user.Password.Set(input.Password)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			err = app.models.Users.Insert(user)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			balance := rand.Intn(100000-1000+1) + 1000

			wallet := &data.Wallet{
				UserId:  user.ID,
				Balance: float64(balance),
			}

			err = app.models.Wallets.Insert(wallet)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !match {
		app.invalidCredentialsResponse(w, r)
		return
	}

	var claims jwt.Claims
	claims.Subject = user.ID
	claims.Issued = jwt.NewNumericTime(time.Now())
	claims.NotBefore = jwt.NewNumericTime(time.Now())
	claims.Expires = jwt.NewNumericTime(time.Now().Add(24 * time.Hour))
	claims.Issuer = "avito-shop"
	claims.Audiences = []string{"avito-shop"}
	claims.Audiences = []string{"avito-shop"}

	jwtBytes, err := claims.HMACSign(jwt.HS256, []byte(app.config.Jwt.Secret))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"token": string(jwtBytes)}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
