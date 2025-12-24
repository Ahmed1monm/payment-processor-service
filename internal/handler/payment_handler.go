package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"

	"paytabs/internal/errors"
	"paytabs/internal/service"
)

// PaymentHandler handles payment endpoints.
type PaymentHandler struct {
	paymentService service.PaymentService
}

// NewPaymentHandler creates a new payment handler.
func NewPaymentHandler(paymentService service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// CardPaymentRequest represents a card payment request.
type CardPaymentRequest struct {
	MerchantAccountID string `json:"merchant_account_id" validate:"required,uuid"`
	CardID            string `json:"card_id" validate:"required,uuid"`
	Amount            string `json:"amount" validate:"required"`
}

// PaymentResponse represents a payment response.
type PaymentResponse struct {
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

// ProcessCardPayment godoc
// @Summary Process a card payment
// @Tags payments
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CardPaymentRequest true "Payment data"
// @Success 200 {object} PaymentResponse
// @Failure 400 {object} errors.ErrorResponse
// @Failure 401 {object} errors.ErrorResponse
// @Failure 404 {object} errors.ErrorResponse
// @Failure 500 {object} errors.ErrorResponse
// @Router /payments/card [post]
func (h *PaymentHandler) ProcessCardPayment(c echo.Context) error {
	var req CardPaymentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid request body",
			Code:  "INVALID_REQUEST",
		})
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: err.Error(),
			Code:  "VALIDATION_ERROR",
		})
	}

	// Parse merchant account ID
	merchantAccountID, err := uuid.Parse(req.MerchantAccountID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid merchant_account_id",
			Code:  "INVALID_UUID",
		})
	}

	// Parse card ID
	cardID, err := uuid.Parse(req.CardID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid card_id",
			Code:  "INVALID_UUID",
		})
	}

	// Parse amount
	amount, err := decimal.NewFromString(req.Amount)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.ErrorResponse{
			Error: "invalid amount",
			Code:  "INVALID_AMOUNT",
		})
	}

	// Process payment
	payment, err := h.paymentService.ProcessCardPayment(
		c.Request().Context(),
		merchantAccountID,
		cardID,
		amount,
	)

	if err != nil {
		httpErr := errors.MapErrorToHTTP(err)
		return echo.NewHTTPError(httpErr.StatusCode, httpErr.ToErrorResponse())
	}

	status := "accepted"
	message := "Payment processed successfully"
	if payment.Status == "failed" {
		status = "failed"
		message = "Payment processing failed"
	}

	return c.JSON(http.StatusOK, PaymentResponse{
		PaymentID: payment.ID.String(),
		Status:    status,
		Message:   message,
	})
}
