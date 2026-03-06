package transaction

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	domain "github.com/zhunismp/tianfu-bank/services/transaction-service/core/domain/transaction"
	"github.com/zhunismp/tianfu-bank/shared/apperror"
)

type TransactionHttpHandler struct {
	txnSvc domain.TransactionService
}

func NewTransactionHttpHandler(txnSvc domain.TransactionService) *TransactionHttpHandler {
	return &TransactionHttpHandler{txnSvc: txnSvc}
}

func (h *TransactionHttpHandler) Deposit(c *fiber.Ctx) error {
	idempotencyKey := c.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "X-Idempotency-Key header is required"})
	}

	var req DepositRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cmd := &domain.DepositCmd{
		AccountID:      req.AccountID,
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
	}

	event, err := h.txnSvc.Deposit(c.Context(), cmd)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toTransactionResponse(event))
}

func (h *TransactionHttpHandler) Withdraw(c *fiber.Ctx) error {
	idempotencyKey := c.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "X-Idempotency-Key header is required"})
	}

	var req WithdrawRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cmd := &domain.WithdrawCmd{
		AccountID:      req.AccountID,
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
	}

	event, err := h.txnSvc.Withdraw(c.Context(), cmd)
	if err != nil {
		return handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toTransactionResponse(event))
}

func (h *TransactionHttpHandler) Transfer(c *fiber.Ctx) error {
	idempotencyKey := c.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "X-Idempotency-Key header is required"})
	}

	var req TransferRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cmd := &domain.TransferCmd{
		SourceAccountID:      req.SourceAccountID,
		DestinationAccountID: req.DestinationAccountID,
		Amount:               req.Amount,
		IdempotencyKey:       idempotencyKey,
	}

	events, err := h.txnSvc.Transfer(c.Context(), cmd)
	if err != nil {
		return handleError(c, err)
	}

	// events may be nil for idempotent replays
	if events == nil || len(events) < 2 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "transfer already processed"})
	}

	resp := TransferResponse{
		SourceEvent:      toTransactionResponse(events[0]),
		DestinationEvent: toTransactionResponse(events[1]),
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func (h *TransactionHttpHandler) GetHistory(c *fiber.Ctx) error {
	accountID := c.Params("accountId")
	if accountID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "accountId is required"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	query := &domain.GetHistoryQuery{
		AccountID: accountID,
		Limit:     limit,
		Offset:    offset,
	}

	items, err := h.txnSvc.GetHistory(c.Context(), query)
	if err != nil {
		return handleError(c, err)
	}

	txns := make([]TransactionResponse, len(items))
	for i, item := range items {
		txns[i] = TransactionResponse{
			ID:           item.ID,
			AccountID:    accountID,
			EventType:    item.EventType,
			Amount:       item.Amount.String(),
			BalanceAfter: item.BalanceAfter.String(),
			CreatedAt:    item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	return c.JSON(HistoryResponse{
		AccountID:    accountID,
		Transactions: txns,
	})
}

func handleError(c *fiber.Ctx, err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return c.Status(apperror.MapToHTTPStatus(appErr.Code)).JSON(fiber.Map{
			"code":    appErr.Code,
			"message": appErr.Message,
		})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"code":    apperror.ErrCodeInternal,
		"message": "an internal error occurred",
	})
}

func toTransactionResponse(event *domain.TransactionEvent) *TransactionResponse {
	if event == nil {
		return nil
	}
	return &TransactionResponse{
		ID:        event.ID,
		AccountID: event.AccountID,
		EventType: event.EventType,
		Amount:    event.Amount.String(),
		CreatedAt: event.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
