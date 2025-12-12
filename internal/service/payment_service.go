package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"paytabs/internal/cache"
	"paytabs/internal/errors"
	"paytabs/internal/model"
	"paytabs/internal/repository"
)

// PaymentService handles payment processing operations.
type PaymentService interface {
	ProcessCardPayment(ctx context.Context, merchantAccountID uuid.UUID, amount decimal.Decimal, cardNumber, cardExpiry, cardCVV string) (*model.Payment, error)
}

type paymentService struct {
	accountRepo    repository.AccountRepository
	paymentRepo    repository.PaymentRepository
	paymentLogRepo repository.PaymentLogRepository
	cache          *cache.Client
	validator      *CardValidator
	// Mutex map for per-account locking
	accountMutexes sync.Map
	// Channel for async payment logging
	logChannel chan model.PaymentLog
}

// NewPaymentService creates a new payment service.
func NewPaymentService(
	accountRepo repository.AccountRepository,
	paymentRepo repository.PaymentRepository,
	paymentLogRepo repository.PaymentLogRepository,
	cache *cache.Client,
) PaymentService {
	service := &paymentService{
		accountRepo:    accountRepo,
		paymentRepo:    paymentRepo,
		paymentLogRepo: paymentLogRepo,
		cache:          cache,
		validator:      NewCardValidator(),
		logChannel:     make(chan model.PaymentLog, 100),
	}

	// Start async log worker
	go service.logWorker(context.Background())

	return service
}

// getMutex returns a mutex for a specific account ID.
func (s *paymentService) getMutex(accountID uuid.UUID) *sync.Mutex {
	key := accountID.String()
	value, _ := s.accountMutexes.LoadOrStore(key, &sync.Mutex{})
	return value.(*sync.Mutex)
}

// logWorker processes payment logs asynchronously.
func (s *paymentService) logWorker(ctx context.Context) {
	batch := make([]model.PaymentLog, 0, 10)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log, ok := <-s.logChannel:
			if !ok {
				// Channel closed, flush remaining logs
				if len(batch) > 0 {
					_ = s.paymentLogRepo.CreateBatch(ctx, batch)
				}
				return
			}
			batch = append(batch, log)
			if len(batch) >= 10 {
				_ = s.paymentLogRepo.CreateBatch(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// Flush batch periodically
			if len(batch) > 0 {
				_ = s.paymentLogRepo.CreateBatch(ctx, batch)
				batch = batch[:0]
			}
		case <-ctx.Done():
			return
		}
	}
}

// ProcessCardPayment processes a card payment for a merchant.
func (s *paymentService) ProcessCardPayment(ctx context.Context, merchantAccountID uuid.UUID, amount decimal.Decimal, cardNumber, cardExpiry, cardCVV string) (*model.Payment, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.ErrInvalidAmount
	}

	// Validate card
	if err := s.validator.ValidateCard(cardNumber, cardExpiry, cardCVV); err != nil {
		payment := s.createPaymentRecord(merchantAccountID, amount, cardNumber, cardExpiry, cardCVV, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, err
	}

	// Get mutex for this account
	mutex := s.getMutex(merchantAccountID)
	mutex.Lock()
	defer mutex.Unlock()

	// Validate merchant account exists and is active
	merchant, err := s.accountRepo.FindByIDForUpdate(ctx, merchantAccountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			payment := s.createPaymentRecord(merchantAccountID, amount, cardNumber, cardExpiry, cardCVV, model.PaymentStatusFailed)
			_ = s.paymentRepo.Create(ctx, payment)
			s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, errors.ErrAccountNotFound.Error())
			return payment, errors.ErrAccountNotFound
		}
		payment := s.createPaymentRecord(merchantAccountID, amount, cardNumber, cardExpiry, cardCVV, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, err
	}

	if !merchant.Active {
		payment := s.createPaymentRecord(merchantAccountID, amount, cardNumber, cardExpiry, cardCVV, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, errors.ErrAccountInactive.Error())
		return payment, errors.ErrAccountInactive
	}

	// Create payment record
	payment := s.createPaymentRecord(merchantAccountID, amount, cardNumber, cardExpiry, cardCVV, model.PaymentStatusPending)
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, fmt.Errorf("create payment: %w", err)
	}

	// Update merchant balance atomically
	newBalance := merchant.Balance.Add(amount)
	if err := s.accountRepo.UpdateBalance(ctx, merchantAccountID, newBalance); err != nil {
		payment.Status = model.PaymentStatusFailed
		_ = s.paymentRepo.Update(ctx, payment) // Update status
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, fmt.Sprintf("failed to update balance: %v", err))
		return payment, fmt.Errorf("update balance: %w", err)
	}

	// Mark payment as accepted
	payment.Status = model.PaymentStatusAccepted
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		// Log error but payment is already processed
		s.logPayment(ctx, payment.ID, model.PaymentStatusAccepted, "")
		return payment, nil
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("account:%s", merchantAccountID.String()))

	// Log successful payment
	s.logPayment(ctx, payment.ID, model.PaymentStatusAccepted, "")

	return payment, nil
}

// createPaymentRecord creates a payment record with masked card number.
func (s *paymentService) createPaymentRecord(merchantAccountID uuid.UUID, amount decimal.Decimal, cardNumber, cardExpiry, cardCVV string, status model.PaymentStatus) *model.Payment {
	return &model.Payment{
		MerchantAccountID: merchantAccountID,
		Amount:           amount,
		CardNumber:       s.validator.MaskCardNumber(cardNumber),
		CardExpiry:       cardExpiry,
		CardCVV:          cardCVV,
		Status:           status,
	}
}

// logPayment logs a payment attempt asynchronously.
func (s *paymentService) logPayment(ctx context.Context, paymentID uuid.UUID, status model.PaymentStatus, errorMessage string) {
	log := model.PaymentLog{
		PaymentID:    paymentID,
		Status:       status,
		ErrorMessage: errorMessage,
	}

	// Send to async log channel (non-blocking)
	select {
	case s.logChannel <- log:
	default:
		// Channel full, log synchronously as fallback
		_ = s.paymentLogRepo.Create(ctx, &log)
	}
}

