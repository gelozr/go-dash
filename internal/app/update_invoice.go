package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/invoice"
)

type UpdateInvoice struct {
	custSvc *customer.Service
	invSvc  *invoice.Service
	txm     db.TxManager
}

func NewUpdateInvoice(custSvc *customer.Service, invSvc *invoice.Service, txm db.TxManager) *UpdateInvoice {
	return &UpdateInvoice{custSvc, invSvc, txm}
}

func (u *UpdateInvoice) Execute(ctx context.Context, id uuid.UUID, in invoice.UpdateInput) (*invoice.Invoice, error) {
	var inv *invoice.Invoice

	txErr := u.txm.Do(ctx, func(txCtx context.Context) error {
		// a null customer_id unlinks the customer, so only a new value needs checking
		if in.CustomerID.IsPresent && !in.CustomerID.IsNull {
			exists, err := u.custSvc.Exists(txCtx, in.CustomerID.Val)
			if err != nil {
				return fmt.Errorf("exists customer: %w", err)
			}
			if !exists {
				return customer.ErrCustomerNotFound
			}
		}

		var err error
		inv, err = u.invSvc.Update(txCtx, id, in)
		return err
	})

	if txErr != nil {
		return nil, fmt.Errorf("update invoice tx: %w", txErr)
	}

	return inv, nil
}
