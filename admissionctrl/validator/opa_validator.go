package validator

import (
	"github.com/hashicorp/nomad/nomad/structs"
)

func NewOpaValidator() *OpaValidator {
	return &OpaValidator{}
}

type OpaValidator struct {
}

func (v *OpaValidator) Validate(job *structs.Job) (*structs.Job, []error, error) {

	return nil, nil, nil
}

// Name
func (v *OpaValidator) Name() string {
	return "opa"
}
