package lib

import (
	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate = NewValidator()

func NewValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterValidation("abspath", func(fl validator.FieldLevel) bool {
		return IsAbs(fl.Field().String())
	})
	v.RegisterValidation("validid", func(fl validator.FieldLevel) bool {
		return IsValidID(fl.Field().String())
	})

	return v
}
