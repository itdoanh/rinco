module github.com/itdoanh/rinco/packages/go

go 1.23

require (
	github.com/coreos/go-oidc/v3 v3.11.0
	github.com/go-webauthn/webauthn v0.10.2
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/labstack/echo/v4 v4.12.0
	github.com/o1egl/paseto v1.0.0
	github.com/prometheus/client_golang v1.19.1
	github.com/redis/go-redis/v9 v9.5.2
	go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho v0.53.0
	go.opentelemetry.io/contrib/instrumentation/github.com/redis/go-redis/v9 v0.53.0
	go.opentelemetry.io/otel v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric v1.27.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.27.0
	go.opentelemetry.io/otel/exporters/prometheus v0.53.0
	go.opentelemetry.io/otel/sdk v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.24.0
	golang.org/x/oauth2 v0.20.0
)