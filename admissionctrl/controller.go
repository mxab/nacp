package admissionctrl

// Admission Controller code copied from Nomad and adjusted to use api.Job instead of structs.Job
// https://github.com/hashicorp/nomad/blob/v1.5.0-beta.1/nomad/job_endpoint_hooks.go

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mxab/nacp/admissionctrl/types"
	"github.com/mxab/nacp/o11y"
	"go.opentelemetry.io/otel"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
)

type Metrics struct {
	validatorWarningCount o11y.NacpValidatorWarningCount
	validatorErrorCount   o11y.NacpValidatorErrorCount
	mutatorWarningCount   o11y.NacpMutatorWarningCount
	mutatorErrorCount     o11y.NacpMutatorErrorCount
	mutatorMutationCount  o11y.NacpMutatorMutationCount
}

func newMetrics() *Metrics {
	meter := otel.Meter("nacp.controller")

	validatorWarningCount, err := o11y.NewNacpValidatorWarningCount(meter)
	if err != nil {
		panic(err)
	}
	validatorErrorCount, err := o11y.NewNacpValidatorErrorCount(meter)
	if err != nil {
		panic(err)
	}
	mutatorWarningCount, err := o11y.NewNacpMutatorWarningCount(meter)
	if err != nil {
		panic(err)
	}
	mutatorErrorCount, err := o11y.NewNacpMutatorErrorCount(meter)
	if err != nil {
		panic(err)
	}
	mutatorMutationCount, err := o11y.NewNacpMutatorMutationCount(meter)
	if err != nil {
		panic(err)
	}
	return &Metrics{
		validatorWarningCount: validatorWarningCount,
		validatorErrorCount:   validatorErrorCount,
		mutatorWarningCount:   mutatorWarningCount,
		mutatorErrorCount:     mutatorErrorCount,
		mutatorMutationCount:  mutatorMutationCount,
	}
}

type AdmissionController interface {
	Name() string
}

type JobMutator interface {
	AdmissionController
	Mutate(*types.Payload) (*api.Job, bool, []error, error)
}

type JobValidator interface {
	AdmissionController
	Validate(*types.Payload) (warnings []error, err error)
}

type JobHandler struct {
	mutators     []JobMutator
	validators   []JobValidator
	resolveToken bool
	logger       *slog.Logger
	metrics      *Metrics
}

func NewJobHandler(mutators []JobMutator, validators []JobValidator, logger *slog.Logger, resolverToken bool) *JobHandler {
	return &JobHandler{
		mutators:     mutators,
		validators:   validators,
		logger:       logger,
		resolveToken: resolverToken,
		metrics:      newMetrics(),
	}
}

func (j *JobHandler) ApplyAdmissionControllers(payload *types.Payload) (out *api.Job, warnings []error, err error) {
	// Mutators run first before validators, so validators view the final rendered job.
	// So, mutators must handle invalid jobs.
	out, warnings, err = j.AdmissionMutators(payload)
	if err != nil {
		return nil, nil, err
	}

	validateWarnings, err := j.AdmissionValidators(payload)
	if err != nil {
		return nil, nil, err
	}
	warnings = append(warnings, validateWarnings...)

	return out, warnings, nil
}

// AdmissionMutators returns an updated job as well as warnings or an error.
func (j *JobHandler) AdmissionMutators(payload *types.Payload) (job *api.Job, warnings []error, err error) {
	var w []error
	job = payload.Job
	j.logger.Debug("applying job mutators", "mutators", len(j.mutators), "job", payload.Job.ID)
	for _, mutator := range j.mutators {
		j.logger.Debug("applying job mutator", "mutator", mutator.Name(), "job", payload.Job.ID)
		var mutated bool
		job, mutated, w, err = mutator.Mutate(&types.Payload{
			Job:     job,
			Context: payload.Context})
		if err != nil {
			j.metrics.mutatorErrorCount.Add(context.Background(), 1, mutator.Name())
			return nil, nil, fmt.Errorf("error in job mutator %s: %v", mutator.Name(), err)
		}
		if job == nil {
			j.metrics.mutatorErrorCount.Add(context.Background(), 1, mutator.Name())
			return nil, nil, fmt.Errorf("job mutator %s returned nil job", mutator.Name())
		}
		if mutated {
			j.metrics.mutatorMutationCount.Add(context.Background(), 1, mutator.Name())
		}
		j.metrics.mutatorWarningCount.Add(context.Background(), float64(len(w)), mutator.Name())

		j.logger.Debug("job mutate results", "mutator", mutator.Name(), "warnings", w, "error", err)

		warnings = append(warnings, w...)
	}
	return job, warnings, err
}

// AdmissionValidators returns a slice of validation warnings and a multierror
// of validation failures.
func (j *JobHandler) AdmissionValidators(payload *types.Payload) ([]error, error) {
	// ensure job is not mutated
	j.logger.Debug("applying job validators", "validators", len(j.validators), "job", payload.Job.ID)
	job := copyJob(payload.Job)

	var warnings []error
	var errs error

	for _, validator := range j.validators {
		j.logger.Debug("applying job validator", "validator", validator.Name(), "job", job.ID)
		w, err := validator.Validate(payload)
		j.metrics.validatorWarningCount.Add(context.Background(), float64(len(w)), validator.Name())
		j.logger.Debug("job validate results", "validator", validator.Name(), "warnings", w, "error", err)
		if err != nil {
			j.metrics.validatorErrorCount.Add(context.Background(), 1, validator.Name())
			errs = multierror.Append(errs, err)
		}
		warnings = append(warnings, w...)
	}

	return warnings, errs

}

func (j *JobHandler) ResolveToken() bool {
	return j.resolveToken
}

func copyJob(job *api.Job) *api.Job {
	jobCopy := &api.Job{}
	data, err := json.Marshal(job)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(data, jobCopy)
	if err != nil {
		return nil
	}
	return jobCopy
}
