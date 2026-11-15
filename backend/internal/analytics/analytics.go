package analytics

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/httpx"
	"shopai/internal/shop"
)

type Handler struct{ pool *pgxpool.Pool }

func NewHandler(pool *pgxpool.Pool) *Handler { return &Handler{pool: pool} }

type TopProduct struct {
	ProductID int64   `json:"product_id"`
	Name      string  `json:"name"`
	Sold      int     `json:"sold"`
	Revenue   float64 `json:"revenue"`
}

type DayStat struct {
	Date    string  `json:"date"`
	Orders  int     `json:"orders"`
	Revenue float64 `json:"revenue"`
}

type Result struct {
	Days        int          `json:"days"`
	Revenue     float64      `json:"revenue"`
	OrdersCount int          `json:"orders_count"`
	AvgCheck    float64      `json:"avg_check"`
	LowStock    int          `json:"low_stock_products"`
	TopProducts []TopProduct `json:"top_products"`
	ByDay       []DayStat    `json:"by_day"`
}

// Get: GET /api/seller/analytics?days=30. Отменённые заказы в выручку и счётчики не входят.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	shopID, ok := shop.SellerShopID(w, r, h.pool)
	if !ok {
		return
	}
	days := httpx.QueryInt(r, "days", 30)
	if days < 1 || days > 365 {
		days = 30
	}
	ctx := r.Context()
	res := Result{Days: days, TopProducts: []TopProduct{}, ByDay: []DayStat{}}

	err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_amount), 0)::float8, COUNT(*) FROM orders
		 WHERE shop_id = $1 AND status <> 'cancelled' AND created_at >= now() - make_interval(days => $2)`,
		shopID, days).Scan(&res.Revenue, &res.OrdersCount)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	if res.OrdersCount > 0 {
		res.AvgCheck = res.Revenue / float64(res.OrdersCount)
	}

	err = h.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM products WHERE shop_id = $1 AND deleted_at IS NULL AND stock <= 5", shopID).Scan(&res.LowStock)
	if err != nil {
		httpx.Internal(w, err)
		return
	}

	rows, err := h.pool.Query(ctx,
		`SELECT oi.product_id, MAX(oi.product_name), SUM(oi.quantity)::int, SUM(oi.quantity * oi.unit_price)::float8
		 FROM order_items oi JOIN orders o ON o.id = oi.order_id
		 WHERE o.shop_id = $1 AND o.status <> 'cancelled' AND o.created_at >= now() - make_interval(days => $2)
		 GROUP BY oi.product_id ORDER BY 3 DESC, 4 DESC LIMIT 5`, shopID, days)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	for rows.Next() {
		var t TopProduct
		if err := rows.Scan(&t.ProductID, &t.Name, &t.Sold, &t.Revenue); err != nil {
			rows.Close()
			httpx.Internal(w, err)
			return
		}
		res.TopProducts = append(res.TopProducts, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}

	rows, err = h.pool.Query(ctx,
		`SELECT to_char(created_at, 'YYYY-MM-DD'), COUNT(*)::int, SUM(total_amount)::float8
		 FROM orders
		 WHERE shop_id = $1 AND status <> 'cancelled' AND created_at >= now() - make_interval(days => $2)
		 GROUP BY 1 ORDER BY 1`, shopID, days)
	if err != nil {
		httpx.Internal(w, err)
		return
	}
	for rows.Next() {
		var d DayStat
		if err := rows.Scan(&d.Date, &d.Orders, &d.Revenue); err != nil {
			rows.Close()
			httpx.Internal(w, err)
			return
		}
		res.ByDay = append(res.ByDay, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		httpx.Internal(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}
