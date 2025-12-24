package main

import (
	"log"
	"net/http"
	"os"

	_ "paytabs/docs" // swagger docs

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"paytabs/internal/auth"
	"paytabs/internal/cache"
	"paytabs/internal/config"
	"paytabs/internal/db"
	"paytabs/internal/handler"
	"paytabs/internal/model"
	"paytabs/internal/repository"
	"paytabs/internal/router"
	"paytabs/internal/service"
)

// @title Payment Processor API
// @version 1.0
// @description Payment processor API with card payments, account transfers, and JWT authentication.
// @host localhost:5000
// @BasePath /api
// @schemes http
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg := config.Load()

	e := echo.New()
	e.Use(middleware.RequestID())

	gormDB, err := db.NewMySQL(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}

	// Drop tables if RESET_DB environment variable is set
	if os.Getenv("RESET_DB") == "true" {
		log.Println("RESET_DB=true detected, dropping all tables...")
		tables := []interface{}{
			&model.Transfer{},
			&model.PaymentLog{},
			&model.Payment{},
			&model.Card{},
			&model.Account{},
		}
		for _, table := range tables {
			if err := gormDB.Migrator().DropTable(table); err != nil {
				log.Printf("Warning: Failed to drop table (may not exist): %v", err)
			}
		}
		log.Println("Tables dropped")
	}

	// Run migrations for all models
	if err := gormDB.AutoMigrate(
		&model.Account{},
		&model.Card{},
		&model.Payment{},
		&model.PaymentLog{},
		&model.Transfer{},
	); err != nil {
		log.Fatalf("auto-migrate: %v", err)
	}

	cacheClient := cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)

	// Initialize repositories
	accountRepo := repository.NewAccountRepository(gormDB)
	cardRepo := repository.NewCardRepository(gormDB)
	paymentRepo := repository.NewPaymentRepository(gormDB)
	paymentLogRepo := repository.NewPaymentLogRepository(gormDB)
	transferRepo := repository.NewTransferRepository(gormDB)

	// Initialize auth components
	jwtService := auth.NewJWTService(cfg.JWTSecret)
	tokenStore := auth.NewTokenStore(cacheClient)

	// Initialize services
	authService := service.NewAuthService(accountRepo, jwtService, tokenStore)
	accountService := service.NewAccountService(accountRepo, cardRepo, cacheClient)
	paymentService := service.NewPaymentService(accountRepo, cardRepo, paymentRepo, paymentLogRepo, cacheClient)
	transferService := service.NewTransferService(cardRepo, transferRepo, cacheClient)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	accountHandler := handler.NewAccountHandler(accountService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	transferHandler := handler.NewTransferHandler(transferService)
	seedHandler := handler.NewSeedHandler(accountService)

	// Register routes
	router.Register(
		e,
		cfg,
		authHandler,
		accountHandler,
		paymentHandler,
		transferHandler,
		seedHandler,
	)

	// Log swagger full path
	var swaggerURL string
	if cfg.SwaggerHost != "" {
		// SwaggerHost may already include scheme (http:// or https://)
		if len(cfg.SwaggerHost) >= 7 && cfg.SwaggerHost[:7] == "http://" {
			swaggerURL = cfg.SwaggerHost + "/api-docs"
		} else if len(cfg.SwaggerHost) >= 8 && cfg.SwaggerHost[:8] == "https://" {
			swaggerURL = cfg.SwaggerHost + "/api-docs"
		} else {
			swaggerURL = "http://" + cfg.SwaggerHost + "/api-docs"
		}
	} else {
		// For docker-compose: container listens on 8080, mapped to 5000 externally
		// Use 5000 as the external port for swagger URL
		swaggerURL = "http://localhost:5000/api-docs"
	}
	log.Printf("Swagger documentation available at: %s", swaggerURL)

	addr := ":" + cfg.ServerPort
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server start: %v", err)
	}
}
