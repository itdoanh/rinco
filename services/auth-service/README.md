# Auth Service

Service xác thực và phân quyền cho RINCO platform.

## Tính năng

- **PASETO v4**: Stateless tokens thay thế JWT, bảo mật cao hơn
- **Argon2id**: Password hashing với parameters cao (m=65536, t=3, p=4)
- **WebAuthn/FIDO2**: Hardware-backed authentication (YubiKey, TouchID, Windows Hello)
- **Argon2 Proof-of-Work**: Chống bot, DDoS cho login endpoint
- **Multi-tenant**: Tách biệt authentication theo tenant
- **RBAC**: Role-based access control với 5 roles (super_admin, tenant_admin, manager, user, guest)
- **MFA**: Multi-factor authentication với TOTP
- **Refresh Token Rotation**: Tự động rotate refresh tokens

## Công nghệ

- **Language**: Go 1.23+
- **Framework**: Echo v4
- **Database**: PostgreSQL (pgx/v5)
- **Cryptography**: o1egl/paseto, golang.org/x/crypto/argon2
- **WebAuthn**: github.com/go-webauthn/webauthn

## API Endpoints

```
POST   /v1/auth/login                  - Đăng nhập với email/password
POST   /v1/auth/login/pow              - Login với Proof-of-Work challenge
POST   /v1/auth/refresh                - Refresh access token
POST   /v1/auth/logout                 - Đăng xuất, thu hồi refresh token
GET    /v1/auth/me                     - Lấy thông tin user hiện tại
POST   /v1/auth/webauthn/register      - Bắt đầu FIDO2 registration
POST   /v1/auth/webauthn/verify        - Verify FIDO2 registration
POST   /v1/auth/webauthn/login/start   - Bắt đầu FIDO2 login
POST   /v1/auth/webauthn/login/finish  - Hoàn thành FIDO2 login
POST   /v1/auth/mfa/enable             - Enable MFA
POST   /v1/auth/mfa/verify             - Verify MFA code
```

## Environment Variables

```bash
AUTH_SERVICE_PORT=8081
DATABASE_URL=postgres://postgres:postgres@localhost:5432/rinco?sslmode=disable
PASETO_KEY=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
ARGON2_MEMORY=65536
ARGON2_TIME=3
ARGON2_PARALLELISM=4
JWT_TTL=3600
REFRESH_TOKEN_TTL=2592000
POW_DIFFICULTY=3
```

## Development

```bash
# Build
go build -o bin/auth-service ./cmd/main.go

# Run
./bin/auth-service

# Test
go test ./...
```

## Docker

```bash
docker build -t rinco/auth-service:latest .
docker run -p 8081:8081 --env-file .env rinco/auth-service:latest
```
