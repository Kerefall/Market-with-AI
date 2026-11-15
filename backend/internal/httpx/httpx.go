package httpx

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

func Internal(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	Error(w, http.StatusInternalServerError, "internal error")
}

// Decode читает JSON из тела запроса (не больше 1 МБ). При ошибке сам отвечает 400.
func Decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		Error(w, http.StatusBadRequest, "invalid json")
		return false
	}
	return true
}

func IDParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		Error(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func QueryInt(r *http.Request, key string, def int) int {
	v, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return v
}

func QueryFloat(r *http.Request, key string) (float64, bool) {
	v, err := strconv.ParseFloat(r.URL.Query().Get(key), 64)
	return v, err == nil
}

// Paging возвращает limit (1..100, по умолчанию 20) и offset (>= 0).
func Paging(r *http.Request) (int, int) {
	limit := QueryInt(r, "limit", 20)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := QueryInt(r, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// PgCode возвращает SQLSTATE ошибки PostgreSQL (23505 — уникальность, 23503 — внешний ключ).
func PgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}
