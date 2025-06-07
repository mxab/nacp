package validator

import (
	"context"
	"log/slog"

	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/mxab/nacp/admissionctrl/notation"
)

type NotationValidator struct {
	logger   *slog.Logger
	name     string
	verifier notation.ImageVerifier
}

func (v *NotationValidator) Validate(payload *types.Payload) ([]error, error) {
	for _, tg := range payload.Job.TaskGroups {
		for _, task := range tg.Tasks {
			// check if the task driver is docker
			// should we consider podman?
			if task.Driver != "docker" {
				continue
			}

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

func NewNotationValidator(logger *slog.Logger, name string, verifier notation.ImageVerifier) *NotationValidator {
	return &NotationValidator{
		logger:   logger,
		name:     name,
		verifier: verifier,
	}
}
