package admin

import (
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/auth"
	"shopai/internal/httpx"
	"shopai/internal/shop"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

// ShopsByStatus: GET /api/admin/moderation/shops?status=pending
func (h *Handler) ShopsByStatus(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	limit, offset := httpx.Paging(r)
	rows, err := h.pool.Query(r.Context(),
		"SELECT "+shop.Cols+" FROM shops WHERE status = $1 ORDER BY id LIMIT $2 OFFSET $3", status, limit, offset)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer rows.Close()
	out := []*shop.Shop{}
	for rows.Next() {
		s, err := shop.Scan(rows)
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ProductsByStatus: GET /api/admin/moderation/products?status=pending
func (h *Handler) ProductsByStatus(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	limit, offset := httpx.Paging(r)
	rows, err := h.pool.Query(r.Context(),
		`SELECT id, shop_id, name, description, price::float8, status, created_at
		 FROM products WHERE status = $1 AND deleted_at IS NULL ORDER BY id LIMIT $2 OFFSET $3`, status, limit, offset)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer rows.Close()
	type item struct {
		ID          int64     `json:"id"`
		ShopID      int64     `json:"shop_id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Price       float64   `json:"price"`
		Status      string    `json:"status"`
		CreatedAt   time.Time `json:"created_at"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.ID, &it.ShopID, &it.Name, &it.Description, &it.Price, &it.Status, &it.CreatedAt); err != nil {
			httpx.Internal(w, err)
			return
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

type decisionInput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

// ModerateShop: POST /api/admin/shops/{id}/moderate {"decision":"approved|rejected","reason":"..."}
func (h *Handler) ModerateShop(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, "shop", []string{"approved", "rejected"}, "rejected")
}

// ModerateProduct: POST /api/admin/products/{id}/moderate {"decision":"published|rejected","reason":"..."}
func (h *Handler) ModerateProduct(w http.ResponseWriter, r *http.Request) {
	h.moderate(w, r, "product", []string{"published", "rejected"}, "rejected")
}

func (h *Handler) moderate(w http.ResponseWriter, r *http.Request, kind string, allowed []string, needReason string) {
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	var in decisionInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	valid := false
	for _, a := range allowed {
		if in.Decision == a {
			valid = true
		}
	}
	if !valid {
		httpx.Error(w, http.StatusBadRequest, "decision must be one of: "+strings.Join(allowed, ", "))
		return
	}
	if in.Decision == needReason && in.Reason == "" {
		httpx.Error(w, http.StatusBadRequest, "reason is required when rejecting")
		return
	}
	claims := auth.FromContext(r.Context())
	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer tx.Rollback(ctx)

	var affected int64
	if kind == "shop" {
		tag, err := tx.Exec(ctx, "UPDATE shops SET status = $2 WHERE id = $1", id, in.Decision)
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		affected = tag.RowsAffected()
		if affected > 0 {
			_, err = tx.Exec(ctx,
				"INSERT INTO moderation_records (shop_id, moderator_id, decision, reason) VALUES ($1, $2, $3, $4)",
				id, claims.UserID, in.Decision, in.Reason)
		}
		if err != nil {
			httpx.Internal(w, err)
			return
		}
	} else {
		tag, err := tx.Exec(ctx,
			"UPDATE products SET status = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL", id, in.Decision)
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		affected = tag.RowsAffected()
		if affected > 0 {
			_, err = tx.Exec(ctx,
				"INSERT INTO moderation_records (product_id, moderator_id, decision, reason) VALUES ($1, $2, $3, $4)",
				id, claims.UserID, in.Decision, in.Reason)
		}
		if err != nil {
			httpx.Internal(w, err)
			return
		}
	}
	if affected == 0 {
		httpx.Error(w, http.StatusNotFound, kind+" not found")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "decision": in.Decision})
}

// CreateCategory: POST /api/admin/categories {"name":"...","parent_id":null}
func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	var id int64
	err := h.pool.QueryRow(r.Context(),
		"INSERT INTO categories (name, parent_id) VALUES ($1, $2) RETURNING id", in.Name, in.ParentID).Scan(&id)
	if httpx.PgCode(err) == "23503" {
		httpx.Error(w, http.StatusBadRequest, "unknown parent_id")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "name": in.Name, "parent_id": in.ParentID})
}
