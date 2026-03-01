package account

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kruangsuriya/tianfu-bank/services/account-service/core/domain/account"
)

type AccountHttpHandler struct {
	accountSvc account.AccountService
}

func NewAccountHttpHandler(accountSvc account.AccountService) *AccountHttpHandler {
	return &AccountHttpHandler{accountSvc: accountSvc}
}

func (h *AccountHttpHandler) CreateAccount(c *fiber.Ctx) error {
	var req CreateAccountRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	cmd := account.CreateAccountCmd{
		UserId:      req.UserId,
		AccountType: req.AccountType,
	}

	acc, err := h.accountSvc.CreateAccount(c.Context(), &cmd)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(acc)
}

func (h *AccountHttpHandler) GetAccount(c *fiber.Ctx) error {
	id := c.Params("accountId")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "accountId is required"})
	}

	q := account.GetAccountQuery{AccountId: id}

	acc, err := h.accountSvc.GetAccountById(c.Context(), &q)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(acc)
}
