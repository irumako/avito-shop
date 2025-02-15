package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Order struct {
	ID           string    `json:"id"`
	WalletId     string    `json:"wallet_id"`
	ProductId    string    `json:"merch_id"`
	PurchaseCost float64   `json:"purchase_cost"`
	PurchasedAt  time.Time `json:"purchased_at"`
}

type OrderModel struct {
	DB *sql.DB
}

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

func (o OrderModel) BuyProductTX(order *Order) error {
	orderQuery := `
		INSERT INTO merch_orders (wallet_id, merch_id, purchase_cost)
		VALUES ($1, $2, $3)
		RETURNING id, purchased_at
	`
	walletQuery := `
		UPDATE wallets
		SET balance = balance - $1
		WHERE id = $2 AND balance >= $1
		RETURNING updated_at
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := o.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func(tx *sql.Tx) {
		err := tx.Rollback()
		if err != nil {

		}
	}(tx)

	err = tx.QueryRowContext(ctx, orderQuery, order.WalletId, order.ProductId, order.PurchaseCost).
		Scan(&order.ID, &order.PurchasedAt)
	if err != nil {
		return err
	}

	var updatedAt time.Time
	err = tx.QueryRowContext(ctx, walletQuery, order.PurchaseCost, order.WalletId).Scan(&updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInsufficientFunds
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (o OrderModel) GetByWalletId(id string) ([]Order, error) {
	query := `
		SELECT id, wallet_id, merch_id, purchase_cost, purchased_at
		FROM merch_orders
		WHERE wallet_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := o.DB.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var orders []Order

	for rows.Next() {
		var order Order
		err := rows.Scan(
			&order.ID,
			&order.WalletId,
			&order.ProductId,
			&order.PurchaseCost,
			&order.PurchasedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(orders) == 0 {
		return nil, ErrRecordNotFound
	}

	return orders, nil
}
