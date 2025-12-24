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
	ProcessCardPayment(ctx context.Context, merchantAccountID uuid.UUID, cardID uuid.UUID, amount decimal.Decimal) (*model.Payment, error)
}

type paymentService struct {
	accountRepo    repository.AccountRepository
	cardRepo       repository.CardRepository
	paymentRepo    repository.PaymentRepository
	paymentLogRepo repository.PaymentLogRepository
	cache          *cache.Client
	// Mutex map for per-card locking
	cardMutexes sync.Map
	// Channel for async payment logging
	logChannel chan model.PaymentLog
}

// NewPaymentService creates a new payment service.
func NewPaymentService(
	accountRepo repository.AccountRepository,
	cardRepo repository.CardRepository,
	paymentRepo repository.PaymentRepository,
	paymentLogRepo repository.PaymentLogRepository,
	cache *cache.Client,
) PaymentService {
	service := &paymentService{
		accountRepo:    accountRepo,
		cardRepo:       cardRepo,
		paymentRepo:    paymentRepo,
		paymentLogRepo: paymentLogRepo,
		cache:          cache,
		logChannel:     make(chan model.PaymentLog, 100),
	}

	// Start async log worker
	go service.logWorker(context.Background())

	return service
}

// getMutex returns a mutex for a specific card ID.
func (s *paymentService) getMutex(cardID uuid.UUID) *sync.Mutex {
	key := cardID.String()
	value, _ := s.cardMutexes.LoadOrStore(key, &sync.Mutex{})
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
func (s *paymentService) ProcessCardPayment(ctx context.Context, merchantAccountID uuid.UUID, cardID uuid.UUID, amount decimal.Decimal) (*model.Payment, error) {
	// Validate amount
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, errors.ErrInvalidAmount
	}

	// Get mutex for this card
	mutex := s.getMutex(cardID)
	mutex.Lock()
	defer mutex.Unlock()

	// Validate merchant account exists and is active
	merchant, err := s.accountRepo.FindByID(ctx, merchantAccountID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
			_ = s.paymentRepo.Create(ctx, payment)
			s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, errors.ErrAccountNotFound.Error())
			return payment, errors.ErrAccountNotFound
		}
		payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, err
	}

	if !merchant.Active {
		payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, errors.ErrAccountInactive.Error())
		return payment, errors.ErrAccountInactive
	}

	if !merchant.IsMerchant {
		payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, "account is not a merchant")
		return payment, fmt.Errorf("account is not a merchant")
	}

	// Validate card exists and is active
	card, err := s.cardRepo.FindByIDForUpdate(ctx, cardID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
			_ = s.paymentRepo.Create(ctx, payment)
			s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, "card not found")
			return payment, fmt.Errorf("card not found")
		}
		payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, err
	}

	if !card.Active {
		payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusFailed)
		_ = s.paymentRepo.Create(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, "card is not active")
		return payment, fmt.Errorf("card is not active")
	}

	// Create payment record
	payment := s.createPaymentRecord(merchantAccountID, cardID, amount, model.PaymentStatusPending)
	if err := s.paymentRepo.Create(ctx, payment); err != nil {
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, err.Error())
		return payment, fmt.Errorf("create payment: %w", err)
	}

	// Update card balance atomically (deduct from card)
	newBalance := card.Balance.Sub(amount)
	if newBalance.LessThan(decimal.Zero) {
		payment.Status = model.PaymentStatusFailed
		_ = s.paymentRepo.Update(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, errors.ErrInsufficientBalance.Error())
		return payment, errors.ErrInsufficientBalance
	}

	if err := s.cardRepo.UpdateBalance(ctx, cardID, newBalance); err != nil {
		payment.Status = model.PaymentStatusFailed
		_ = s.paymentRepo.Update(ctx, payment)
		s.logPayment(ctx, payment.ID, model.PaymentStatusFailed, fmt.Sprintf("failed to update balance: %v", err))
		return payment, fmt.Errorf("update balance: %w", err)
	}

	// Mark payment as accepted
	payment.Status = model.PaymentStatusAccepted
	if err := s.paymentRepo.Update(ctx, payment); err != nil {
		s.logPayment(ctx, payment.ID, model.PaymentStatusAccepted, "")
		return payment, nil
	}

	// Invalidate cache
	_ = s.cache.Delete(ctx, fmt.Sprintf("card:%s", cardID.String()))

	// Log successful payment
	s.logPayment(ctx, payment.ID, model.PaymentStatusAccepted, "")

	return payment, nil
}

// createPaymentRecord creates a payment record.
func (s *paymentService) createPaymentRecord(merchantAccountID uuid.UUID, cardID uuid.UUID, amount decimal.Decimal, status model.PaymentStatus) *model.Payment {
	return &model.Payment{
		MerchantAccountID: merchantAccountID,
		CardID:            cardID,
		Amount:            amount,
		Status:            status,
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

