# Generate Dockerfile for Go services
$ErrorActionPreference = 'Stop'

$RootDir = Split-Path -Parent $PSScriptRoot

$Services = @(
  @{ Name = 'tenant-service'; Port = 8082 },
  @{ Name = 'crm-service'; Port = 8083 },
  @{ Name = 'dynamic-model-service'; Port = 8084 },
  @{ Name = 'lead-service'; Port = 8085 },
  @{ Name = 'landing-service'; Port = 8086 },
  @{ Name = 'email-service'; Port = 8087 },
  @{ Name = 'notification-service'; Port = 8088 },
  @{ Name = 'observability-service'; Port = 8089 }
)

foreach ($svc in $Services) {
  $name = $svc.Name
  $port = $svc.Port
  $svcDir = "services/$name"
  $dockerfilePath = "$RootDir/$svcDir/Dockerfile"

  if (Test-Path $dockerfilePath) {
    Write-Host "Skip (exists): $name"
    continue
  }

  $content = @"
FROM golang:1.23-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY packages/ packages/
COPY $svcDir/ $svcDir/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/$name ./$svcDir/cmd

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/$name /app/$name
EXPOSE $port
ENTRYPOINT ["/app/$name"]
"@
  Set-Content -Path $dockerfilePath -Value $content
  Write-Host "Generated: $name (port $port)"
}

Write-Host "Done."
