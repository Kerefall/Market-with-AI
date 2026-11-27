package order

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/auth"
	"shopai/internal/httpx"
	"shopai/internal/shop"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

type Item struct {
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
}

type HistoryEntry struct {
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changed_at"`
}

type Order struct {
	ID              int64          `json:"id"`
	ShopID          int64          `json:"shop_id"`
	Number          string         `json:"number"`
	CustomerName    string         `json:"customer_name"`
	CustomerPhone   string         `json:"customer_phone"`
	CustomerEmail   string         `json:"customer_email"`
	DeliveryAddress string         `json:"delivery_address"`
	TotalAmount     float64        `json:"total_amount"`
	Status          string         `json:"status"`         // new | assembling | shipped | delivered | cancelled
	PaymentStatus   string         `json:"payment_status"` // unpaid | paid | failed
	CreatedAt       time.Time      `json:"created_at"`
	Items           []Item         `json:"items,omitempty"`
	History         []HistoryEntry `json:"history,omitempty"`
}

const cols = "id, shop_id, number, customer_name, customer_phone, customer_email, delivery_address, total_amount::float8, status, payment_status, created_at"

type scanner interface{ Scan(dest ...any) error }

func scanOrder(s scanner) (*Order, error) {
	var o Order
	err := s.Scan(&o.ID, &o.ShopID, &o.Number, &o.CustomerName, &o.CustomerPhone, &o.CustomerEmail,
		&o.DeliveryAddress, &o.TotalAmount, &o.Status, &o.PaymentStatus, &o.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

type apiErr struct {
	Status int
	Msg    string
}

func (e *apiErr) Error() string { return e.Msg }

func (h *Handler) loadDetails(ctx context.Context, o *Order) error {
	o.Items = []Item{}
	o.History = []HistoryEntry{}
	rows, err := h.pool.Query(ctx,
		`SELECT product_id, product_name, quantity, unit_price::float8 FROM order_items WHERE order_id = $1 ORDER BY id`, o.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ProductID, &it.ProductName, &it.Quantity, &it.UnitPrice); err != nil {
			rows.Close()
			return err
		}
		o.Items = append(o.Items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	rows, err = h.pool.Query(ctx,
		`SELECT status, changed_at FROM order_status_history WHERE order_id = $1 ORDER BY id`, o.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.Status, &e.ChangedAt); err != nil {
			rows.Close()
			return err
		}
		o.History = append(o.History, e)
	}
	rows.Close()
	return rows.Err()
}

func newNumber() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("060102"), strings.ToUpper(hex.EncodeToString(b)))
}

// ---------- Покупатель (без аккаунта) ----------

type createInput struct {
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	DeliveryAddress string `json:"delivery_address"`
	Items           []struct {
		ProductID int64 `json:"product_id"`
		Quantity  int   `json:"quantity"`
	} `json:"items"`
}

func (in *createInput) validate() string {
	in.CustomerName = strings.TrimSpace(in.CustomerName)
	in.CustomerPhone = strings.TrimSpace(in.CustomerPhone)
	in.CustomerEmail = strings.TrimSpace(in.CustomerEmail)
	in.DeliveryAddress = strings.TrimSpace(in.DeliveryAddress)
	if in.CustomerName == "" || in.CustomerPhone == "" || in.DeliveryAddress == "" {
		return "customer_name, customer_phone and delivery_address are required"
	}
	if _, err := mail.ParseAddress(in.CustomerEmail); err != nil {
		return "bad customer_email"
	}
	if len(in.Items) == 0 || len(in.Items) > 50 {
		return "items must contain 1..50 positions"
	}
	return ""
}

type line struct {
	productID int64
	shopID    int64
	name      string
	price     float64
	stock     int
	qty       int
}

// create: корзина с товарами нескольких магазинов разбивается на отдельные заказы по магазинам.
// Остатки списываются в той же транзакции, строки товаров блокируются (FOR UPDATE).
func (h *Handler) create(ctx context.Context, in createInput) ([]*Order, error) {
	qty := map[int64]int{}
	for _, it := range in.Items {
		if it.ProductID <= 0 || it.Quantity <= 0 || it.Quantity > 1000 {
			return nil, &apiErr{http.StatusBadRequest, "invalid item"}
		}
		qty[it.ProductID] += it.Quantity
	}
	ids := make([]int64, 0, len(qty))
	for id := range qty {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`SELECT p.id, p.shop_id, p.name, p.price::float8, p.stock
		 FROM products p JOIN shops s ON s.id = p.shop_id
		 WHERE p.id = ANY($1) AND p.deleted_at IS NULL AND p.status = 'published'
		   AND NOT p.is_hidden AND s.status = 'approved'
		 ORDER BY p.id FOR UPDATE OF p`, ids)
	if err != nil {
		return nil, err
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.productID, &l.shopID, &l.name, &l.price, &l.stock); err != nil {
			rows.Close()
			return nil, err
		}
		l.qty = qty[l.productID]
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(lines) != len(ids) {
		return nil, &apiErr{http.StatusBadRequest, "some products are unavailable"}
	}
	for _, l := range lines {
		if l.stock < l.qty {
			return nil, &apiErr{http.StatusConflict, fmt.Sprintf("not enough stock for %q (available: %d)", l.name, l.stock)}
		}
	}

	byShop := map[int64][]line{}
	var shopIDs []int64
	for _, l := range lines {
		if _, ok := byShop[l.shopID]; !ok {
			shopIDs = append(shopIDs, l.shopID)
		}
		byShop[l.shopID] = append(byShop[l.shopID], l)
	}
	sort.Slice(shopIDs, func(i, j int) bool { return shopIDs[i] < shopIDs[j] })

	var result []*Order
	for _, sid := range shopIDs {
		total := 0.0
		for _, l := range byShop[sid] {
			total += l.price * float64(l.qty)
		}
		o, err := scanOrder(tx.QueryRow(ctx,
			`INSERT INTO orders (shop_id, number, customer_name, customer_phone, customer_email, delivery_address, total_amount)
			 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+cols,
			sid, newNumber(), in.CustomerName, in.CustomerPhone, in.CustomerEmail, in.DeliveryAddress, total))
		if err != nil {
			return nil, err
		}
		o.Items = []Item{}
		for _, l := range byShop[sid] {
			if _, err := tx.Exec(ctx,
				`INSERT INTO order_items (order_id, product_id, product_name, quantity, unit_price)
				 VALUES ($1, $2, $3, $4, $5)`, o.ID, l.productID, l.name, l.qty, l.price); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(ctx,
				"UPDATE products SET stock = stock - $1, updated_at = now() WHERE id = $2", l.qty, l.productID); err != nil {
				return nil, err
			}
			o.Items = append(o.Items, Item{ProductID: l.productID, ProductName: l.name, Quantity: l.qty, UnitPrice: l.price})
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1, 'new', NULL)", o.ID); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in createInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	orders, err := h.create(r.Context(), in)
	if err != nil {
		var ae *apiErr
		if errors.As(err, &ae) {
			httpx.Error(w, ae.Status, ae.Msg)
			return
		}
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"orders": orders})
}

// Track — статус заказа по номеру и email покупателя.
func (h *Handler) Track(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	email := r.URL.Query().Get("email")
	o, err := scanOrder(h.pool.QueryRow(r.Context(),
		"SELECT "+cols+" FROM orders WHERE number = $1 AND lower(customer_email) = lower($2)", number, email))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := h.loadDetails(r.Context(), o); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// Pay — тестовая оплата. Тело: {"email": "...", "simulate": "success" | "failure"}.
func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	var in struct {
		Email    string `json:"email"`
		Simulate string `json:"simulate"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.Simulate != "" && in.Simulate != "success" && in.Simulate != "failure" {
		httpx.Error(w, http.StatusBadRequest, `simulate must be "success" or "failure"`)
		return
	}
	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	defer tx.Rollback(ctx)

	var id int64
	var total float64
	var ordStatus, payStatus string
	err = tx.QueryRow(ctx,
		`SELECT id, total_amount::float8, status, payment_status FROM orders
		 WHERE number = $1 AND lower(customer_email) = lower($2) FOR UPDATE`, number, in.Email).
		Scan(&id, &total, &ordStatus, &payStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if ordStatus == "cancelled" {
		httpx.Error(w, http.StatusConflict, "order is cancelled")
		return
	}
	if payStatus == "paid" {
		httpx.Error(w, http.StatusConflict, "order is already paid")
		return
	}
	paymentStatus, newPayStatus := "success", "paid"
	if in.Simulate == "failure" {
		paymentStatus, newPayStatus = "failed", "failed"
	}
	var payID int64
	if err := tx.QueryRow(ctx,
		"INSERT INTO payments (order_id, amount, status) VALUES ($1, $2, $3) RETURNING id",
		id, total, paymentStatus).Scan(&payID); err != nil {
		httpx.Internal(w, err)
		return
	}
	if _, err := tx.Exec(ctx, "UPDATE orders SET payment_status = $2 WHERE id = $1", id, newPayStatus); err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"payment_id": payID, "amount": total, "payment_status": newPayStatus, "result": paymentStatus,
	})
}

// ---------- Селлер ----------

func (h *Handler) SellerList(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	limit, offset := httpx.Paging(r)
	args := []any{shopID}
	q := "SELECT " + cols + " FROM orders WHERE shop_id = $1"
	if st := r.URL.Query().Get("status"); st != "" {
		args = append(args, st)
		q += " AND status = $" + strconv.Itoa(len(args))
	}
	args = append(args, limit, offset)
	q += " ORDER BY id DESC LIMIT $" + strconv.Itoa(len(args)-1) + " OFFSET $" + strconv.Itoa(len(args))

	rows, err := h.pool.Query(r.Context(), q, args...)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	items := []*Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			httpx.Internal(w, err)
			return
		}
		items = append(items, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}

func (h *Handler) SellerGet(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	o, err := scanOrder(h.pool.QueryRow(r.Context(),
		"SELECT "+cols+" FROM orders WHERE id = $1 AND shop_id = $2", id, shopID))
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := h.loadDetails(r.Context(), o); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

var transitions = map[string][]string{
	"new":        {"assembling", "cancelled"},
	"assembling": {"shipped", "cancelled"},
	"shipped":    {"delivered"},
}

func allowed(from, to string) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// ChangeStatus: new → assembling → shipped → delivered; отмена из new/assembling возвращает остатки.
func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	id, ok := httpx.IDParam(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
	}
	if !httpx.Decode(w, r, &in) {
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

	var cur string
	err = tx.QueryRow(ctx, "SELECT status FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE", id, shopID).Scan(&cur)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if !allowed(cur, in.Status) {
		httpx.Error(w, http.StatusConflict, fmt.Sprintf("cannot change status from %q to %q", cur, in.Status))
		return
	}
	if _, err := tx.Exec(ctx, "UPDATE orders SET status = $2 WHERE id = $1", id, in.Status); err != nil {
		httpx.Internal(w, err)
		return
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1, $2, $3)",
		id, in.Status, claims.UserID); err != nil {
		httpx.Internal(w, err)
		return
	}
	if in.Status == "cancelled" {
		if _, err := tx.Exec(ctx,
			`UPDATE products p SET stock = p.stock + oi.quantity, updated_at = now()
			 FROM order_items oi WHERE oi.order_id = $1 AND p.id = oi.product_id`, id); err != nil {
			httpx.Internal(w, err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Internal(w, err)
		return
	}
	o, err := scanOrder(h.pool.QueryRow(ctx, "SELECT "+cols+" FROM orders WHERE id = $1", id))
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if err := h.loadDetails(ctx, o); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}
