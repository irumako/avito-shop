package data

import (
	"avito-shop/internal/validator"
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/mock"
	"math"
	"time"
)

type Transaction struct {
	ID               string    `json:"id"`
	SenderWalletId   string    `json:"sender_wallet_id"`
	ReceiverWalletId string    `json:"receiver_wallet_id"`
	Amount           float64   `json:"amount"`
	CreatedAt        time.Time `json:"created_at"`
}

type TransactionModel struct {
	DB *sql.DB
}

func (t TransactionModel) SendCoinTX(transaction *Transaction) error {
	transactionQuery := `
		INSERT INTO transactions (sender_wallet_id, receiver_wallet_id, amount)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	senderWalletQuery := `
		UPDATE wallets
		SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
		RETURNING updated_at
	`
	receiverWalletQuery := `
		UPDATE wallets
		SET balance = balance + $1
		WHERE id = $2 
		RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := t.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)

	err = tx.QueryRowContext(
		ctx, transactionQuery, transaction.SenderWalletId, transaction.ReceiverWalletId, transaction.Amount).
		Scan(&transaction.ID, &transaction.CreatedAt)
	if err != nil {
		return err
	}

	var updatedAt time.Time
	err = tx.QueryRowContext(
		ctx, senderWalletQuery, transaction.Amount, transaction.SenderWalletId).
		Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsufficientFunds
		}
		return err
	}

	err = tx.QueryRowContext(
		ctx, receiverWalletQuery, transaction.Amount, transaction.ReceiverWalletId).
		Scan(&updatedAt)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (t TransactionModel) GetBySenderWalletId(id string) ([]Transaction, error) {
	query := `
		SELECT id, sender_wallet_id, receiver_wallet_id, amount, created_at
		FROM transactions
		WHERE sender_wallet_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := t.DB.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var transactions []Transaction

	for rows.Next() {
		var transaction Transaction
		err := rows.Scan(
			&transaction.ID,
			&transaction.SenderWalletId,
			&transaction.ReceiverWalletId,
			&transaction.Amount,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, ErrRecordNotFound
	}

	return transactions, nil
}

func (t TransactionModel) GetByReceiverWalletId(id string) ([]Transaction, error) {
	query := `
		SELECT id, sender_wallet_id, receiver_wallet_id, amount, created_at
		FROM transactions
		WHERE receiver_wallet_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := t.DB.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var transactions []Transaction

	for rows.Next() {
		var transaction Transaction
		err := rows.Scan(
			&transaction.ID,
			&transaction.SenderWalletId,
			&transaction.ReceiverWalletId,
			&transaction.Amount,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(transactions) == 0 {
		return nil, ErrRecordNotFound
	}

	return transactions, nil
}

func ValidateAmount(v *validator.Validator, amount float64) {
	v.Check(amount >= 0, "amount", "must be positive")
	v.Check(amount <= math.Pow(10, 10), "amount", "too large")
}

type MockTransactionModel struct{ mock.Mock }

func (t *MockTransactionModel) SendCoinTX(transaction *Transaction) error {
	args := t.Called(transaction)
	return args.Error(0)
}

func (t *MockTransactionModel) GetByReceiverWalletId(id string) ([]Transaction, error) {
	args := t.Called(id)
	transactions, ok := args.Get(0).([]Transaction)
	if !ok && args.Get(0) != nil {
		panic("expected []Transaction type for GetByReceiverWalletId")
	}
	return transactions, args.Error(1)
}

func (t *MockTransactionModel) GetBySenderWalletId(id string) ([]Transaction, error) {
	args := t.Called(id)
	transactions, ok := args.Get(0).([]Transaction)
	if !ok && args.Get(0) != nil {
		panic("expected []Transaction type for GetByReceiverWalletId")
	}
	return transactions, args.Error(1)
}
