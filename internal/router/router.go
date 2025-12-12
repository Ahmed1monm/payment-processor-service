package router

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v4"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"

	"paytabs/internal/config"
	"paytabs/internal/handler"
)

// Register wires routes and middleware.
func Register(
	e *echo.Echo,
	cfg *config.Config,
	userHandler *handler.UserHandler,
	authHandler *handler.AuthHandler,
	accountHandler *handler.AccountHandler,
	paymentHandler *handler.PaymentHandler,
	transferHandler *handler.TransferHandler,
	seedHandler *handler.SeedHandler,
) {
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Add validator
	e.Validator = &CustomValidator{validator: validator.New()}

	if cfg.SwaggerHost != "" {
		// Swag uses this for server URL in docs when set.
	}

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	api := e.Group("/api")

	// Public routes
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)
	api.POST("/auth/refresh", authHandler.Refresh)
	api.POST("/auth/logout", authHandler.Logout)
	api.GET("/seed/accounts", seedHandler.SeedAccounts)

	// Legacy user routes (optional, can be removed)
	api.GET("/users", userHandler.ListUsers)
	api.GET("/users/:id", userHandler.GetUser)
	api.POST("/users", userHandler.CreateUser)

	// Secured routes (require JWT authentication)
	secured := api.Group("", echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(cfg.JWTSecret),
		TokenLookup: "header:" + echo.HeaderAuthorization,
	}))

	secured.GET("/me", func(c echo.Context) error {
		token, ok := c.Get("user").(*jwt.Token)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		claims, _ := token.Claims.(jwt.MapClaims)
		return c.JSON(http.StatusOK, echo.Map{"token_claims": claims})
	})

	// Account routes
	secured.GET("/accounts/:id/balance", accountHandler.GetBalance)

	// Payment routes
	secured.POST("/payments/card", paymentHandler.ProcessCardPayment)

	// Transfer routes
	secured.POST("/transfers", transferHandler.ProcessTransfer)
}

// CustomValidator wraps validator for Echo.
type CustomValidator struct {
	validator *validator.Validate
}

// Validate implements echo.Validator interface.
func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
