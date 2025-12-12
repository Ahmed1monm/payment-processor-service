package main

import (
	"log"
	"net/http"

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
// @host localhost:8080
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

	// Run migrations for all models
	if err := gormDB.AutoMigrate(
		&model.User{},
		&model.Account{},
		&model.Payment{},
		&model.PaymentLog{},
		&model.Transfer{},
	); err != nil {
		log.Fatalf("auto-migrate: %v", err)
	}

	cacheClient := cache.New(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)

	// Initialize repositories
	userRepo := repository.NewUserRepository(gormDB)
	accountRepo := repository.NewAccountRepository(gormDB)
	paymentRepo := repository.NewPaymentRepository(gormDB)
	paymentLogRepo := repository.NewPaymentLogRepository(gormDB)
	transferRepo := repository.NewTransferRepository(gormDB)

	// Initialize auth components
	jwtService := auth.NewJWTService(cfg.JWTSecret)
	tokenStore := auth.NewTokenStore(cacheClient)

	// Initialize services
	userService := service.NewUserService(userRepo, cacheClient)
	authService := service.NewAuthService(userRepo, jwtService, tokenStore)
	accountService := service.NewAccountService(accountRepo, cacheClient)
	paymentService := service.NewPaymentService(accountRepo, paymentRepo, paymentLogRepo, cacheClient)
	transferService := service.NewTransferService(accountRepo, transferRepo, cacheClient)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	authHandler := handler.NewAuthHandler(authService)
	accountHandler := handler.NewAccountHandler(accountService)
	paymentHandler := handler.NewPaymentHandler(paymentService)
	transferHandler := handler.NewTransferHandler(transferService)
	seedHandler := handler.NewSeedHandler(accountService)

	// Register routes
	router.Register(
		e,
		cfg,
		userHandler,
		authHandler,
		accountHandler,
		paymentHandler,
		transferHandler,
		seedHandler,
	)

	addr := ":" + cfg.ServerPort
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server start: %v", err)
	}
}
