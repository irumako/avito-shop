package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Users    UserModel
	Wallets  WalletModel
	Products ProductModel
	Orders   OrderModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:    UserModel{DB: db},
		Wallets:  WalletModel{DB: db},
		Products: ProductModel{DB: db},
		Orders:   OrderModel{DB: db},
	}
}
