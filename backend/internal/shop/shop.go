package shop

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/auth"
	"shopai/internal/httpx"
)

var ErrNoShop = errors.New("seller has no shop")

type Shop struct {
	ID          int64     `json:"id"`
	OwnerID     int64     `json:"owner_id"`
	Name        string    `json:"name"`
	LogoURL     string    `json:"logo_url"`
	Description string    `json:"description"`
	Contacts    string    `json:"contacts"`
	Status      string    `json:"status"` // pending | approved | rejected
	CreatedAt   time.Time `json:"created_at"`
}

const Cols = "id, owner_id, name, logo_url, description, contacts, status, created_at"

type scanner interface{ Scan(dest ...any) error }

func Scan(s scanner) (*Shop, error) {
	var sh Shop
	if err := s.Scan(&sh.ID, &sh.OwnerID, &sh.Name, &sh.LogoURL, &sh.Description, &sh.Contacts, &sh.Status, &sh.CreatedAt); err != nil {
		return nil, err
	}
	return &sh, nil
}

func ShopIDByOwner(ctx context.Context, pool *pgxpool.Pool, ownerID int64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, "SELECT id FROM shops WHERE owner_id = $1", ownerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNoShop
	}
	return id, err
}

// SellerShopID определяет магазин текущего селлера. При ошибке сам отвечает клиенту.
// Так обеспечивается правило ТЗ: селлер работает только со своим магазином.
func SellerShopID(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) (int64, bool) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return 0, false
	}
	id, err := ShopIDByOwner(r.Context(), pool, claims.UserID)
	if errors.Is(err, ErrNoShop) {
		httpx.Error(w, http.StatusConflict, "create a shop first")
		return 0, false
	}
	if err != nil {
		httpx.Internal(w, err)
		return 0, false
	}
	return id, true
}

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	var in struct {
		Name        string `json:"name"`
		LogoURL     string `json:"logo_url"`
		Description string `json:"description"`
		Contacts    string `json:"contacts"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	sh, err := Scan(h.pool.QueryRow(r.Context(),
		`INSERT INTO shops (owner_id, name, logo_url, description, contacts)
		 VALUES ($1, $2, $3, $4, $5) RETURNING `+Cols,
		claims.UserID, in.Name, in.LogoURL, in.Description, in.Contacts))
	if httpx.PgCode(err) == "23505" {
		httpx.Error(w, http.StatusConflict, "shop already exists")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, sh)
}

func (h *Handler) GetMine(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	sh, err := Scan(h.pool.QueryRow(r.Context(), "SELECT "+Cols+" FROM shops WHERE owner_id = $1", claims.UserID))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "shop not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sh)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	var in struct {
		Name        *string `json:"name"`
		LogoURL     *string `json:"logo_url"`
		Description *string `json:"description"`
		Contacts    *string `json:"contacts"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			httpx.Error(w, http.StatusBadRequest, "name must not be empty")
			return
		}
		in.Name = &n
	}
	sh, err := Scan(h.pool.QueryRow(r.Context(),
		`UPDATE shops SET
		   name        = COALESCE($2, name),
		   logo_url    = COALESCE($3, logo_url),
		   description = COALESCE($4, description),
		   contacts    = COALESCE($5, contacts)
		 WHERE owner_id = $1 RETURNING `+Cols,
		claims.UserID, in.Name, in.LogoURL, in.Description, in.Contacts))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusConflict, "create a shop first")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sh)
}

// Public — страница магазина для покупателей (только одобренные магазины).
func (h *Handler) Public(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	sh, err := Scan(h.pool.QueryRow(r.Context(), "SELECT "+Cols+" FROM shops WHERE id = $1 AND status = 'approved'", id))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "shop not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sh)
}
