package lib

import (
	"github.com/go-playground/validator/v10"
	"github.com/spf13/afero"
)

var Validate *validator.Validate = NewValidator(afero.NewOsFs())

func NewValidator(fs afero.Fs) *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterValidation("abspath", func(fl validator.FieldLevel) bool {
		return IsAbs(fl.Field().String())
	})
	v.RegisterValidation("validid", func(fl validator.FieldLevel) bool {
		return IsValidID(fl.Field().String())
	})
	v.RegisterValidation("dir", func(fl validator.FieldLevel) bool {
		exist, _ := afero.DirExists(fs, fl.Field().String())
		return exist
	})

	return v
}
