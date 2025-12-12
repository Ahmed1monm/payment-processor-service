# Payment Processor System

A production-ready RESTful API for processing card payments and account-to-account transfers, built with Go, Echo framework, MySQL, and Redis.

## Features

- **Card Payment Processing**: Accept card payments with Luhn algorithm validation
- **Account Transfers**: Secure account-to-account money transfers with atomic balance updates
- **JWT Authentication**: Access and refresh token-based authentication with Redis storage
- **Account Management**: Account balance queries and external API seeding
- **Payment Logging**: All payment attempts are logged regardless of success/failure
- **Concurrency Safety**: Mutex-based locking for payments, transaction-based transfers
- **Comprehensive Testing**: Unit and integration tests
- **API Documentation**: Swagger/OpenAPI documentation

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
   cd E:\side-projects\go
   ```

2. Start all services (MySQL, Redis, and API):
   ```powershell
   docker-compose up --build
   ```

3. The API will be available at `http://localhost:8080`
4. Swagger UI: `http://localhost:8080/swagger/index.html`

### Manual Setup

1. **Set environment variables** (or use defaults):
   ```powershell
   $env:SERVER_PORT="8080"
   $env:MYSQL_DSN="user:password@tcp(localhost:3306)/app?charset=utf8mb4&parseTime=True&loc=Local"
   $env:REDIS_ADDR="localhost:6379"
   $env:REDIS_DB="0"
   $env:REDIS_PASSWORD=""  # Optional
   $env:JWT_SECRET="your-secret-key-here"  # Change this!
   ```

2. **Start MySQL and Redis** (if not using Docker):
   - MySQL should be running on port 3306
   - Redis should be running on port 6379

3. **Run the server**:
   ```powershell
   cd E:\side-projects\go
   go run ./cmd/server
   ```

4. **Seed accounts** (optional):
   ```powershell
   # Option 1: Using the standalone CLI script (recommended)
   go run ./cmd/seed
   # Or build and run:
   go build -o seed.exe ./cmd/seed
   .\seed.exe
   
   # Option 2: Using the HTTP endpoint
   curl http://localhost:8080/api/seed/accounts
   ```

## API Endpoints

### Authentication (Public)

- `POST /api/auth/register` - Register a new user
  ```json
  {
    "email": "user@example.com",
    "password": "password123",
    "name": "John Doe"
  }
  ```

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

- `GET /api/accounts/{id}/balance` - Get account balance
  - Requires: `Authorization: Bearer <access_token>`

### Payments (Protected)

- `POST /api/payments/card` - Process a card payment
  ```json
  {
    "merchant_account_id": "uuid-here",
    "amount": "100.50",
    "card_number": "4111111111111111",
    "card_expiry": "12/25",
    "card_cvv": "123"
  }
  ```
  - Requires: `Authorization: Bearer <access_token>`
  - Validates card using Luhn algorithm
  - Updates merchant balance atomically
  - Logs all payment attempts

### Transfers (Protected)

- `POST /api/transfers` - Transfer money between accounts
  ```json
  {
    "source_account_id": "uuid-here",
    "destination_account_id": "uuid-here",
    "amount": "50.00"
  }
  ```
  - Requires: `Authorization: Bearer <access_token>`
  - Validates both accounts exist and are active
  - Checks sufficient balance
  - Atomic balance updates using database transactions

### Seed Data (Public)

- `GET /api/seed/accounts` - Fetch and seed accounts from external API
  - Fetches accounts from: https://gist.githubusercontent.com/paytabscom/...
  - Alternatively, use the standalone CLI script: `go run ./cmd/seed`

## Testing

### Running Unit Tests

```powershell
go test ./internal/service/...
```

### Running Integration Tests

```powershell
go test ./internal/handler/...
```

### Running All Tests

```powershell
go test ./...
```

### Test Coverage

```powershell
go test -cover ./...
```

## Swagger Documentation

1. **View Swagger UI**: Navigate to `http://localhost:8080/swagger/index.html`

2. **Regenerate Swagger Docs** (if you modify handlers):
   ```powershell
   cd E:\side-projects\go\cmd\server
   $env:PATH="C:\Program Files\Go\bin;" + $env:PATH
   swag init -g main.go -o ..\docs
   ```

## Example Usage Flow

1. **Register a user**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{"email":"merchant@example.com","password":"secure123","name":"Merchant"}'
   ```

2. **Login to get tokens**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"merchant@example.com","password":"secure123"}'
   ```

3. **Seed accounts** (get merchant account IDs):
   ```bash
   # Using CLI script (recommended)
   go run ./cmd/seed
   
   # Or using HTTP endpoint
   curl http://localhost:8080/api/seed/accounts
   ```

4. **Process a card payment**:
   ```bash
   curl -X POST http://localhost:8080/api/payments/card \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "merchant_account_id": "account-uuid-here",
       "amount": "100.00",
       "card_number": "4111111111111111",
       "card_expiry": "12/25",
       "card_cvv": "123"
     }'
   ```

5. **Check account balance**:
   ```bash
   curl http://localhost:8080/api/accounts/ACCOUNT_UUID/balance \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
   ```

6. **Transfer money**:
   ```bash
   curl -X POST http://localhost:8080/api/transfers \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
     -d '{
       "source_account_id": "source-uuid",
       "destination_account_id": "dest-uuid",
       "amount": "50.00"
     }'
   ```

## Concurrency & Safety

### Payment Processing
- Uses per-account mutexes to prevent concurrent balance updates
- Row-level locking (`SELECT ... FOR UPDATE`) ensures data consistency
- All payment attempts are logged asynchronously via channel-based worker

### Transfer Processing
- Database transactions ensure atomic balance updates
- Both source and destination accounts are locked during transfer
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
- `ACCOUNT_INACTIVE` - Account is not active
- `INSUFFICIENT_BALANCE` - Insufficient funds
- `INVALID_CARD` - Card validation failed
- `INVALID_AMOUNT` - Invalid payment/transfer amount
- `INVALID_CREDENTIALS` - Authentication failed
- `INVALID_REFRESH_TOKEN` - Refresh token invalid/expired

## Database Schema

The system uses the following main tables:
- `users` - User accounts with authentication
- `accounts` - Merchant/user accounts with balances
- `payments` - Card payment records
- `payment_logs` - Payment attempt logs
- `transfers` - Account-to-account transfers

All tables use UUIDs as primary keys and include `created_at`, `updated_at` timestamps.

## Security Considerations

1. **Passwords**: Hashed using bcrypt (cost factor 10)
2. **Tokens**: JWT tokens with HMAC-SHA256 signing
3. **Card Data**: Card numbers are masked before storage (only last 4 digits)
4. **Input Validation**: All inputs validated using go-playground/validator
5. **SQL Injection**: Protected by GORM parameterized queries
6. **Rate Limiting**: Consider adding rate limiting middleware for production

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
