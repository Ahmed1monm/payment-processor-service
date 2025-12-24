# Payment Processor System

A production-ready RESTful API for processing card payments and card-to-card transfers, built with Go, Echo framework, MySQL, and Redis. Features account-based authentication with merchant/user distinction and card-based balance management.

## Features

- **Card Payment Processing**: Process payments using card balances with merchant account validation
- **Card-to-Card Transfers**: Secure card-to-card money transfers with atomic balance updates
- **JWT Authentication**: Access and refresh token-based authentication with Redis storage
- **Account Management**: Account registration with merchant/user distinction, card balance queries
- **Card Management**: Cards linked to accounts with individual balances
- **Payment Logging**: All payment attempts are logged regardless of success/failure
- **Concurrency Safety**: Mutex-based locking for card operations, transaction-based transfers
- **Comprehensive Testing**: Unit and integration tests
- **API Documentation**: Swagger/OpenAPI documentation available at `/api-docs`

## Architecture

The system follows a clean architecture pattern with clear separation of concerns:

```
internal/
├── auth/          # JWT service and token storage
├── cache/         # Redis cache wrapper
├── config/        # Configuration management
├── db/            # Database connection
├── errors/        # Custom error types
├── handler/       # HTTP handlers (presentation layer)
├── model/         # Domain models
├── repository/    # Data access layer
├── router/        # Route registration
└── service/       # Business logic layer
```

### Design Decisions

1. **Decimal Precision**: Uses `shopspring/decimal` for financial amounts to avoid floating-point precision issues
2. **UUIDs**: All entity IDs use UUIDs for better distributed system compatibility
3. **Password Hashing**: bcrypt with cost factor 10 for secure password storage
4. **Token Storage**: Refresh tokens stored in Redis with TTL matching token expiry
5. **Concurrency**: 
   - Per-account mutexes for payment processing
   - Database transactions with row-level locking (`SELECT ... FOR UPDATE`) for transfers
   - Async channel-based payment logging
6. **Error Handling**: Consistent error response format with HTTP status codes
7. **Caching**: Account data cached in Redis with 5-minute TTL

## Prerequisites

- Go 1.22 or higher
- MySQL 8.0 or higher
- Redis 7.0 or higher
- Docker and Docker Compose (optional, for containerized setup)

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository and navigate to the project directory:
   ```bash
   cd /path/to/go
   ```

2. **Reset database** (optional, if you want a fresh start):
   ```bash
   # This will remove all data volumes and start fresh
   docker-compose down -v
   ```

3. Start all services (MySQL, Redis, and API):
   ```bash
   docker-compose up --build
   ```
   - The seed script will automatically drop existing tables and recreate them
   - 500 accounts will be seeded from the external API

4. The API will be available at `http://localhost:5000`
5. Swagger UI: `http://localhost:5000/api-docs`

### Manual Setup

1. **Copy environment file**:
   ```bash
   cp .env.example .env
   ```
   Then edit `.env` with your configuration values.

2. **Set environment variables** (or use defaults):
   ```bash
   export SERVER_PORT="5000"
   export MYSQL_DSN="user:password@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
   export REDIS_ADDR="localhost:6379"
   export REDIS_DB="0"
   export REDIS_PASSWORD=""  # Optional
   export JWT_SECRET="your-secret-key-here"  # Change this!
   export RESET_DB="true"  # Optional: Drop and recreate tables on startup
   ```

3. **Start MySQL and Redis** (if not using Docker):
   - MySQL should be running on port 3306
   - Redis should be running on port 6379

4. **Run the server**:
   ```bash
   cd /path/to/go
   go run ./cmd/server
   ```

5. **Seed accounts** (optional):
   ```bash
   # Option 1: Using the standalone CLI script (recommended)
   # This will drop existing tables and recreate them with fresh data
   go run ./cmd/seed
   # Or build and run:
   go build -o seed ./cmd/seed
   ./seed
   
   # Option 2: Using the HTTP endpoint
   curl http://localhost:5000/api/seed/accounts
   ```

## API Endpoints

### Authentication (Public)

- `POST /api/auth/register` - Register a new account
  ```json
  {
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe",
    "is_merchant": false
  }
  ```
  - Creates an `Account` record (not a separate user table)
  - `is_merchant`: Set to `true` for merchant accounts, `false` for regular users

- `POST /api/auth/login` - Login and get tokens
  ```json
  {
    "email": "user@example.com",
    "password": "password123"
  }
  ```
  Returns: `access_token` and `refresh_token`

- `POST /api/auth/refresh` - Refresh access token
  ```json
  {
    "refresh_token": "your-refresh-token"
  }
  ```

- `POST /api/auth/logout` - Logout (invalidate refresh token)
  ```json
  {
    "refresh_token": "your-refresh-token"
  }
  ```

### Account Management (Protected)

- `GET /api/accounts/{id}/balance` - Get total balance across all cards for an account
  - Requires: `Authorization: Bearer <access_token>`
  - Returns the sum of balances from all active cards linked to the account

### Payments (Protected)

- `POST /api/payments/card` - Process a card payment
  ```json
  {
    "merchant_account_id": "uuid-here",
    "card_id": "uuid-here",
    "amount": "100.50"
  }
  ```
  - Requires: `Authorization: Bearer <access_token>`
  - `merchant_account_id`: Must be an account with `is_merchant: true`
  - `card_id`: The card to deduct payment from (card must exist and be active)
  - Deducts amount from the card's balance
  - Logs all payment attempts

### Transfers (Protected)

- `POST /api/transfers` - Transfer money between cards
  ```json
  {
    "source_card_id": "uuid-here",
    "destination_card_id": "uuid-here",
    "amount": "50.00"
  }
  ```
  - Requires: `Authorization: Bearer <access_token>`
  - Transfers balance from one card to another
  - Validates both cards exist and are active
  - Checks sufficient balance on source card
  - Atomic balance updates using database transactions

### Seed Data (Public)

- `GET /api/seed/accounts` - Fetch and seed accounts from external API
  - Fetches accounts from: https://gist.githubusercontent.com/paytabscom/...
  - Alternatively, use the standalone CLI script: `go run ./cmd/seed`

## Testing

### Running Unit Tests

```bash
go test ./internal/service/...
```

### Running Integration Tests

```bash
go test ./internal/handler/...
```

### Running All Tests

```bash
go test ./...
```

### Test Coverage

```bash
go test -cover ./...
```

## Swagger Documentation

1. **View Swagger UI**: Navigate to `http://localhost:5000/api-docs`

2. **Regenerate Swagger Docs** (if you modify handlers):
   ```bash
   cd cmd/server
   swag init -g main.go -o ../docs
   ```

## Example Usage Flow

1. **Register an account** (merchant or user):
   ```bash
   # Register as merchant
   curl -X POST http://localhost:5000/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"merchant@example.com","password":"secure123","name":"Merchant","is_merchant":true}'
   
   # Register as regular user
   curl -X POST http://localhost:5000/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"secure123","name":"User","is_merchant":false}'
   ```

2. **Login to get tokens**:
   ```bash
   curl -X POST http://localhost:5000/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"merchant@example.com","password":"secure123"}'
   ```

3. **Seed accounts** (creates accounts from external API):
   ```bash
   # Using CLI script (recommended - drops and recreates tables)
   go run ./cmd/seed
   
   # Or using HTTP endpoint
   curl http://localhost:5000/api/seed/accounts
   ```

4. **Create a card** (cards must be created separately - balance is stored on cards):
   ```bash
   # Note: Card creation endpoint would need to be implemented
   # Cards are linked to accounts and have their own balance
   ```

5. **Process a card payment**:
   ```bash
   curl -X POST http://localhost:5000/api/payments/card \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "merchant_account_id": "merchant-account-uuid",
       "card_id": "card-uuid-here",
       "amount": "100.00"
     }'
   ```

6. **Check account balance** (sum of all card balances):
   ```bash
   curl http://localhost:5000/api/accounts/ACCOUNT_UUID/balance \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
   ```

7. **Transfer money between cards**:
   ```bash
   curl -X POST http://localhost:5000/api/transfers \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "source_card_id": "source-card-uuid",
       "destination_card_id": "dest-card-uuid",
       "amount": "50.00"
     }'
   ```

## Concurrency & Safety

### Payment Processing
- Uses per-card mutexes to prevent concurrent balance updates
- Validates merchant account and card status before processing
- Deducts payment amount from card balance
- Row-level locking (`SELECT ... FOR UPDATE`) ensures data consistency
- All payment attempts are logged asynchronously via channel-based worker

### Transfer Processing
- Database transactions ensure atomic balance updates
- Both source and destination cards are locked during transfer
- Validates card status and sufficient balance
- Rollback on any error prevents partial updates

### Token Management
- Refresh tokens stored in Redis with TTL
- Access tokens have 15-minute expiry
- Refresh tokens have 7-day expiry

## Error Handling

All errors follow a consistent format:
```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

Common error codes:
- `ACCOUNT_NOT_FOUND` - Account doesn't exist
- `CARD_NOT_FOUND` - Card doesn't exist
- `ACCOUNT_INACTIVE` - Account is not active
- `INSUFFICIENT_BALANCE` - Insufficient funds on card
- `INVALID_CARD` - Card validation failed or card is inactive
- `INVALID_AMOUNT` - Invalid payment/transfer amount
- `INVALID_CREDENTIALS` - Authentication failed
- `INVALID_REFRESH_TOKEN` - Refresh token invalid/expired
- `ACCOUNT_ALREADY_EXISTS` - Account with email already exists

## Database Schema

The system uses the following main tables:

### `accounts`
- `id` (UUID, Primary Key) - Account identifier
- `name` (String) - Account name
- `email` (String, Unique) - Account email (used for authentication)
- `password_hash` (String) - Bcrypt hashed password
- `is_merchant` (Boolean) - Whether account is a merchant
- `active` (Boolean) - Account status
- `created_at`, `updated_at` (Timestamps)
- `deleted_at` (Soft delete)

### `cards`
- `id` (UUID, Primary Key) - Card identifier
- `account_id` (UUID, Foreign Key → accounts.id) - Owner account
- `card_number` (String) - Masked card number
- `card_expiry` (String) - Card expiry (MM/YY format)
- `balance` (Decimal) - Card balance (financial amounts stored here)
- `active` (Boolean) - Card status
- `created_at`, `updated_at` (Timestamps)
- `deleted_at` (Soft delete)

### `payments`
- `id` (UUID, Primary Key) - Payment identifier
- `merchant_account_id` (UUID, Foreign Key → accounts.id) - Merchant receiving payment
- `card_id` (UUID, Foreign Key → cards.id) - Card used for payment
- `amount` (Decimal) - Payment amount
- `status` (Enum: pending, accepted, failed)
- `created_at`, `updated_at` (Timestamps)
- `deleted_at` (Soft delete)

### `transfers`
- `id` (UUID, Primary Key) - Transfer identifier
- `source_card_id` (UUID, Foreign Key → cards.id) - Source card
- `destination_card_id` (UUID, Foreign Key → cards.id) - Destination card
- `amount` (Decimal) - Transfer amount
- `status` (Enum: pending, completed, failed)
- `error_message` (String, Optional) - Error details if failed
- `created_at`, `updated_at` (Timestamps)
- `deleted_at` (Soft delete)

### `payment_logs`
- `id` (UUID, Primary Key) - Log identifier
- `payment_id` (UUID, Foreign Key → payments.id) - Related payment
- `status` (Enum) - Payment status at log time
- `error_message` (String, Optional) - Error details
- `created_at` (Timestamp)

**Key Design Points:**
- All tables use UUIDs as primary keys
- Balance is stored on `cards`, not `accounts`
- Accounts can have multiple cards
- Payments and transfers operate on card balances
- All tables include `created_at`, `updated_at` timestamps
- Soft deletes supported via `deleted_at`

## Security Considerations

1. **Passwords**: Hashed using bcrypt (cost factor 10), stored in accounts table
2. **Tokens**: JWT tokens with HMAC-SHA256 signing
3. **Card Data**: Card numbers stored as masked (only last 4 digits visible)
4. **Input Validation**: All inputs validated using go-playground/validator
5. **SQL Injection**: Protected by GORM parameterized queries
6. **Authentication**: Registration creates accounts directly (no separate users table)
7. **Merchant Validation**: Payments require merchant accounts (`is_merchant: true`)
8. **Rate Limiting**: Consider adding rate limiting middleware for production

## Production Recommendations

1. **Environment Variables**: Use a secrets manager (e.g., AWS Secrets Manager, HashiCorp Vault)
2. **Database**: Use connection pooling and read replicas for scaling
3. **Redis**: Use Redis Cluster for high availability
4. **Monitoring**: Add logging (e.g., structured logging with correlation IDs)
5. **Metrics**: Add Prometheus metrics for monitoring
6. **Rate Limiting**: Implement rate limiting to prevent abuse
7. **HTTPS**: Always use HTTPS in production
8. **CORS**: Configure CORS appropriately for your frontend

## License

This project is a demonstration/assessment project.
