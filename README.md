# E-Commerce Backend - Setup Guide

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

### 3. Run Schema

```bash
psql -U ecommerce_user -d ecommerce_db -f database/schema.sql
```

Or if using default postgres user:
```bash
psql -d ecommerce_db -f database/schema.sql
```

## Application Setup

### 1. Install Dependencies

```bash
cd backend
go mod tidy
```

This will download:
- `github.com/lib/pq` - PostgreSQL driver
- `github.com/google/uuid` - UUID generation
- `golang.org/x/crypto` - Password hashing (bcrypt)
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `github.com/rs/cors` - CORS middleware

### 2. Configure Environment Variables

Create a `.env` file in the backend directory or export these variables:

```bash
# Database connection
export DATABASE_URL="postgres://ecommerce_user:your_secure_password@localhost:5432/ecommerce_db?sslmode=disable"

# Frontend URL for CORS
export BASE_URL="http://localhost:3000"

# Server port (optional, defaults to 8080)
export PORT="8080"
```

**Alternative DATABASE_URL formats:**

For local development:
```
DATABASE_URL="postgres://username:password@localhost:5432/dbname?sslmode=disable"
```

For production (with SSL):
```
DATABASE_URL="postgres://username:password@host:5432/dbname?sslmode=require"
```

### 3. Run the Server

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Authentication

#### Register User
```bash
POST /register
Content-Type: application/json

{
  "username": "john_doe",
  "password": "securepassword123"
}
```

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

#### Protected Route (Example)
```bash
GET /protected
Cookie: token=<jwt_token>
```

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

```
backend/
├── main.go              # Main application entry point
├── database/
│   ├── db.go           # Database connection and configuration
│   └── schema.sql      # Database schema
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

## Next Steps

### Implement Product APIs
Create handlers for product CRUD operations:
- `GET /products` - List all products
- `GET /products/:id` - Get product by ID
- `POST /products` - Create product (admin)
- `PUT /products/:id` - Update product (admin)
- `DELETE /products/:id` - Delete product (admin)

### Implement Cart APIs
- `GET /cart` - Get user's cart
- `POST /cart/items` - Add item to cart
- `PUT /cart/items/:id` - Update cart item quantity
- `DELETE /cart/items/:id` - Remove item from cart

### Implement Order APIs
- `POST /orders` - Create order from cart
- `GET /orders` - Get user's orders
- `GET /orders/:id` - Get order details

### Add Middleware
- Authentication middleware for protected routes
- Admin role checking middleware
- Request logging
- Rate limiting

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

## Security Notes

⚠️ **Important for Production:**

1. Change `jwtKey` in main.go to use environment variable
2. Use strong, unique DATABASE_URL password
3. Enable SSL for database connections (`sslmode=require`)
4. Implement rate limiting
5. Add request validation
6. Use HTTPS in production
7. Implement proper error handling (don't expose sensitive info)
8. Add SQL injection prevention (use parameterized queries - already implemented)
9. Implement CSRF protection
10. Add logging and monitoring

## License

This is a demo project for learning purposes.

