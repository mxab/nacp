package mutator

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl/mutator/jsonpatcher"
	"github.com/open-policy-agent/opa/v1/sdk"
)

type OpaBundleMutator struct {
	name   string
	path   string
	logger *slog.Logger
	opa    *sdk.OPA
}

func (m *OpaBundleMutator) Mutate(context context.Context, job *api.Job) (result *api.Job, mutated bool, warns []error, err error) {
	mutated = false
	warns = []error{}
	err = nil

	decision, err := m.opa.Decision(context, sdk.DecisionOptions{
		Input: job,
		Path:  m.path,
	})
	if err != nil {
		err = fmt.Errorf("failed to perform policy decision: %w", err)
		return
	}
	m.logger.DebugContext(context, "OPA decision", slog.Any("result", decision))

	if rmap, ok := decision.Result.(map[string]interface{}); ok {
		if errs, found := rmap["errors"]; found {
			if errlist, ok := errs.([]interface{}); ok {

				for _, e := range errlist {
					if emsg, ok := e.(string); ok {
						err = multierror.Append(err, errors.New(emsg))
					} else if e != nil {
						err = fmt.Errorf("policy yielded an invalid error entry value: %v", e)
						return
					}

				}
				if err != nil {
					return
				}
			} else if errs != nil {
				err = fmt.Errorf("policy yielded an invalid errors value: %v", errs)
				return
			}
		}
		if warnsRaw, found := rmap["warnings"]; found {
			if warnlist, ok := warnsRaw.([]interface{}); ok {

				for _, w := range warnlist {
					if wmsg, ok := w.(string); ok {
						warns = append(warns, errors.New(wmsg))
					} else if w != nil {
						err = fmt.Errorf("policy yielded an invalid warning entry value: %v", w)
						return
					}
				}
			} else if warnsRaw != nil {
				err = fmt.Errorf("policy yielded an invalid warnings value: %v", warnsRaw)
				return
			}
		}
		if patch, found := rmap["patch"]; found {
			if ops, ok := patch.([]interface{}); ok {
				result, mutated, err = jsonpatcher.PatchJob(job, ops)
				if err != nil {
					err = fmt.Errorf("policy yielded patch failed: %w", err)
					return
				}
			} else if patch != nil {
				err = fmt.Errorf("policy yielded an invalid patch value: %v", patch)
				return
			}
		} else {
			// No patch, return original job
			result = job
		}
	}

	return
}

func NewOpaBundleMutator(name string, path string, logger *slog.Logger, sdk *sdk.OPA) (*OpaBundleMutator, error) {
	return &OpaBundleMutator{
		name:   name,
		path:   path,
		logger: logger,
		opa:    sdk,
	}, nil
}

func (m *OpaBundleMutator) Name() string {
	return m.name
}
