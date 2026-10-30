package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/logger"
)

type CustomerHandler struct {
	svc       *customer.Service
	validator validation.Validator
	logger    logger.Logger
}

func NewCustomerHandler(svc *customer.Service, validator validation.Validator, log logger.Logger) *CustomerHandler {
	return &CustomerHandler{
		svc:       svc,
		validator: validator,
		logger:    log.With("component", "http.customer"),
	}
}

func (h *CustomerHandler) List(c fiber.Ctx) error {
	customers, err := h.svc.List(c.Context())
	if err != nil {
		switch {
		case errors.Is(err, customer.ErrCustomerNotFound):
			return fiber.NewError(fiber.StatusNotFound, "customer not found.")
		default:
			return fmt.Errorf("retrieve customer: %w", err)
		}
	}

	return c.JSON(
		response.New(response.ToCustomers(customers)),
	)
}

func (h *CustomerHandler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid id.")
	}

	cust, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, customer.ErrCustomerNotFound):
			return fiber.NewError(fiber.StatusNotFound, "Customer not found.")
		default:
			return fmt.Errorf("get customer by id: %w", err)
		}
	}

	return c.JSON(
		response.New(response.ToCustomer(*cust)),
	)
}

func (h *CustomerHandler) Create(c fiber.Ctx) error {
	var req request.CreateCustomer

	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("create customer bind request body: %w", err)
	}

	if err := h.validator.ValidateStruct(c.Context(), req); err != nil {
		return fmt.Errorf("create customer validation: %w", err)
	}

	reqCust := req.ToCustomer()

	cust, err := h.svc.Create(c.Context(), reqCust)
	if err != nil {
		switch {
		case errors.Is(err, customer.ErrEmailAlreadyTaken):
			return fiber.NewError(fiber.StatusConflict, "email already taken.")
		default:
			return fmt.Errorf("create customer: %w", err)
		}
	}

	return c.Status(http.StatusCreated).JSON(
		response.New(response.ToCustomer(*cust)),
	)
}

func (h *CustomerHandler) SearchWithInvoiceInfo(c fiber.Ctx) error {
	search := c.Query("search")
	result, err := h.svc.SearchWithInvoiceInfo(c.Context(), search)
	if err != nil {
		return fmt.Errorf("search customer with invoice info: %w", err)
	}

	return c.JSON(
		response.New(response.ToCustomerWithInvoiceInfoList(result)),
	)
}
