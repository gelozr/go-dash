package request

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/optional"
)

type CreateInvoice struct {
	CustomerID string  `json:"customer_id" validate:"required"`
	Amount     float64 `json:"amount" validate:"required,gt=0"`
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
	CustomerID *string  `json:"customer_id" form:"customer_id" validate:"omitnil,required,uuid4"`
	Amount     *float64 `json:"amount" validate:"omitnil,min=0"`
	Status     *string  `json:"status" validate:"omitnil,required,oneof=pending paid"`
	Date       *string  `json:"date" validate:"omitnil,required,rfc3339"`
	// Date Optional[nullable.Null[string]] `json:"date" validate:"omitnil,required,rfc3339"`
	// Date       Optional[nullable.Null[string]] `json:"date" validate:"omitnil,required,rfc3339"`
	// IsActive optional.Optional[*bool]   `json:"is_active" validate:"omitnil,boolean"`

	clearCustomerID bool
	clearDate       bool
}

func (req *UpdateInvoice) UnmarshalJSON(data []byte) error {
	type invoiceFields UpdateInvoice
	body := struct {
		*invoiceFields
		CustomerID optional.Optional[*string] `json:"customer_id"`
		Status     optional.Optional[string]  `json:"status"`
		Date       optional.Optional[*string] `json:"date"`
	}{invoiceFields: (*invoiceFields)(req)}
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	// A JSON null must fail status validation, while an omitted status skips it.
	if body.Status.IsPresent {
		req.Status = &body.Status.Val
	}
	if body.CustomerID.IsPresent {
		req.CustomerID = body.CustomerID.Val
		req.clearCustomerID = body.CustomerID.IsNull
	}
	if body.Date.IsPresent {
		req.Date = body.Date.Val
		req.clearDate = body.Date.IsNull
	}
	return nil
}

func (req *UpdateInvoice) ToDTO() (invoice.UpdateInput, error) {
	customerID, err := optional.StringToUUID(invoiceStringUpdate(req.CustomerID, req.clearCustomerID))
	if err != nil {
		return invoice.UpdateInput{},
			response.NewError("invalid customer id", http.StatusUnprocessableEntity, err)
	}

	date, err := optional.StringToTime(invoiceStringUpdate(req.Date, req.clearDate), time.RFC3339)
	if err != nil {
		return invoice.UpdateInput{},
			response.NewError("invalid date", http.StatusUnprocessableEntity, err)
	}

	update := invoice.UpdateInput{
		CustomerID: customerID,
		Amount:     req.Amount,
		Date:       date,
	}
	if req.Status != nil {
		update.Status = *req.Status
	}
	return update, nil
}

func invoiceStringUpdate(value *string, clear bool) optional.Optional[string] {
	if value == nil && !clear {
		return optional.Optional[string]{}
	}
	return optional.FromPtr(value)
}
