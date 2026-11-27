package request

import (
	"github.com/gelozr/go-dash/internal/customer"
)

type CreateCustomer struct {
	Name     string  `json:"name" validate:"required,max=255"`
	Email    string  `json:"email" validate:"required,max=255,email"`
	ImageURL *string `json:"image_url" form:"image_url" validate:"omitempty,max=255"`
}

func (req *CreateCustomer) ToCustomer() customer.Customer {
	return customer.Customer{
		Name:     req.Name,
		Email:    req.Email,
		ImageURL: req.ImageURL,
	}
}
