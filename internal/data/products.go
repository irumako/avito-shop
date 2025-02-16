package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Product struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Cost float64 `json:"cost"`
}

type ProductModel struct {
	DB *sql.DB
}

func (p ProductModel) Get(id string) (*Product, error) {
	query := `
			SELECT id, name, cost
			FROM merch
			WHERE id = $1
			`
	var product Product
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := p.DB.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Cost,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &product, nil
}

func (p ProductModel) GetByName(name string) (*Product, error) {
	query := `
			SELECT id, name, cost
			FROM merch
			WHERE name = $1
			`
	var product Product
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := p.DB.QueryRowContext(ctx, query, name).Scan(
		&product.ID,
		&product.Name,
		&product.Cost,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &product, nil
}
