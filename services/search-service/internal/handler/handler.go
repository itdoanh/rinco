package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"github.com/itdoanh/rinco/services/search-service/internal/models"
	"github.com/itdoanh/rinco/services/search-service/internal/repository"
)

const indexName = "rinco_documents"

type Server struct {
	store *repository.MeilisearchStore
	pool  *pgxpool.Pool
	log   *slog.Logger
	mu    sync.RWMutex
}

func New(store *repository.MeilisearchStore, pool *pgxpool.Pool) *Server {
	return &Server{store: store, pool: pool, log: slog.Default()}
}

// IndexDocument adds or updates a document in the search index
func (s *Server) IndexDocument(c echo.Context) error {
	var doc models.Document
	if err := c.Bind(&doc); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid document"})
	}
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = time.Now()

	docs := []models.Document{doc}
	if err := s.store.IndexDocuments(c.Request().Context(), indexName, docs); err != nil {
		s.log.Error("index document failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, doc)
}

// BulkIndex indexes many documents at once
func (s *Server) BulkIndex(c echo.Context) error {
	var docs []models.Document
	if err := c.Bind(&docs); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid documents"})
	}
	now := time.Now()
	for i := range docs {
		if docs[i].ID == "" {
			docs[i].ID = uuid.New().String()
		}
		if docs[i].CreatedAt.IsZero() {
			docs[i].CreatedAt = now
		}
		docs[i].UpdatedAt = now
	}
	if err := s.store.IndexDocuments(c.Request().Context(), indexName, docs); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]int{"indexed": len(docs)})
}

// Search executes a search query
func (s *Server) Search(c echo.Context) error {
	q := models.SearchQuery{
		Query:      c.QueryParam("q"),
		Highlight: true,
	}
	if t := c.Request().Header.Get("X-Tenant-ID"); t != "" {
		q.TenantID = t
	}
	if v := c.QueryParam("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			q.Page = p
		}
	}
	if v := c.QueryParam("hits_per_page"); v != "" {
		if hpp, err := strconv.Atoi(v); err == nil {
			q.HitsPerPage = hpp
		}
	}
	if v := c.QueryParam("type"); v != "" {
		q.Types = []models.SearchableType{models.SearchableType(v)}
	}

	result, err := s.store.Search(c.Request().Context(), indexName, q)
	if err != nil {
		s.log.Error("search failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteDocument removes a document
func (s *Server) DeleteDocument(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "id required"})
	}
	if err := s.store.DeleteDocument(c.Request().Context(), indexName, id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Reindex rebuilds the entire search index from PostgreSQL data
func (s *Server) Reindex(c echo.Context) error {
	ctx := c.Request().Context()
	if s.pool == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "no database"})
	}

	// Build documents from PostgreSQL across all CRM tables
	docs := []models.Document{}

	// Index contacts
	type contactRow struct {
		ID    string
		Tenant string
		Name  string
		Email string
		Phone string
	}
	rows, err := s.pool.Query(ctx, "SELECT id::text, tenant_id::text, COALESCE(full_name, ''), COALESCE(email, ''), COALESCE(phone, '') FROM contacts LIMIT 10000")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c contactRow
			if err := rows.Scan(&c.ID, &c.Tenant, &c.Name, &c.Email, &c.Phone); err == nil {
				docs = append(docs, models.Document{
					ID:       "contact_" + c.ID,
					TenantID: c.Tenant,
					Type:     models.TypeContact,
					Title:    c.Name,
					Body:     c.Email + " " + c.Phone,
					Metadata: map[string]interface{}{"email": c.Email, "phone": c.Phone},
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
		}
	}

	// Index companies
	row2, err := s.pool.Query(ctx, "SELECT id::text, tenant_id::text, COALESCE(name, '') FROM companies LIMIT 10000")
	if err == nil {
		defer row2.Close()
		for row2.Next() {
			var id, tenant, name string
			if err := row2.Scan(&id, &tenant, &name); err == nil {
				docs = append(docs, models.Document{
					ID:       "company_" + id,
					TenantID: tenant,
					Type:     models.TypeCompany,
					Title:    name,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				})
			}
		}
	}

	if err := s.store.IndexDocuments(ctx, indexName, docs); err != nil {
		s.log.Error("reindex failed", "err", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]int{"indexed": len(docs)})
}

// InitIndex creates the index and configures Vietnamese-friendly settings
func (s *Server) InitIndex(ctx context.Context) error {
	if err := s.store.CreateIndex(ctx, indexName, "id"); err != nil {
		s.log.Warn("create index failed", "err", err)
	}

	settings := map[string]any{
		"searchableAttributes": []string{"title", "body", "tags"},
		"filterableAttributes": []string{"tenant_id", "type", "tags"},
		"sortableAttributes":   []string{"created_at", "updated_at"},
		"rankingRules": []string{
			"words", "typo", "proximity", "attribute", "sort", "exactness",
		},
		"stopWords": []string{},
		"synonyms":  models.SynonymsDictionary,
	}
	return s.store.UpdateSettings(ctx, indexName, settings)
}
