package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Wallet struct {
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	UserId    string    `json:"user_id"`
	Balance   float64   `json:"balance"`
}

type WalletModel struct {
	DB *sql.DB
}

var (
	ErrDuplicateWallet = errors.New("duplicate wallet")
)

func (w WalletModel) Insert(wallet *Wallet) error {
	query := `
			INSERT INTO wallets (user_id, balance)
			VALUES ($1, $2)
			RETURNING id, updated_at
			`

	args := []interface{}{wallet.UserId, wallet.Balance}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := w.DB.QueryRowContext(ctx, query, args...).Scan(&wallet.UserId, &wallet.UpdatedAt)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "wallets_user_id_key"`:
			return ErrDuplicateWallet
		default:
			return err
		}
	}
	return nil
}

func (w WalletModel) Get(id string) (*Wallet, error) {
	query := `
			SELECT id, updated_at, user_id, balance
			FROM wallets
			WHERE id = $1
			`
	var wallet Wallet
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := w.DB.QueryRowContext(ctx, query, id).Scan(
		&wallet.ID,
		&wallet.UpdatedAt,
		&wallet.UserId,
		&wallet.Balance,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &wallet, nil
}

func (w WalletModel) GetByUserId(userId string) (*Wallet, error) {
	query := `
			SELECT id, updated_at, user_id, balance
			FROM wallets
			WHERE user_id = $1
			`
	var wallet Wallet
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := w.DB.QueryRowContext(ctx, query, userId).Scan(
		&wallet.ID,
		&wallet.UpdatedAt,
		&wallet.UserId,
		&wallet.Balance,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &wallet, nil
}

func (w WalletModel) Update(wallet *Wallet) error {
	query := `
		UPDATE wallets
		SET balance = $1
		WHERE id = $2 
		RETURNING updated_at
		`
	args := []interface{}{
		wallet.Balance,
		wallet.ID,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := w.DB.QueryRowContext(ctx, query, args...).Scan(&wallet.UpdatedAt)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}
