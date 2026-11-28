package response

import (
	"fmt"
	"net/http"

	"github.com/gelozr/go-dash/internal/listing"
)

type Response[T any] struct {
	Data T `json:"data"`
}

func New[T any](data T) Response[T] {
	return Response[T]{
		Data: data,
	}
}

type Paginated[T any] struct {
	Total       int64 `json:"total"`
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
	Data        []T   `json:"data"`
}

// PaginateList converts a listing.Result[S] to Paginated[R]
func PaginateList[S, R any](res listing.Result[S], mapper func(S) R) Paginated[R] {
	return Paginated[R]{
		Total:       res.Total,
		CurrentPage: res.Page,
		PerPage:     res.PerPage,
		HasNext:     res.HasNext,
		HasPrev:     res.HasPrev,
		Data:        ToList(res.Items, mapper),
	}
}

func ToList[S, D any](data []S, mapper func(S) D) []D {
	res := make([]D, len(data))
	for i, v := range data {
		res[i] = mapper(v)
	}
	return res
}

type AppError struct {
	Message string
	Code    int
	Err     error
}

func NewError(message string, code int, errs ...error) *AppError {
	if code == 0 {
		code = http.StatusInternalServerError
	}
	if message == "" {
		message = http.StatusText(code)
	}
	var err error
	if len(errs) > 0 {
		err = errs[0]
	}
	return &AppError{
		Message: message,
		Code:    code,
		Err:     err,
	}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err.Error())
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

type ValidationError struct {
	Message string              `json:"message"`
	Errors  map[string][]string `json:"errors"`
}
