package payment

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/httpx"
)

// Handler реализует оплату через платёжный виджет CloudPayments:
//  1. Intent — отдаёт фронтенду параметры для widget.start(...) (сумма берётся из БД, а не от клиента);
//  2. Check / Pay / Fail — уведомления (webhooks) от CloudPayments, подпись проверяется по API Secret.
type Handler struct {
	pool     *pgxpool.Pool
	publicID string
	secret   string
}

func NewHandler(pool *pgxpool.Pool, publicID, secret string) *Handler {
	return &Handler{pool: pool, publicID: publicID, secret: secret}
}

func (h *Handler) enabled() bool { return h.publicID != "" && h.secret != "" }

// Коды ответа на уведомления CloudPayments.
const (
	codeOK            = 0
	codeInvoiceNotFnd = 10
	codeBadAmount     = 12
	codeRejected      = 13
)

func reply(w http.ResponseWriter, code int) {
	httpx.JSON(w, http.StatusOK, map[string]int{"code": code})
}

// ---------- 1. Параметры для виджета ----------

// Intent: GET /api/orders/{number}/payment-intent?email=...
// Возвращает объект, который фронтенд передаёт в `new cp.CloudPayments().start(...)`.
func (h *Handler) Intent(w http.ResponseWriter, r *http.Request) {
	if !h.enabled() {
		httpx.Error(w, http.StatusServiceUnavailable, "online payment is not configured")
		return
	}
	number := chi.URLParam(r, "number")
	email := r.URL.Query().Get("email")

	var total float64
	var status, payStatus, name string
	err := h.pool.QueryRow(r.Context(),
		`SELECT total_amount::float8, status, payment_status, customer_name FROM orders
		 WHERE number = $1 AND lower(customer_email) = lower($2)`, number, email).
		Scan(&total, &status, &payStatus, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "order not found")
		return
	}
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if status == "cancelled" {
		httpx.Error(w, http.StatusConflict, "order is cancelled")
		return
	}
	if payStatus == "paid" {
		httpx.Error(w, http.StatusConflict, "order is already paid")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"publicTerminalId": h.publicID,
		"paymentSchema":    "Single", // одностадийная оплата: деньги списываются сразу
		"amount":           total,
		"currency":         "RUB",
		"externalId":       number, // вернётся в уведомлениях и по нему мы найдём заказ
		"description":      "Оплата заказа " + number,
		"receiptEmail":     email,
		"userInfo":         map[string]string{"email": email, "fullName": name},
	})
}

// ---------- 2. Уведомления ----------

// verified читает тело, проверяет подпись (HMAC-SHA256 от сырого тела, ключ — API Secret,
// заголовок X-Content-HMAC или Content-HMAC, значение в Base64) и разбирает поля.
func (h *Handler) verified(w http.ResponseWriter, r *http.Request) (map[string]string, bool) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil || len(body) == 0 || !h.enabled() {
		reply(w, codeRejected)
		return nil, false
	}
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	got := r.Header.Get("X-Content-HMAC")
	if got == "" {
		got = r.Header.Get("Content-HMAC")
	}
	if !hmac.Equal([]byte(got), []byte(want)) {
		log.Printf("cloudpayments: bad signature from %s", r.RemoteAddr)
		reply(w, codeRejected)
		return nil, false
	}
	fields, err := parseBody(r.Header.Get("Content-Type"), body)
	if err != nil {
		log.Printf("cloudpayments: cannot parse body: %v", err)
		reply(w, codeRejected)
		return nil, false
	}
	return fields, true
}

// parseBody понимает оба формата уведомлений: form-urlencoded (по умолчанию) и JSON.
func parseBody(contentType string, body []byte) (map[string]string, error) {
	out := map[string]string{}
	if strings.Contains(strings.ToLower(contentType), "json") {
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		for k, v := range raw {
			switch t := v.(type) {
			case string:
				out[k] = t
			case json.Number:
				out[k] = t.String()
			case bool:
				out[k] = strconv.FormatBool(t)
			}
		}
		return out, nil
	}
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}
	for k, v := range vals {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return out, nil
}

func first(m map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := m[k]; v != "" {
			return v
		}
	}
	return ""
}

// orderNumber: номер заказа, который мы передали в виджет как externalId
// (в уведомлениях поле называется InvoiceId; для новой схемы допускаем ExternalId).
func orderNumber(m map[string]string) string {
	return first(m, "InvoiceId", "ExternalId", "externalId", "invoiceId")
}

func sameAmount(raw string, total float64) bool {
	a, err := strconv.ParseFloat(strings.ReplaceAll(raw, ",", "."), 64)
	if err != nil {
		return false
	}
	return math.Round(a*100) == math.Round(total*100)
}

type orderRow struct {
	id        int64
	total     float64
	status    string
	payStatus string
}

func (h *Handler) findOrder(r *http.Request, number string) (*orderRow, error) {
	var o orderRow
	err := h.pool.QueryRow(r.Context(),
		"SELECT id, total_amount::float8, status, payment_status FROM orders WHERE number = $1", number).
		Scan(&o.id, &o.total, &o.status, &o.payStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &o, err
}

// Check: до списания проверяем, что заказ существует, не оплачен, не отменён и сумма совпадает.
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	f, ok := h.verified(w, r)
	if !ok {
		return
	}
	number := orderNumber(f)
	o, err := h.findOrder(r, number)
	if err != nil {
		log.Printf("cloudpayments check: db error: %v", err)
		reply(w, codeRejected)
		return
	}
	switch {
	case o == nil:
		log.Printf("cloudpayments check: order %q not found (fields: %v)", number, keys(f))
		reply(w, codeInvoiceNotFnd)
	case !sameAmount(f["Amount"], o.total):
		reply(w, codeBadAmount)
	case o.status == "cancelled" || o.payStatus == "paid":
		reply(w, codeRejected)
	default:
		reply(w, codeOK)
	}
}

// Pay: платёж прошёл. Операция идемпотентна: повторное уведомление с тем же TransactionId ничего не меняет.
func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	f, ok := h.verified(w, r)
	if !ok {
		return
	}
	number := orderNumber(f)
	txID := f["TransactionId"]
	o, err := h.findOrder(r, number)
	if err != nil {
		log.Printf("cloudpayments pay: db error: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal error") // CloudPayments повторит уведомление
		return
	}
	if o == nil || txID == "" {
		log.Printf("cloudpayments pay: order %q not found or no TransactionId (fields: %v)", number, keys(f))
		reply(w, codeOK)
		return
	}
	if st := f["Status"]; st != "" && st != "Completed" {
		log.Printf("cloudpayments pay: order %s status %q ignored", number, st)
		reply(w, codeOK)
		return
	}
	amount, _ := strconv.ParseFloat(strings.ReplaceAll(f["Amount"], ",", "."), 64)
	if !sameAmount(f["Amount"], o.total) {
		log.Printf("cloudpayments pay: AMOUNT MISMATCH order %s: paid %.2f, expected %.2f (tx %s): нужна ручная проверка", number, amount, o.total, txID)
		reply(w, codeOK)
		return
	}

	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`INSERT INTO payments (order_id, amount, status, provider, provider_tx_id)
		 VALUES ($1, $2, 'success', 'cloudpayments', $3)
		 ON CONFLICT (provider_tx_id) DO NOTHING`, o.id, amount, txID)
	if err != nil {
		log.Printf("cloudpayments pay: insert payment: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if tag.RowsAffected() == 0 { // уже обработано
		reply(w, codeOK)
		return
	}
	if _, err := tx.Exec(ctx, "UPDATE orders SET payment_status = 'paid' WHERE id = $1", o.id); err != nil {
		log.Printf("cloudpayments pay: update order: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if o.status == "cancelled" {
		log.Printf("cloudpayments pay: order %s was cancelled but paid (tx %s): нужен возврат", number, txID)
	}
	reply(w, codeOK)
}

// Fail: платёж отклонён. Записываем неудачную попытку, оплаченный заказ не трогаем.
func (h *Handler) Fail(w http.ResponseWriter, r *http.Request) {
	f, ok := h.verified(w, r)
	if !ok {
		return
	}
	number := orderNumber(f)
	o, err := h.findOrder(r, number)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if o == nil || o.payStatus == "paid" {
		reply(w, codeOK)
		return
	}
	amount, _ := strconv.ParseFloat(strings.ReplaceAll(f["Amount"], ",", "."), 64)
	var txID *string
	if v := f["TransactionId"]; v != "" {
		txID = &v
	}
	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO payments (order_id, amount, status, provider, provider_tx_id)
		 VALUES ($1, $2, 'failed', 'cloudpayments', $3)
		 ON CONFLICT (provider_tx_id) DO NOTHING`, o.id, amount, txID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if _, err := tx.Exec(ctx,
		"UPDATE orders SET payment_status = 'failed' WHERE id = $1 AND payment_status <> 'paid'", o.id); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	reply(w, codeOK)
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
