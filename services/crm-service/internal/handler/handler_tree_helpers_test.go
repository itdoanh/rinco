// Tests for CRM tree handler helper functions.
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

func newTestServer() *Server {
	return NewServer(&pgxpool.Pool{}, nil)
}

func TestGetUserID_Empty(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := newTestServer()
	id, ok := s.GetUserID(c)
	if id != "" {
		t.Errorf("expected empty, got %s", id)
	}
	if ok {
		t.Error("expected ok=false")
	}
}

func TestGetUserID_Present(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("user_id", "u1")

	s := newTestServer()
	id, ok := s.GetUserID(c)
	if id != "u1" {
		t.Errorf("expected u1, got %s", id)
	}
	if !ok {
		t.Error("expected ok=true")
	}
}

func TestIsAdmin_False(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	s := newTestServer()
	if s.IsAdmin(c) {
		t.Error("expected false")
	}
}

func TestIsAdmin_True(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("is_admin", true)

	s := newTestServer()
	if !s.IsAdmin(c) {
		t.Error("expected true")
	}
}

func TestIsAdmin_WrongType(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("is_admin", "yes") // string instead of bool

	s := newTestServer()
	if s.IsAdmin(c) {
		t.Error("expected false for non-bool")
	}
}
