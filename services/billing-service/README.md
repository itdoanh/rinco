# Billing Service

> **Service:** Payment & Subscription management for tenant billing
> **Stack:** Go + Echo + pgx (PostgreSQL)
> **DB:** PostgreSQL (`rinco_billing`)

## Stack

- **Go 1.26+**
- **Echo v4** — HTTP framework
- **pgx v5** — PostgreSQL driver with pgxpool
- **Stripe-like HTTP driver** — REST API stubs (replaceable with stripe-go SDK in production)

## Endpoints

### Public Tenant Billing

```
POST   /api/billing/v1/subscriptions          Create subscription
GET    /api/billing/v1/subscriptions          Get current subscription
PATCH  /api/billing/v1/subscriptions          Change plan
DELETE /api/billing/v1/subscriptions          Cancel subscription
GET    /api/billing/v1/portal                 Stripe billing portal session
GET    /api/billing/v1/invoices               List invoices
GET    /api/billing/v1/invoices/:id           Get invoice
GET    /api/billing/v1/usage                  Current month usage & estimate
```

### Webhook

```
POST   /webhooks/stripe                       Stripe webhook receiver (HMAC verified)
```

### Admin (Internal)

```
POST   /internal/billing.v1/admin/subscriptions   Admin create subscription
GET    /health/live                              Liveness
GET    /health/ready                             Readiness
```

## Plans

| Plan          | Monthly | Max Users | Storage  | Req/s |
|---------------|---------|-----------|----------|-------|
| FREE          | $0      | 5         | 1 GB     | 10    |
| PRO           | $99     | 50        | 50 GB    | 100   |
| BUSINESS      | $299    | 200       | 200 GB   | 500   |
| ENTERPRISE    | $990    | ∞         | ∞        | ∞     |

## Database Schema

See `migrations/0001_init.sql`.

Tables:
- `subscriptions` — Tenant subscription state (1 active per tenant)
- `invoices` — Billing invoices with status (draft, open, paid, void)
- `invoice_line_items` — Line item details
- `usage_records` — Monthly API/storage/AI call counters (UNIQUE per tenant+month)
- `payment_methods` — Stored payment methods (Stripe, Momo, VNPay)
- `discount_codes` — Promotional discount codes
- `webhook_events` — Stripe webhook audit trail with idempotency

## Payment Drivers

The `payment.Driver` interface is implemented by `StripeDriver` which uses
direct REST API calls. For production, swap with the official `stripe-go`
SDK package.

## VAT Rates

Configured per country code (VN 10%, DE 19%, FR 20%, etc.). Computed in
`payment.VATRate()`.

## Trial

All paid plans (PRO/BUSINESS/ENTERPRISE) get a 14-day trial by default.
Override per tenant via admin endpoints.

## Setup

```bash
# Run migrations
psql -h localhost -U postgres -d rinco_billing -f migrations/0001_init.sql

# Start service
export STRIPE_SECRET_KEY=sk_test_xxx
export STRIPE_WEBHOOK_SECRET=whsec_xxx
export STRIPE_PRICE_PRO=price_xxx
go run ./cmd/main.go
```

Default port: `8093`.

## Configuration

| Environment variable     | Default                                             | Description                  |
|--------------------------|-----------------------------------------------------|------------------------------|
| `PORT`                   | `8093`                                              | HTTP listen port             |
| `DATABASE_URL`           | `postgres://postgres:postgres@localhost/rinco_billing` | PostgreSQL DSN           |
| `STRIPE_SECRET_KEY`      | (empty)                                             | Stripe API key (sk_test_*)   |
| `STRIPE_WEBHOOK_SECRET`  | (empty)                                             | Webhook signing secret       |
| `STRIPE_PRICE_*`         | (empty)                                             | Price IDs for each plan      |

## Tests

```bash
go test ./...
```

Coverage includes:
- Model invariants (plans, limits)
- Subscription/invoice/usage record structures
- VAT rates per country
- Subscription trial days per plan
- Driver configuration flag

## Scripts

- `cmd/main.go` — Service entry point
