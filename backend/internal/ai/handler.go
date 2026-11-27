package ai

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/auth"
	"shopai/internal/httpx"
	"shopai/internal/shop"
)

type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type priceStats struct {
	Count     int     `json:"similar_count"`
	Min       float64 `json:"min"`
	Avg       float64 `json:"avg"`
	Max       float64 `json:"max"`
	Suggested float64 `json:"suggested_price"`
}

type Handler struct {
	pool *pgxpool.Pool
	llm  Client
}

func NewHandler(pool *pgxpool.Pool, llm Client) *Handler { return &Handler{pool: pool, llm: llm} }

func (h *Handler) categoryName(ctx context.Context, id *int64) string {
	if id == nil {
		return ""
	}
	var n string
	if err := h.pool.QueryRow(ctx, "SELECT name FROM categories WHERE id = $1", *id).Scan(&n); err != nil {
		return ""
	}
	return n
}

// fillFromProduct подтягивает данные собственного товара селлера.
func (h *Handler) fillFromProduct(ctx context.Context, shopID, productID int64) (name, desc string, catID *int64, attrs []Attribute, err error) {
	err = h.pool.QueryRow(ctx,
		"SELECT name, description, category_id FROM products WHERE id = $1 AND shop_id = $2 AND deleted_at IS NULL",
		productID, shopID).Scan(&name, &desc, &catID)
	if err != nil {
		return
	}
	rows, qerr := h.pool.Query(ctx, "SELECT name, value FROM product_attributes WHERE product_id = $1 ORDER BY id", productID)
	if qerr != nil {
		err = qerr
		return
	}
	defer rows.Close()
	for rows.Next() {
		var a Attribute
		if err = rows.Scan(&a.Name, &a.Value); err != nil {
			return
		}
		attrs = append(attrs, a)
	}
	err = rows.Err()
	return
}

func (h *Handler) logRequest(ctx context.Context, userID int64, productID *int64, typ, status string, started time.Time) {
	_, _ = h.pool.Exec(ctx,
		"INSERT INTO ai_requests (user_id, product_id, type, status, duration_ms) VALUES ($1, $2, $3, $4, $5)",
		userID, productID, typ, status, int(time.Since(started).Milliseconds()))
}

func (h *Handler) complete(ctx context.Context, system, user, fallback string) (string, error) {
	if h.llm.Name() == "mock" {
		return fallback, nil
	}
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return h.llm.Complete(cctx, system, user)
}

// Description: POST /api/ai/description
func (h *Handler) Description(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	claims := auth.FromContext(r.Context())
	var in struct {
		ProductID  *int64      `json:"product_id"`
		Name       string      `json:"name"`
		CategoryID *int64      `json:"category_id"`
		Attributes []Attribute `json:"attributes"`
		Keywords   []string    `json:"keywords"`
		Tone       string      `json:"tone"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	started := time.Now()
	if in.ProductID != nil {
		name, _, cat, attrs, err := h.fillFromProduct(ctx, shopID, *in.ProductID)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		if strings.TrimSpace(in.Name) == "" {
			in.Name = name
		}
		if in.CategoryID == nil {
			in.CategoryID = cat
		}
		if len(in.Attributes) == 0 {
			in.Attributes = attrs
		}
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "name or product_id is required")
		return
	}
	if strings.TrimSpace(in.Tone) == "" {
		in.Tone = "дружелюбный"
	}
	cat := h.categoryName(ctx, in.CategoryID)
	text, err := h.complete(ctx, DescriptionSystem,
		descriptionUser(in.Name, cat, in.Tone, in.Attributes, in.Keywords),
		fallbackDescription(in.Name, cat, in.Attributes))
	status := "success"
	if err != nil {
		status = "error"
	}
	h.logRequest(ctx, claims.UserID, in.ProductID, "description", status, started)
	if err != nil {
		log.Printf("ai description: provider error: %v", err)
		httpx.Error(w, http.StatusBadGateway, "AI provider error")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"description": text, "provider": h.llm.Name()})
}

// Price: POST /api/ai/price. Цифры считаются по БД, ИИ только объясняет.
func (h *Handler) Price(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	claims := auth.FromContext(r.Context())
	var in struct {
		ProductID   *int64 `json:"product_id"`
		Name        string `json:"name"`
		CategoryID  *int64 `json:"category_id"`
		Description string `json:"description"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	started := time.Now()
	if in.ProductID != nil {
		name, desc, cat, _, err := h.fillFromProduct(ctx, shopID, *in.ProductID)
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "product not found")
			return
		}
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		if strings.TrimSpace(in.Name) == "" {
			in.Name = name
		}
		if in.CategoryID == nil {
			in.CategoryID = cat
		}
		if in.Description == "" {
			in.Description = desc
		}
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || in.CategoryID == nil {
		httpx.Error(w, http.StatusBadRequest, "name and category_id (or product_id) are required")
		return
	}

	var st priceStats
	err := h.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(MIN(price), 0)::float8, COALESCE(AVG(price), 0)::float8, COALESCE(MAX(price), 0)::float8
		 FROM products
		 WHERE category_id = $1 AND status = 'published' AND deleted_at IS NULL AND NOT is_hidden
		   AND ($2::bigint IS NULL OR id <> $2::bigint)`, in.CategoryID, in.ProductID).
		Scan(&st.Count, &st.Min, &st.Avg, &st.Max)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	st.Avg = math.Round(st.Avg*100) / 100
	if st.Count > 0 {
		st.Suggested = st.Avg
	}
	if st.Count == 0 {
		h.logRequest(ctx, claims.UserID, in.ProductID, "price", "success", started)
		httpx.JSON(w, http.StatusOK, map[string]any{
			"similar_count": 0, "min": 0, "avg": 0, "max": 0, "suggested_price": 0,
			"explanation": "На площадке пока нет похожих товаров в этой категории, поэтому подсказать цену нельзя.",
			"provider":    h.llm.Name(),
		})
		return
	}
	cat := h.categoryName(ctx, in.CategoryID)
	text, err := h.complete(ctx, PriceSystem, priceUser(in.Name, cat, in.Description, st), fallbackPrice(st))
	status := "success"
	if err != nil {
		status = "error"
	}
	h.logRequest(ctx, claims.UserID, in.ProductID, "price", status, started)
	if err != nil {
		log.Printf("ai price: provider error: %v", err)
		httpx.Error(w, http.StatusBadGateway, "AI provider error")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"similar_count": st.Count, "min": st.Min, "avg": st.Avg, "max": st.Max,
		"suggested_price": st.Suggested, "explanation": text, "provider": h.llm.Name(),
	})
}
