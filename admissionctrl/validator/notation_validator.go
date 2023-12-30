package validator

import (
	"context"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl/notation"
)

type NotationValidator struct {
	logger   hclog.Logger
	name     string
	verifier notation.ImageVerifier
}

func (v *NotationValidator) Validate(job *api.Job) ([]error, error) {
	for _, tg := range job.TaskGroups {
		for _, task := range tg.Tasks {
			image, ok := task.Config["image"].(string)
			if !ok {
				continue
			}
			err := v.verifier.VerifyImage(context.Background(), image)
			if err != nil {
				return []error{err}, nil
			}
		}
	}

	return nil, nil
}

func (v *NotationValidator) Name() string {
	return v.name
}

func NewNotationValidator(logger hclog.Logger, name string, verifier notation.ImageVerifier) *NotationValidator {
	return &NotationValidator{
		logger:   logger,
		name:     name,
		verifier: verifier,
	}
}
