package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/httpx"
	"shopai/internal/shop"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

type Image struct {
	URL       string `json:"url"`
	SortOrder int    `json:"sort_order"`
}

type Attribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Product struct {
	ID          int64       `json:"id"`
	ShopID      int64       `json:"shop_id"`
	CategoryID  *int64      `json:"category_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Price       float64     `json:"price"`
	Stock       int         `json:"stock"`
	Status      string      `json:"status"` // pending | published | rejected
	IsHidden    bool        `json:"is_hidden"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	Images      []Image     `json:"images"`
	Attributes  []Attribute `json:"attributes"`
}

const productCols = "p.id, p.shop_id, p.category_id, p.name, p.description, p.price::float8, p.stock, p.status, p.is_hidden, p.created_at, p.updated_at"

type scanner interface{ Scan(dest ...any) error }

func scanProduct(s scanner) (*Product, error) {
	var p Product
	err := s.Scan(&p.ID, &p.ShopID, &p.CategoryID, &p.Name, &p.Description, &p.Price,
		&p.Stock, &p.Status, &p.IsHidden, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Images = []Image{}
	p.Attributes = []Attribute{}
	return &p, nil
}

// loadExtras подгружает фото и характеристики для списка товаров двумя запросами.
func (h *Handler) loadExtras(ctx context.Context, items []*Product) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]int64, len(items))
	idx := make(map[int64]*Product, len(items))
	for i, p := range items {
		ids[i] = p.ID
		idx[p.ID] = p
	}

	rows, err := h.pool.Query(ctx,
		`SELECT product_id, url, sort_order FROM product_images
		 WHERE product_id = ANY($1) ORDER BY product_id, sort_order, id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pid int64
		var im Image
		if err := rows.Scan(&pid, &im.URL, &im.SortOrder); err != nil {
			rows.Close()
			return err
		}
		idx[pid].Images = append(idx[pid].Images, im)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	rows, err = h.pool.Query(ctx,
		`SELECT product_id, name, value FROM product_attributes
		 WHERE product_id = ANY($1) ORDER BY product_id, id`, ids)
	if err != nil {
		return err
	}
	for rows.Next() {
		var pid int64
		var a Attribute
		if err := rows.Scan(&pid, &a.Name, &a.Value); err != nil {
			rows.Close()
			return err
		}
		idx[pid].Attributes = append(idx[pid].Attributes, a)
	}
	rows.Close()
	return rows.Err()
}

func (h *Handler) one(ctx context.Context, where string, args ...any) (*Product, error) {
	p, err := scanProduct(h.pool.QueryRow(ctx,
		"SELECT "+productCols+" FROM products p JOIN shops s ON s.id = p.shop_id WHERE "+where, args...))
	if err != nil {
		return nil, err
	}
	if err := h.loadExtras(ctx, []*Product{p}); err != nil {
		return nil, err
	}
	return p, nil
}

func (h *Handler) own(ctx context.Context, shopID, id int64) (*Product, error) {
	return h.one(ctx, "p.id = $1 AND p.shop_id = $2 AND p.deleted_at IS NULL", id, shopID)
}

// ---------- Селлер ----------

type productInput struct {
	CategoryID  *int64       `json:"category_id"`
	Name        *string      `json:"name"`
	Description *string      `json:"description"`
	Price       *float64     `json:"price"`
	Stock       *int         `json:"stock"`
	Images      *[]string    `json:"images"`     // nil = не менять
	Attributes  *[]Attribute `json:"attributes"` // nil = не менять
}

func (in *productInput) validate() string {
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return "name must not be empty"
		}
		in.Name = &n
	}
	if in.Price != nil && *in.Price < 0 {
		return "price must be >= 0"
	}
	if in.Stock != nil && *in.Stock < 0 {
		return "stock must be >= 0"
	}
	if in.Images != nil {
		if len(*in.Images) > 5 {
			return "at most 5 images per product"
		}
		for _, u := range *in.Images {
			if strings.TrimSpace(u) == "" {
				return "image url must not be empty"
			}
		}
	}
	if in.Attributes != nil {
		for _, a := range *in.Attributes {
			if strings.TrimSpace(a.Name) == "" {
				return "attribute name must not be empty"
			}
		}
	}
	return ""
}

func replaceExtras(ctx context.Context, tx pgx.Tx, id int64, in *productInput) error {
	if in.Images != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM product_images WHERE product_id = $1", id); err != nil {
			return err
		}
		for i, u := range *in.Images {
			if _, err := tx.Exec(ctx,
				"INSERT INTO product_images (product_id, url, sort_order) VALUES ($1, $2, $3)", id, u, i); err != nil {
				return err
			}
		}
	}
	if in.Attributes != nil {
		if _, err := tx.Exec(ctx, "DELETE FROM product_attributes WHERE product_id = $1", id); err != nil {
			return err
		}
		for _, a := range *in.Attributes {
			if _, err := tx.Exec(ctx,
				"INSERT INTO product_attributes (product_id, name, value) VALUES ($1, $2, $3)", id, a.Name, a.Value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	var in productInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Name == nil || in.Price == nil {
		httpx.Error(w, http.StatusBadRequest, "name and price are required")
		return
	}
	if msg := in.validate(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	stock := 0
	if in.Stock != nil {
		stock = *in.Stock
	}
	desc := ""
	if in.Description != nil {
		desc = *in.Description
	}

	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx,
		`INSERT INTO products (shop_id, category_id, name, description, price, stock)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		shopID, in.CategoryID, *in.Name, desc, *in.Price, stock).Scan(&id)
	if httpx.PgCode(err) == "23503" {
		httpx.Error(w, http.StatusBadRequest, "unknown category")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := replaceExtras(ctx, tx, id, &in); err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, err)
		return
	}
	p, err := h.own(ctx, shopID, id)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, p)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	limit, offset := httpx.Paging(r)
	args := []any{shopID}
	q := "SELECT " + productCols + " FROM products p WHERE p.shop_id = $1 AND p.deleted_at IS NULL"
	if st := r.URL.Query().Get("status"); st != "" {
		args = append(args, st)
		q += " AND p.status = $" + strconv.Itoa(len(args))
	}
	args = append(args, limit, offset)
	q += " ORDER BY p.id DESC LIMIT $" + strconv.Itoa(len(args)-1) + " OFFSET $" + strconv.Itoa(len(args))

	items, err := h.queryProducts(r.Context(), q, args...)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Handler) queryProducts(ctx context.Context, q string, args ...any) ([]*Product, error) {
	rows, err := h.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	items := []*Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, h.loadExtras(ctx, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	p, err := h.own(r.Context(), shopID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	var in productInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE products SET
		   category_id = COALESCE($3, category_id),
		   name        = COALESCE($4, name),
		   description = COALESCE($5, description),
		   price       = COALESCE($6, price),
		   stock       = COALESCE($7, stock),
		   updated_at  = now()
		 WHERE id = $1 AND shop_id = $2 AND deleted_at IS NULL`,
		id, shopID, in.CategoryID, in.Name, in.Description, in.Price, in.Stock)
	if httpx.PgCode(err) == "23503" {
		httpx.Error(w, http.StatusBadRequest, "unknown category")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err := replaceExtras(ctx, tx, id, &in); err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, err)
		return
	}
	p, err := h.own(ctx, shopID, id)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// Delete — «мягкое» удаление: заказы со ссылкой на товар остаются целыми.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	tag, err := h.pool.Exec(r.Context(),
		"UPDATE products SET deleted_at = now() WHERE id = $1 AND shop_id = $2 AND deleted_at IS NULL", id, shopID)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "product not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setHidden(hidden bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shopID, ok := shop.SellerShopID(w, r, h.pool)
		if !ok {
			return
		}
		id, ok := httpx.IDParam(w, r, "id")
		if !ok {
			return
		}
		tag, err := h.pool.Exec(r.Context(),
			"UPDATE products SET is_hidden = $3, updated_at = now() WHERE id = $1 AND shop_id = $2 AND deleted_at IS NULL",
			id, shopID, hidden)
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		if tag.RowsAffected() == 0 {
			httpx.Error(w, http.StatusNotFound, "product not found")
			return
		}
		p, err := h.own(r.Context(), shopID, id)
		if err != nil {
			httpx.Internal(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, p)
	}
}

func (h *Handler) Hide() http.HandlerFunc { return h.setHidden(true) }
func (h *Handler) Show() http.HandlerFunc { return h.setHidden(false) }

func (h *Handler) SetStock(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Stock *int `json:"stock"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Stock == nil || *in.Stock < 0 {
		httpx.Error(w, http.StatusBadRequest, "stock must be a number >= 0")
		return
	}
	tag, err := h.pool.Exec(r.Context(),
		"UPDATE products SET stock = $3, updated_at = now() WHERE id = $1 AND shop_id = $2 AND deleted_at IS NULL",
		id, shopID, *in.Stock)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "product not found")
		return
	}
	p, err := h.own(r.Context(), shopID, id)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

// ---------- Публичный каталог (покупатель без аккаунта) ----------

// Видимый товар: опубликован, не скрыт, не удалён, магазин одобрен.
const visible = "p.deleted_at IS NULL AND p.status = 'published' AND NOT p.is_hidden AND s.status = 'approved'"

func (h *Handler) PublicList(w http.ResponseWriter, r *http.Request) {
	limit, offset := httpx.Paging(r)
	qp := r.URL.Query()

	// Нулевой остаток = товар автоматически скрыт из каталога.
	where := []string{visible, "p.stock > 0"}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if q := strings.TrimSpace(qp.Get("q")); q != "" {
		add("(p.name ILIKE ? OR p.description ILIKE ?)", "%"+likeEscape(q)+"%")
	}
	if v := httpx.QueryInt(r, "category_id", 0); v > 0 {
		add("(p.category_id = ? OR p.category_id IN (SELECT id FROM categories WHERE parent_id = ?))", int64(v))
	}
	if v := httpx.QueryInt(r, "shop_id", 0); v > 0 {
		add("p.shop_id = ?", int64(v))
	}
	if v, ok := httpx.QueryFloat(r, "min_price"); ok {
		add("p.price >= ?", v)
	}
	if v, ok := httpx.QueryFloat(r, "max_price"); ok {
		add("p.price <= ?", v)
	}
	order := "p.id DESC"
	switch qp.Get("sort") {
	case "price_asc":
		order = "p.price ASC, p.id DESC"
	case "price_desc":
		order = "p.price DESC, p.id DESC"
	}
	cond := strings.Join(where, " AND ")

	var total int
	err := h.pool.QueryRow(r.Context(),
		"SELECT COUNT(*) FROM products p JOIN shops s ON s.id = p.shop_id WHERE "+cond, args...).Scan(&total)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	args = append(args, limit, offset)
	items, err := h.queryProducts(r.Context(),
		"SELECT "+productCols+" FROM products p JOIN shops s ON s.id = p.shop_id WHERE "+cond+
			" ORDER BY "+order+" LIMIT $"+strconv.Itoa(len(args)-1)+" OFFSET $"+strconv.Itoa(len(args)),
		args...)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (h *Handler) PublicGet(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	p, err := h.one(r.Context(), visible+" AND p.id = $1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, p)
}

func (h *Handler) Categories(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), "SELECT id, name, parent_id FROM categories ORDER BY id")
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer rows.Close()
	type cat struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		ParentID *int64 `json:"parent_id"`
	}
	out := []cat{}
	for rows.Next() {
		var c cat
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID); err != nil {
			httpx.Internal(w, err)
			return
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// likeEscape экранирует спецсимволы шаблона LIKE, чтобы «%» и «_» в поиске искались как обычные символы.
func likeEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
