package main

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps the Postgres connection pool and order persistence.
type Store struct{ pool *pgxpool.Pool }

// NewStore opens a pool and ensures the orders table exists.
func NewStore(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS orders (
		id BIGSERIAL PRIMARY KEY, item TEXT NOT NULL, qty INT NOT NULL DEFAULT 1,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now())`)
	return &Store{pool: pool}, err
}

// CreateOrder inserts an order and returns its id.
func (s *Store) CreateOrder(ctx context.Context, item string, qty int) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO orders(item, qty) VALUES($1,$2) RETURNING id`, item, qty).Scan(&id)
	return id, err
}

// GetOrder returns the item and qty for an order id.
func (s *Store) GetOrder(ctx context.Context, id int64) (string, int, error) {
	var item string
	var qty int
	err := s.pool.QueryRow(ctx, `SELECT item, qty FROM orders WHERE id=$1`, id).Scan(&item, &qty)
	return item, qty, err
}
