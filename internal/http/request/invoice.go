package request

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/optional"
)

type CreateInvoice struct {
	CustomerID string  `json:"customer_id" validate:"required"`
	Amount     float64 `json:"amount" validate:"required"`
	Status     string  `json:"status" validate:"required,oneof=pending paid"`
	Date       string  `json:"date" validate:"required,rfc3339"`
}

func (req *CreateInvoice) ToInvoice() (invoice.Invoice, error) {
	custID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		return invoice.Invoice{},
			response.NewError("invalid customer id", http.StatusUnprocessableEntity, err)
	}

	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		return invoice.Invoice{},
			response.NewError("invalid date", http.StatusUnprocessableEntity, err)
	}

	return invoice.Invoice{
		CustomerID: &custID,
		Amount:     req.Amount,
		Status:     req.Status,
		Date:       &date,
	}, nil
}

type UpdateInvoice struct {
	CustomerID optional.Optional[*string] `json:"customer_id" validate:"omitnil,required,uuid4"`
	Amount     float64                    `json:"amount" validate:"min=0"`
	Status     string                     `json:"status" validate:"required,oneof=pending paid"`
	Date       optional.Optional[*string] `json:"date" validate:"omitnil,required,rfc3339"`
	// Date Optional[nullable.Null[string]] `json:"date" validate:"omitnil,required,rfc3339"`
	// Date       Optional[nullable.Null[string]] `json:"date" validate:"omitnil,required,rfc3339"`
	// IsActive optional.Optional[*bool]   `json:"is_active" validate:"omitnil,boolean"`
}

func (req *UpdateInvoice) ToDTO() (invoice.UpdateInput, error) {
	customerID, err := optional.StringToUUID(req.CustomerID)
	if err != nil {
		return invoice.UpdateInput{},
			response.NewError("invalid customer id", http.StatusUnprocessableEntity, err)
	}

	date, err := optional.StringToTime(req.Date, time.RFC3339)
	if err != nil {
		return invoice.UpdateInput{},
			response.NewError("invalid date", http.StatusUnprocessableEntity, err)
	}

	return invoice.UpdateInput{
		CustomerID: customerID,
		Amount:     req.Amount,
		Status:     req.Status,
		Date:       date,
	}, nil
}
