package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/itdoanh/rinco/services/search-service/internal/handler"
	"github.com/itdoanh/rinco/services/search-service/internal/models"
	"github.com/itdoanh/rinco/services/search-service/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dbURL := envOr("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/rinco_crm?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Warn("postgres connect failed", "err", err)
	} else {
		defer pool.Close()
	}

	meiliHost := envOr("MEILISEARCH_HOST", "http://localhost:7700")
	meiliKey := envOr("MEILISEARCH_API_KEY", "")
	store := repository.NewMeilisearch(meiliHost, meiliKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		logger.Warn("meilisearch unreachable, continuing with limited features", "err", err)
	} else {
		logger.Info("meilisearch connected", "host", meiliHost)
		srvInit := handler.New(store, pool)
		if err := srvInit.InitIndex(context.Background()); err != nil {
			logger.Warn("init index failed", "err", err)
		}
	}

	srv := handler.New(store, pool)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.GET("/health/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	g := e.Group("/api/search/v1")
	g.POST("/documents", srv.IndexDocument)
	g.POST("/documents/bulk", srv.BulkIndex)
	g.DELETE("/documents/:id", srv.DeleteDocument)
	g.GET("/search", srv.Search)
	g.POST("/reindex", srv.Reindex)

	// v1 spec-compliant endpoints ----------------------------------------
	v1 := e.Group("/v1")
	v1.Use(middleware.Recover())

	// Universal search
	v1.POST("/search", srv.UniversalSearch)
	v1.POST("/search/leads", srv.SearchByType(models.TypeLead))
	v1.POST("/search/contacts", srv.SearchByType(models.TypeContact))
	v1.POST("/search/deals", srv.SearchByType(models.TypeDeal))
	v1.POST("/search/users", srv.SearchByType(models.TypeUser))
	v1.POST("/search/global", srv.GlobalSearch)

	// Indexing
	v1.POST("/index", srv.IndexDocument)
	v1.POST("/index/bulk", srv.BulkIndex)
	v1.DELETE("/index/:entity_type/:id", srv.DeleteByEntityType)

	// Autocomplete + facets
	v1.GET("/suggest", srv.Suggest)
	v1.GET("/facets/:entity_type", srv.Facets)

	port := envOr("PORT", "8087")
	srv2 := &http.Server{Addr: ":" + port, Handler: e}
	go func() {
		logger.Info("search-service starting", "port", port)
		if err := srv2.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	if err := srv2.Shutdown(ctx2); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
