package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"shopai/internal/admin"
	"shopai/internal/ai"
	"shopai/internal/analytics"
	"shopai/internal/auth"
	"shopai/internal/catalog"
	"shopai/internal/config"
	"shopai/internal/docs"
	"shopai/internal/httpx"
	"shopai/internal/order"
	"shopai/internal/shop"
	"shopai/internal/user"
)

func NewRouter(pool *pgxpool.Pool, cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, cors(cfg.CORSOrigin))

	userH := user.NewHandler(user.NewService(user.NewRepository(pool), cfg.JWTSecret))
	shopH := shop.NewHandler(pool)
	catalogH := catalog.NewHandler(pool)
	orderH := order.NewHandler(pool)
	aiH := ai.NewHandler(pool, ai.NewClient(cfg.LLMURL, cfg.LLMKey, cfg.LLMModel))
	analyticsH := analytics.NewHandler(pool)
	adminH := admin.NewHandler(pool)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db unavailable"})
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Swagger UI и спецификация OpenAPI для фронтендера
	r.Get("/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(docs.Spec)
	})
	r.Get("/docs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docs.SwaggerHTML))
	})

	r.Route("/api", func(r chi.Router) {
		// --- Публичные методы (покупатель без аккаунта) ---
		r.Post("/auth/register", userH.Register)
		r.Post("/auth/login", userH.Login)

		r.Get("/catalog/products", catalogH.PublicList)
		r.Get("/catalog/products/{id}", catalogH.PublicGet)
		r.Get("/catalog/categories", catalogH.Categories)
		r.Get("/catalog/shops/{id}", shopH.Public)

		r.Post("/orders", orderH.Create)
		r.Get("/orders/{number}", orderH.Track)
		r.Post("/orders/{number}/pay", orderH.Pay)

		// --- Авторизованные ---
		r.Group(func(r chi.Router) {
			r.Use(auth.Authenticate(cfg.JWTSecret))
			r.Get("/me", userH.Me)

			// Селлер
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("seller"))

				r.Post("/shop", shopH.Create)
				r.Get("/shop", shopH.GetMine)
				r.Patch("/shop", shopH.Update)

				r.Get("/seller/products", catalogH.List)
				r.Post("/seller/products", catalogH.Create)
				r.Get("/seller/products/{id}", catalogH.Get)
				r.Patch("/seller/products/{id}", catalogH.Update)
				r.Delete("/seller/products/{id}", catalogH.Delete)
				r.Post("/seller/products/{id}/hide", catalogH.Hide())
				r.Post("/seller/products/{id}/show", catalogH.Show())
				r.Patch("/seller/products/{id}/stock", catalogH.SetStock)

				r.Get("/seller/orders", orderH.SellerList)
				r.Get("/seller/orders/{id}", orderH.SellerGet)
				r.Patch("/seller/orders/{id}/status", orderH.ChangeStatus)

				r.Get("/seller/analytics", analyticsH.Get)

				r.Post("/ai/description", aiH.Description)
				r.Post("/ai/price", aiH.Price)
			})

			// Администратор
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole("admin"))

				r.Get("/admin/users", userH.AdminList)
				r.Get("/admin/moderation/shops", adminH.ShopsByStatus)
				r.Get("/admin/moderation/products", adminH.ProductsByStatus)
				r.Post("/admin/shops/{id}/moderate", adminH.ModerateShop)
				r.Post("/admin/products/{id}/moderate", adminH.ModerateProduct)
				r.Post("/admin/categories", adminH.CreateCategory)
			})
		})
	})
	return r
}
