package gp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/fr"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTrans "github.com/go-playground/validator/v10/translations/en"
	frTrans "github.com/go-playground/validator/v10/translations/fr"

	"github.com/gelozr/go-dash/internal/http"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/optional"
)

type Validator struct {
	validator *validator.Validate
	uni       *ut.UniversalTranslator
}

var _ validation.Validator = (*Validator)(nil)

func New() (*Validator, error) {
	instance := validator.New(validator.WithRequiredStructEnabled())

	// registers a function to get alternate JSON names
	instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "" || name == "-" {
			return fld.Name
		}
		return name
	})

	// setup translations
	enLocale := en.New()
	frLocale := fr.New()
	uni := ut.New(enLocale, enLocale, frLocale)

	enTranslator, _ := uni.GetTranslator("en")
	frTranslator, _ := uni.GetTranslator("fr")

	if err := enTrans.RegisterDefaultTranslations(instance, enTranslator); err != nil {
		return nil, fmt.Errorf("register [en] translations error: %w", err)
	}

	if err := frTrans.RegisterDefaultTranslations(instance, frTranslator); err != nil {
		return nil, fmt.Errorf("register [fr] translations error: %w", err)
	}

	instance.RegisterAlias("rfc3339", "datetime="+time.RFC3339)
	for _, translation := range []struct {
		translator ut.Translator
		message    string
	}{
		{enTranslator, "{0} must be a valid RFC-3339 date-time"},
		{frTranslator, "{0} doit être une date et une heure valides au format RFC 3339"},
	} {
		err := instance.RegisterTranslation(
			"rfc3339",
			translation.translator,
			func(ut ut.Translator) error {
				return ut.Add("rfc3339", translation.message, true)
			},
			func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T("rfc3339", fe.Field())
				return t
			},
		)
		if err != nil {
			return nil, fmt.Errorf("register [%s] rfc3339 validation translation error: %w", translation.translator.Locale(), err)
		}
	}

	registerOptionalType[string](instance)
	registerOptionalType[int](instance)
	registerOptionalType[bool](instance)
	registerOptionalType[float64](instance)

	return &Validator{
		validator: instance,
		uni:       uni,
	}, nil
}

func (g *Validator) ValidateStruct(ctx context.Context, s any) error {
	err := g.validator.StructCtx(ctx, s)

	return g.buildErrors(ctx, err)
}

func (g *Validator) buildErrors(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	locale := http.Locale(ctx)
	var vErrs validator.ValidationErrors
	if !errors.As(err, &vErrs) {
		return err
	}

	trans, found := g.uni.GetTranslator(locale)
	if !found {
		trans, _ = g.uni.GetTranslator("en")
	}

	errs := make(validation.Errors, len(vErrs))
	for _, vErr := range vErrs {
		f := vErr.Field()
		errs[f] = append(errs[f], vErr.Translate(trans))
	}
	return errs
}

func registerOptionalType[T any](validator *validator.Validate) {
	var z T
	t := reflect.TypeOf(z)

	// Check if T is a pointer then exit
	if t.Kind() == reflect.Ptr {
		log.Fatalf("registerOptionalType2: type parameter T must not be a pointer, got %v\n", t)
	}

	validator.RegisterCustomTypeFunc(func(field reflect.Value) any {
		o, ok := field.Interface().(optional.Optional[T])

		if !ok {
			return nil
		}
		// not in JSON ⇒ return nil of T
		if !o.IsPresent {
			return (*T)(nil)
		}

		// present + non-nullable ⇒ underlying value
		return o.Val
	}, optional.Optional[T]{})

	validator.RegisterCustomTypeFunc(func(field reflect.Value) any {
		o, ok := field.Interface().(optional.Optional[*T])

		if !ok {
			return nil
		}
		// not in JSON and explicitly null ⇒ return nil of T
		if !o.IsPresent || o.IsNull {
			return (*T)(nil)
		}

		// present + non-nullable ⇒ underlying value
		return o.Val
	}, optional.Optional[*T]{})
}
