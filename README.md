# Gess Backend - Setup Guide

## Prerequisites

- Go 1.25 or higher
- PostgreSQL 12 or higher
- Git

## Database Setup

### 1. Install PostgreSQL

**macOS:**
```bash
brew install postgresql@15
brew services start postgresql@15
```

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
```

### 2. Create Database

```bash
# Connect to PostgreSQL
psql postgres

# In psql prompt:
CREATE DATABASE ecommerce_db;
CREATE USER ecommerce_user WITH ENCRYPTED PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE ecommerce_db TO ecommerce_user;
\q
```

### 3. Run schema and seed migration

Create tables (DDL only):

```bash
psql -U ecommerce_user -d ecommerce_db -f database/schema.sql
```

Then load seed data:

```bash
psql -U ecommerce_user -d ecommerce_db -f database/migrations/001_seed_data.sql
```

Or if using the default `postgres` user:

```bash
psql -d ecommerce_db -f database/schema.sql
psql -d ecommerce_db -f database/migrations/001_seed_data.sql
```

## Application Setup

### 1. Install Dependencies

```bash
cd store-backend
go mod tidy
```

This will download:
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/google/uuid` - UUID generation
- `golang.org/x/crypto` - Password hashing (bcrypt)
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `github.com/rs/cors` - CORS middleware
- `github.com/jwambugu/mpesa-golang-sdk` - M-PESA Daraja (STK Push) integration

### 2. Configure Environment Variables

Create a `.env` file in the backend directory or export these variables:

```bash
export DATABASE_URL="postgres://ecommerce_user:your_secure_password@localhost:5432/ecommerce_db?sslmode=disable"

export BASE_URL="http://localhost:3000"

export ADMIN_UI_URL="http://localhost:4200"

export PORT="8080"

# Required: JWT signing secret (minimum 32 bytes). Example: openssl rand -base64 48
export JWT_SECRET="replace-with-a-long-random-secret-at-least-32-chars"

# Session cookie (optional). For local HTTP, defaults are COOKIE_SECURE=false and COOKIE_SAMESITE=lax.
# Production cross-site cookies: COOKIE_SAMESITE=none and COOKIE_SECURE=true
export COOKIE_SECURE="false"
export COOKIE_SAMESITE="lax"

# M-PESA (Safaricom Daraja) - optional; omit to disable M-PESA checkout
export MPESA_CONSUMER_KEY="your_consumer_key"
export MPESA_CONSUMER_SECRET="your_consumer_secret"
export MPESA_PASSKEY="your_passkey"
export MPESA_SHORTCODE="174379"
export MPESA_CALLBACK_BASE_URL="https://your-api.example.com"
export MPESA_ENV="sandbox"
```

**M-PESA checkout:** Get credentials from the [Safaricom Daraja](https://developer.safaricom.co.ke) portal. Use `MPESA_ENV=sandbox` for testing. The callback URL must be publicly reachable over HTTPS in production (`MPESA_CALLBACK_BASE_URL` + `/webhooks/mpesa/stk`). If any M-PESA variable is missing, checkout via M-PESA is disabled.

**Alternative DATABASE_URL formats:**

For local development:
```bash
DATABASE_URL="postgres://username:password@localhost:5432/dbname?sslmode=disable"
```

For production (with SSL):
```bash
DATABASE_URL="postgres://username:password@host:5432/dbname?sslmode=require"
```

### 3. Run the Server

Load environment variables and start the server:

```bash
source .env.sh   # or: set -a && source .env.sh && set +a
go run .
```

The server will start on `http://localhost:8080`. Ensure `ADMIN_UI_URL` is set in `.env.sh` (or equivalent) so the admin dashboard at `http://localhost:4200` can make API requests. If `ADMIN_UI_URL` is not set, the server falls back to allowing `http://localhost:4200` for development.

## Observability

- **Logs:** JSON lines to stdout via `log/slog`. Set `LOG_LEVEL` to `debug`, `info`, `warn`, or `error` (default `info`). Each HTTP request gets an `X-Request-ID` (echoed from the client when valid, otherwise generated); access logs include `request_id`, `method`, `path`, `route`, `status`, and `duration_ms`.
- **Metrics:** `GET /metrics` exposes Prometheus metrics. Important series include `http_request_duration_seconds` (histogram, labels `method`, `route`, `status_class`), `db_up` (updated by a periodic PostgreSQL ping), `mail_configured`, and `mpesa_consumer_configured`. Restrict `/metrics` at the network edge in production.
- **Tracing:** Optional OTLP/HTTP export. Set `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to enable (see [OTLP environment variables](https://opentelemetry.io/docs/specs/otel/protocol/exporter/)). Set `OTEL_SERVICE_NAME` to override the default service name `gess-backend`. If no OTLP endpoint is set, tracing export is disabled but W3C trace context propagation is still configured.

## API Endpoints

### Authentication

#### Register User
```bash
POST /register
Content-Type: application/json

{
  "username": "john_doe",
  "email": "john@example.com",
  "password": "securepassword123"
}
```

Public registration always creates a `user` role. To create users with `admin` or `super_admin`, use **POST /admin/users** (requires an authenticated admin). Only a `super_admin` may assign the `super_admin` role. Promote the first admin with SQL (see Security Notes).

#### Login
```bash
POST /login
Content-Type: application/json

{
  "username": "john_doe",
  "password": "securepassword123"
}
```

Response includes a JWT token in both the response body and an HTTP-only cookie.

#### Logout
```bash
POST /logout
```

#### Current user
```bash
GET /me
Cookie: token=<jwt_token>
```

#### Create user (admin)
```bash
POST /admin/users
Authorization: Bearer <admin_jwt>
Content-Type: application/json

{
  "username": "staff1",
  "email": "staff1@example.com",
  "password": "securepassword123",
  "role": "admin"
}
```

### M-PESA Checkout (Lipa Na M-Pesa Online)

Checkout with M-PESA: the client sends address and phone; the backend creates the order, deducts stock, clears the cart, and initiates an STK push. The customer enters their M-PESA PIN on their phone. Safaricom sends a callback to your server; on success the order moves to `processing`, on cancel/timeout the order is `cancelled` and stock is restored.

#### Initiate M-PESA checkout
```bash
POST /orders/checkout-mpesa
Authorization: Bearer <token>
Content-Type: application/json

{
  "shipping_address_id": "uuid-of-address",
  "phone_number": "254712345678"
}
```

Response (201): order object plus `checkout_request_id` and message "Complete payment on your phone". The client can poll `GET /orders/:id` to see when `status` becomes `processing` and `mpesa_receipt_number` is set.

#### Webhook (Safaricom only)
`POST /webhooks/mpesa/stk` – no auth; receives STK push result from Safaricom. Must be HTTPS in production.

## Database Schema

The database includes the following tables:

- **users** - User accounts with authentication
- **categories** - Product categories
- **products** - Product catalog with pricing and inventory
- **addresses** - User shipping/billing addresses
- **carts** - Shopping carts
- **cart_items** - Items in shopping carts
- **orders** - Customer orders
- **order_items** - Items in orders
- **reviews** - Product reviews and ratings

### Sample Data

The schema includes sample categories:
- Electronics
- Clothing
- Books
- Home & Garden
- Sports

## Project Structure

```text
backend/
├── main.go              # Main application entry point
├── database/
│   ├── db.go                      # Database connection and configuration
│   ├── schema.sql               # Tables, indexes, constraints (no seed data)
│   └── migrations/
│       └── 001_seed_data.sql    # Forward-only seed migration
├── models/
│   ├── user.go         # User model with password hashing
│   ├── product.go      # Product model
│   ├── category.go     # Category model
│   ├── cart.go         # Cart and CartItem models
│   ├── order.go        # Order and OrderItem models
│   ├── address.go      # Address model
│   └── review.go       # Review model
├── go.mod              # Go module dependencies
└── README.md           # This file
```

## Implementation status

The service already exposes a full storefront and admin surface: public catalog (`/products`, `/products/batch`, product reviews), categories and brands, authenticated cart and addresses, order creation and listing, guest checkout (`POST /checkout/guest`), M-PESA checkout and webhook, user reviews, and admin CRUD for products, categories, brands, users, and orders. JWT auth and admin role checks are applied via `AuthMiddleware` and `AdminMiddleware` in [`middleware/auth.go`](middleware/auth.go). HTTP routes are registered in [`main.go`](main.go); request handlers live under [`handlers/`](handlers/).

The **API Endpoints** section above documents authentication and M-PESA in detail; refer to `main.go` for the complete route map.

### Roadmap / production hardening

- Broader rate limiting (today only `POST /checkout/guest` is limited per client IP in [`handlers/checkout.go`](handlers/checkout.go))
- Structured request or access logging middleware
- CSRF strategy for cookie-based sessions where needed
- Expand README API documentation to cover catalog, cart, orders, and admin routes end-to-end

## Development Tips

### View Database Tables
```bash
psql -U ecommerce_user -d ecommerce_db -c "\dt"
```

### Query Examples
```sql
-- View all users
SELECT * FROM users;

-- View products with category names
SELECT p.*, c.name as category_name 
FROM products p 
LEFT JOIN categories c ON p.category_id = c.id;

-- View orders with items
SELECT o.*, oi.* 
FROM orders o 
JOIN order_items oi ON o.id = oi.order_id;
```

### Reset Database
```bash
psql -U ecommerce_user -d ecommerce_db -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
psql -U ecommerce_user -d ecommerce_db -f database/schema.sql
```

## Troubleshooting

### Connection Issues

If you get `connection refused`:
1. Check PostgreSQL is running: `pg_isready`
2. Verify DATABASE_URL is correct
3. Check PostgreSQL logs: `tail -f /usr/local/var/log/postgresql@15.log`

### Import Errors

If you see import errors:
```bash
go mod tidy
go mod download
```

### Permission Denied

If you get permission errors with PostgreSQL:
```sql
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ecommerce_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ecommerce_user;
```

## Integration tests

With PostgreSQL schema and seed applied (`database/schema.sql` then `database/migrations/001_seed_data.sql`):

```bash
export TEST_DATABASE_URL="postgres://user:pass@localhost:5432/ecommerce_db?sslmode=disable"
go test -tags=integration ./handlers/ -v
```

These tests cover guest checkout, cart stock checks, and the M-PESA STK callback handler against a real database.

## Security Notes

⚠️ **Important for Production:**

1. Set a strong `JWT_SECRET` (at least 32 characters); the server refuses to start without it
2. Promote the first admin manually (no open self-service admin signup), for example:  
   `UPDATE users SET role = 'admin' WHERE username = 'your_first_user';`
3. Use strong, unique DATABASE_URL password
4. Enable SSL for database connections (`sslmode=require`)
5. Add broader rate limiting (guest checkout already has per-IP limiting; see [`handlers/checkout.go`](handlers/checkout.go))
6. Add request validation
7. Use HTTPS in production; set `COOKIE_SECURE=true` and choose `COOKIE_SAMESITE` appropriately
8. Implement proper error handling (don't expose sensitive info)
9. Add SQL injection prevention (use parameterized queries - already implemented)
10. Implement CSRF protection
11. Add logging and monitoring

## License
