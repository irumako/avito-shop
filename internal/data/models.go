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
	Products interface {
		Get(id string) (*Product, error)
		GetByName(name string) (*Product, error)
	}
	Wallets interface {
		Insert(wallet *Wallet) error
		Get(id string) (*Wallet, error)
		GetByUserId(userId string) (*Wallet, error)
		Update(wallet *Wallet) error
	}
	Orders interface {
		BuyProductTX(order *Order) error
		GetByWalletId(id string) ([]Order, error)
	}
	Users interface {
		Insert(user *User) error
		Update(user *User) error
		Get(id string) (*User, error)
		GetByUsername(username string) (*User, error)
	}
	Transactions interface {
		SendCoinTX(transaction *Transaction) error
		GetByReceiverWalletId(id string) ([]Transaction, error)
		GetBySenderWalletId(id string) ([]Transaction, error)
	}
}

func NewModels(db *sql.DB) Models {
	return Models{
		Users:        UserModel{DB: db},
		Wallets:      WalletModel{DB: db},
		Products:     ProductModel{DB: db},
		Orders:       OrderModel{DB: db},
		Transactions: TransactionModel{DB: db},
	}
}
