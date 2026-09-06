// Auth Service - handles authentication, authorization, FIDO2.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	rincoauth "github.com/rinco/go/pkg/auth"
	"github.com/rinco/go/pkg/db"
	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "auth-service"
	version     = "1.0.0"
)

var (
	pasetoKeyRing *rincoauth.KeyRing
)

type loginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	TenantSlug string `json:"tenant_slug" validate:"required"`
	DeviceFP   string `json:"device_fingerprint"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	UserID       string `json:"user_id"`
	TenantID     string `json:"tenant_id"`
	Roles        []string `json:"roles"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	// DB connection
	dbCfg := db.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     5432,
		User:     getEnv("DB_USER", "rinco"),
		Password: getEnv("DB_PASSWORD", "rinco_dev_password"),
		Database: getEnv("DB_NAME", "rinco"),
		SSLMode:  "disable",
	}
	database, err := db.New(dbCfg)
	if err != nil {
		logger.Fatal(context.Background(), "db connect failed", err)
	}
	defer database.Close()

	// PASETO key
	currentKey := getEnv("PASETO_KEY_CURRENT", "")
	if currentKey == "" {
		var err error
		currentKey, err = rincoauth.GeneratePASETOKey()
		if err != nil {
			logger.Fatal(context.Background(), "generate paseto key", err)
		}
		logger.Warn(context.Background(), "PASETO_KEY_CURRENT not set, generated ephemeral (DO NOT USE IN PROD)",
			zap.String("key", currentKey))
	}
	previousKey := getEnv("PASETO_KEY_PREVIOUS", "")
	pasetoKeyRing, err = rincoauth.NewKeyRing(currentKey, previousKey)
	if err != nil {
		logger.Fatal(context.Background(), "create key ring", err)
	}

	// Echo HTTP server
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	// Routes
	e.GET("/health", healthHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	auth := e.Group("/v1/auth")
	auth.POST("/login", loginHandler(database))
	auth.POST("/refresh", refreshHandler(database))
	auth.POST("/logout", logoutHandler(database))
	auth.POST("/webauthn/register/begin", webauthnRegisterBegin(database))
	auth.POST("/webauthn/register/finish", webauthnRegisterFinish(database))
	auth.POST("/webauthn/login/begin", webauthnLoginBegin(database))
	auth.POST("/webauthn/login/finish", webauthnLoginFinish(database))
	auth.POST("/pow/challenge", powChallengeHandler())
	auth.POST("/pow/verify", powVerifyHandler())

	// Protected routes
	authProtected := e.Group("/v1/auth")
	authProtected.Use(authMiddleware())
	authProtected.GET("/me", meHandler(database))

	port := ":" + getEnv("PORT", "8081")
	logger.Info(context.Background(), "starting auth service",
		zap.String("port", port),
		zap.String("env", env))

	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

// ============ Handlers ============

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func loginHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req loginRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, apiError{Code: "INVALID_REQUEST", Message: err.Error()})
		}

		ctx := c.Request().Context()

		// Find user
		var (
			userID, tenantID, passwordHash string
			roles                          []string
			isSuperAdmin                   bool
		)
		err := database.QueryRowContext(ctx, `
			SELECT u.id::text, u.tenant_id::text, COALESCE(u.password_hash, ''), u.roles, u.is_super_admin
			FROM auth.users u
			JOIN tenant.tenants t ON t.id = u.tenant_id
			WHERE u.email = $1 AND t.slug = $2 AND u.deleted_at IS NULL
			LIMIT 1
		`, req.Email, req.TenantSlug).Scan(&userID, &tenantID, &passwordHash, &roles, &isSuperAdmin)

		if err != nil {
			logger.Warn(ctx, "user not found", zap.String("email", req.Email))
			return c.JSON(http.StatusUnauthorized, apiError{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"})
		}

		// Verify password
		ok, err := rincoauth.VerifyPassword(req.Password, passwordHash)
		if err != nil || !ok {
			logger.Warn(ctx, "invalid password", zap.String("user_id", userID))
			return c.JSON(http.StatusUnauthorized, apiError{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password"})
		}

		// Generate access token
		accessToken, err := generateAccessToken(userID, tenantID, roles, isSuperAdmin)
		if err != nil {
			logger.Error(ctx, "generate access token", err)
			return c.JSON(http.StatusInternalServerError, apiError{Code: "TOKEN_FAILED", Message: "Failed to generate token"})
		}

		// Generate refresh token
		refreshToken, err := rincoauth.GenerateSecureToken(32)
		if err != nil {
			logger.Error(ctx, "generate refresh token", err)
			return c.JSON(http.StatusInternalServerError, apiError{Code: "TOKEN_FAILED", Message: "Failed to generate token"})
		}

		// Store refresh token
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		_, err = database.ExecContext(ctx, `
			INSERT INTO auth.refresh_tokens (id, user_id, token_hash, device_fingerprint, ip_address, expires_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, uuid.NewV7(), userID, hashToken(refreshToken), req.DeviceFP, c.RealIP(), expiresAt)
		if err != nil {
			logger.Error(ctx, "store refresh token", err)
		}

		// Update last_login_at
		_, _ = database.ExecContext(ctx, `UPDATE auth.users SET last_login_at = NOW() WHERE id = $1`, userID)

		return c.JSON(http.StatusOK, loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    3600,
			UserID:       userID,
			TenantID:     tenantID,
			Roles:        roles,
		})
	}
}

func refreshHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req refreshRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, apiError{Code: "INVALID_REQUEST", Message: err.Error()})
		}

		ctx := c.Request().Context()
		tokenHash := hashToken(req.RefreshToken)

		var userID, tenantID string
		var roles []string
		var isSuperAdmin bool
		err := database.QueryRowContext(ctx, `
			SELECT u.id::text, u.tenant_id::text, u.roles, u.is_super_admin, u.is_super_admin
			FROM auth.refresh_tokens rt
			JOIN auth.users u ON u.id = rt.user_id
			WHERE rt.token_hash = $1
			  AND rt.revoked_at IS NULL
			  AND rt.expires_at > NOW()
			LIMIT 1
		`, tokenHash).Scan(&userID, &tenantID, &roles, &isSuperAdmin, &isSuperAdmin)

		if err != nil {
			return c.JSON(http.StatusUnauthorized, apiError{Code: "INVALID_REFRESH_TOKEN", Message: "Invalid or expired refresh token"})
		}

		accessToken, err := generateAccessToken(userID, tenantID, roles, isSuperAdmin)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, apiError{Code: "TOKEN_FAILED", Message: "Failed to generate token"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"access_token": accessToken,
			"expires_in":   3600,
		})
	}
}

func logoutHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		userID := logger.UserIDFromContext(ctx)
		if userID == "" {
			return c.JSON(http.StatusUnauthorized, apiError{Code: "UNAUTHORIZED", Message: "Not authenticated"})
		}

		// Revoke all refresh tokens for user
		_, err := database.ExecContext(ctx, `
			UPDATE auth.refresh_tokens SET revoked_at = NOW()
			WHERE user_id = $1 AND revoked_at IS NULL
		`, userID)

		if err != nil {
			logger.Error(ctx, "revoke tokens", err)
		}

		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

func meHandler(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		userID := logger.UserIDFromContext(ctx)
		tenantID := logger.TenantIDFromContext(ctx)

		var email, fullName string
		var roles []string
		err := database.QueryRowContext(ctx, `
			SELECT email, COALESCE(full_name, ''), roles FROM auth.users WHERE id = $1
		`, userID).Scan(&email, &fullName, &roles)

		if err != nil {
			return c.JSON(http.StatusNotFound, apiError{Code: "USER_NOT_FOUND", Message: "User not found"})
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"user_id":   userID,
			"tenant_id": tenantID,
			"email":     email,
			"full_name": fullName,
			"roles":     roles,
		})
	}
}

func webauthnRegisterBegin(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Implementation của WebAuthn registration begin
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

func webauthnRegisterFinish(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

func webauthnLoginBegin(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

func webauthnLoginFinish(database *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

func powChallengeHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Argon2 PoW challenge
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

func powVerifyHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "not_implemented"})
	}
}

// ============ Middleware ============

func authMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			auth := c.Request().Header.Get("Authorization")
			if auth == "" || len(auth) < 8 || auth[:7] != "Bearer " {
				return c.JSON(http.StatusUnauthorized, apiError{Code: "MISSING_TOKEN", Message: "Missing bearer token"})
			}

			token := auth[7:]
			claims, err := pasetoKeyRing.Decrypt(token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, apiError{Code: "INVALID_TOKEN", Message: "Invalid or expired token"})
			}

			ctx = logger.WithUserID(ctx, claims.UserID)
			ctx = logger.WithTenantID(ctx, claims.TenantID)
			if len(claims.Roles) > 0 && claims.Roles[0] == "super_admin" {
				ctx = context.WithValue(ctx, db.IsAdminKey, true)
			}

			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}

// ============ Helpers ============

func generateAccessToken(userID, tenantID string, roles []string, isSuperAdmin bool) (string, error) {
	now := time.Now()
	claims := rincoauth.Claims{
		Subject:   userID,
		TenantID:  tenantID,
		UserID:    userID,
		Roles:     roles,
		IssuedAt:  now,
		NotBefore: now,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	if isSuperAdmin {
		claims.Scope = "super_admin"
	}
	return pasetoKeyRing.Encrypt(claims)
}

func hashToken(token string) string {
	// Simple hash for token lookup; in production use sha256
	return token[:32]
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// unused, but keeps compile time check for unused imports.
var _ = errors.New
var _ = fmt.Sprintf
