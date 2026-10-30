package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/app"
	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/listing"
	"github.com/gelozr/go-dash/internal/logger"
)

type InvoiceHandler struct {
	invSvc        *invoice.Service
	createInvoice *app.CreateInvoice
	validator     validation.Validator
	logger        logger.Logger
}

func NewInvoiceHandler(
	invSvc *invoice.Service,
	createInvoice *app.CreateInvoice,
	validator validation.Validator,
	logger logger.Logger,
) *InvoiceHandler {
	return &InvoiceHandler{
		invSvc:        invSvc,
		createInvoice: createInvoice,
		validator:     validator,
		logger:        logger.With("component", "http.invoice"),
	}
}

func (h *InvoiceHandler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid id.")
	}

	inv, err := h.invSvc.Get(c.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, invoice.ErrInvoiceNotFound):
			return fiber.NewError(fiber.StatusNotFound, "invoice not found.")
		default:
			return fmt.Errorf("get invoice by id: %w", err)
		}
	}

	return c.JSON(
		response.New(response.ToInvoice(*inv)),
	)
}

func (h *InvoiceHandler) GetLatest(c fiber.Ctx) error {
	limit := min(max(getDefaultNum(c.Query("limit"), 5), 1), 100)

	invs, err := h.invSvc.ListWithCustomerInfo(c.Context(), listing.SortLatest, limit)
	if err != nil {
		return fmt.Errorf("list invoices: %w", err)
	}

	return c.JSON(
		response.New(response.ToInvoicesWithCustomerInfo(invs)),
	)
}

func (h *InvoiceHandler) Search(c fiber.Ctx) error {
	size := getDefaultNum(c.Query("size"), 10)
	page := getDefaultNum(c.Query("page"), 1)

	p := listing.NewPage(page, size)

	filter := invoice.SearchFilter{
		Text: c.Query("search"),
		Sort: listing.SortLatest,
	}

	result, err := h.invSvc.Search(c.Context(), filter, p)
	if err != nil {
		return fmt.Errorf("search invoices: %w", err)
	}

	return c.JSON(
		response.PaginateList(result, response.ToInvoice),
	)
}

func (h *InvoiceHandler) Create(c fiber.Ctx) error {
	var req request.CreateInvoice

	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("create invoice bind request body: %w", err)
	}

	if err := h.validator.ValidateStruct(c.Context(), req); err != nil {
		return fmt.Errorf("create invoice validation: %w", err)
	}

	reqInv, err := req.ToInvoice()
	if err != nil {
		return fmt.Errorf("create invoice to dto: %w", err)
	}

	inv, err := h.createInvoice.Execute(c.Context(), reqInv)
	if err != nil {
		switch {
		case errors.Is(err, customer.ErrCustomerNotFound):
			return fiber.NewError(fiber.StatusNotFound, "customer not found.")
		default:
			return fmt.Errorf("create invoice: %w", err)
		}
	}

	return c.Status(http.StatusCreated).JSON(
		response.New(response.ToInvoice(*inv)),
	)
}

func (h *InvoiceHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid id.")
	}

	var req request.UpdateInvoice
	if err = c.Bind().Body(&req); err != nil {
		return fmt.Errorf("update invoice bind request body: %w", err)
	}

	if err = h.validator.ValidateStruct(c.Context(), req); err != nil {
		return fmt.Errorf("create invoice validation: %w", err)
	}

	updateInput, err := req.ToDTO()
	if err != nil {
		return err
	}

	inv, err := h.invSvc.Update(c.Context(), id, updateInput)
	if err != nil {
		switch {
		case errors.Is(err, invoice.ErrInvoiceNotFound):
			return fiber.NewError(fiber.StatusNotFound, "invoice not found.")
		default:
			return fmt.Errorf("update invoice: %w", err)
		}
	}

	return c.JSON(
		response.New(response.ToInvoice(*inv)),
	)
}

func (h *InvoiceHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid id.")
	}

	if err = h.invSvc.Delete(c.Context(), id); err != nil {
		switch {
		case errors.Is(err, invoice.ErrInvoiceNotFound):
			return fiber.NewError(fiber.StatusNotFound, "invoice not found.")
		default:
			return fmt.Errorf("delete invoice by id: %w", err)
		}
	}

	return c.SendStatus(fiber.StatusNoContent)
}
