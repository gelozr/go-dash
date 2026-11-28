package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/invoice"
)

type Invoice struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID *uuid.UUID `json:"customer_id"`
	Amount     float64    `json:"amount"`
	Status     string     `json:"status"`
	Date       *time.Time `json:"date"`
	IsActive   *bool      `json:"is_active"`
}

func ToInvoice(inv invoice.Invoice) Invoice {
	return Invoice{
		ID:         inv.ID,
		CustomerID: inv.CustomerID,
		Amount:     inv.Amount,
		Status:     inv.Status,
		Date:       inv.Date,
		IsActive:   inv.IsActive,
	}
}

type InvoiceWithCustomerInfo struct {
	ID               uuid.UUID  `json:"id"`
	CustomerID       *uuid.UUID `json:"customer_id"`
	Amount           float64    `json:"amount"`
	Status           string     `json:"status"`
	Date             *time.Time `json:"date"`
	CustomerName     string     `json:"name"`
	CustomerEmail    string     `json:"email"`
	CustomerImageURL string     `json:"image_url"`
}

func ToInvoiceWithCustomerInfo(inv invoice.WithCustomerInfo) InvoiceWithCustomerInfo {
	return InvoiceWithCustomerInfo{
		ID:               inv.ID,
		CustomerID:       inv.CustomerID,
		Amount:           inv.Amount,
		Status:           inv.Status,
		Date:             inv.Date,
		CustomerName:     inv.CustomerName,
		CustomerEmail:    inv.CustomerEmail,
		CustomerImageURL: inv.CustomerImageURL,
	}
}

func ToInvoicesWithCustomerInfo(invs []invoice.WithCustomerInfo) []InvoiceWithCustomerInfo {
	return ToList(invs, ToInvoiceWithCustomerInfo)
}
