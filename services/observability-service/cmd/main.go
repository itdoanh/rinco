// Observability Service - aggregate metrics, health, alerting.
package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/rinco/go/pkg/logger"
	rincowebmw "github.com/rinco/go/pkg/middleware"
)

const (
	serviceName = "observability-service"
	version     = "1.0.0"
)

type serviceHealth struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	URL      string `json:"url"`
	LastCheck time.Time `json:"last_check"`
	LatencyMs int64 `json:"latency_ms"`
}

func main() {
	env := getEnv("ENV", "development")
	logger.Init(serviceName, env, version)
	defer logger.Sync()

	e := echo.New()
	e.HideBanner = true
	e.Use(rincowebmw.Recovery())
	e.Use(rincowebmw.Trace())
	e.Use(rincowebmw.Logger())
	e.Use(rincowebmw.Metrics(serviceName))
	e.Use(rincowebmw.CORS([]string{"*"}))
	e.Use(rincowebmw.SecurityHeaders())

	e.GET("/health", healthHandler)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.GET("/v1/observability/health/all", healthAllHandler)
	e.GET("/v1/observability/health/:service", healthServiceHandler)

	port := ":" + getEnv("PORT", "8089")
	logger.Info(context.Background(), "starting observability service", zap.String("port", port))
	if err := e.Start(port); err != nil && err != http.ErrServerClosed {
		logger.Fatal(context.Background(), "server failed", err)
	}
}

func healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": serviceName})
}

func healthAllHandler(c echo.Context) error {
	services := []serviceHealth{
		checkService("auth-service", "http://auth-service:8081/health"),
		checkService("tenant-service", "http://tenant-service:8082/health"),
		checkService("crm-service", "http://crm-service:8083/health"),
		checkService("dynamic-model-service", "http://dynamic-model-service:8084/health"),
		checkService("landing-service", "http://landing-service:8086/health"),
		checkService("email-service", "http://email-service:8087/health"),
		checkService("notification-service", "http://notification-service:8088/health"),
		checkService("lead-scoring", "http://lead-scoring:8092/health"),
		checkService("ai-sre", "http://ai-sre:8090/health"),
		checkService("rag-chatbot", "http://rag-chatbot:8091/health"),
		checkService("stt-service", "http://stt-service:8093/health"),
	}

	overall := "healthy"
	for _, s := range services {
		if s.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"overall":  overall,
		"services": services,
		"checked_at": time.Now(),
	})
}

func healthServiceHandler(c echo.Context) error {
	name := c.Param("service")

	urls := map[string]string{
		"auth-service":          "http://auth-service:8081/health",
		"tenant-service":        "http://tenant-service:8082/health",
		"crm-service":           "http://crm-service:8083/health",
		"dynamic-model-service": "http://dynamic-model-service:8084/health",
		"landing-service":       "http://landing-service:8086/health",
		"email-service":         "http://email-service:8087/health",
		"notification-service":  "http://notification-service:8088/health",
		"lead-scoring":          "http://lead-scoring:8092/health",
		"ai-sre":                "http://ai-sre:8090/health",
		"rag-chatbot":           "http://rag-chatbot:8091/health",
		"stt-service":           "http://stt-service:8093/health",
	}

	url, ok := urls[name]
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "unknown service"})
	}

	return c.JSON(http.StatusOK, checkService(name, url))
}

func checkService(name, url string) serviceHealth {
	client := &http.Client{Timeout: 3 * time.Second}
	start := time.Now()
	resp, err := client.Get(url)
	latency := time.Since(start).Milliseconds()

	status := "ok"
	if err != nil || resp.StatusCode != 200 {
		status = "down"
	}
	if resp != nil {
		resp.Body.Close()
	}

	return serviceHealth{
		Name:      name,
		Status:    status,
		URL:       url,
		LastCheck: time.Now(),
		LatencyMs: latency,
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
