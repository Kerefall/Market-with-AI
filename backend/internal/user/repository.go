package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/httpx"
)

var (
	ErrEmailTaken = errors.New("email already registered")
	ErrNotFound   = errors.New("user not found")
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

const selectUser = `SELECT u.id, u.email, u.password_hash, r.name, u.created_at
	FROM users u JOIN roles r ON r.id = u.role_id `

type scanner interface{ Scan(dest ...any) error }

func scan(s scanner) (*User, error) {
	var u User
	if err := s.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) Create(ctx context.Context, email, hash, role string) (*User, error) {
	u := &User{Email: email, Role: role}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role_id)
		 VALUES ($1, $2, (SELECT id FROM roles WHERE name = $3))
		 RETURNING id, created_at`, email, hash, role,
	).Scan(&u.ID, &u.CreatedAt)
	if httpx.PgCode(err) == "23505" {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	u, err := scan(r.pool.QueryRow(ctx, selectUser+"WHERE u.email = $1", email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
	u, err := scan(r.pool.QueryRow(ctx, selectUser+"WHERE u.id = $1", id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return u, err
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]User, error) {
	rows, err := r.pool.Query(ctx, selectUser+"ORDER BY u.id LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		u, err := scan(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}
