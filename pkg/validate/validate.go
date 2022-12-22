package validate

import "github.com/go-playground/validator/v10"

// Single instance of Validate, for cache purposes
var Validate *validator.Validate

func init() {
	Validate = validator.New()
}
