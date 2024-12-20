package admissionctrl

// Admission Controller code copied from Nomad and adjusted to use api.Job instead of structs.Job
// https://github.com/hashicorp/nomad/blob/v1.5.0-beta.1/nomad/job_endpoint_hooks.go

import (
	"encoding/json"
	"fmt"
	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
)

type AdmissionController interface {
	Name() string
}

type JobMutator interface {
	AdmissionController
	Mutate(*types.Payload) (*api.Job, []error, error)
}

type JobValidator interface {
	AdmissionController
	Validate(*types.Payload) (warnings []error, err error)
}

type JobHandler struct {
	mutators     []JobMutator
	validators   []JobValidator
	resolveToken bool
	logger       hclog.Logger
}

func NewJobHandler(mutators []JobMutator, validators []JobValidator, logger hclog.Logger, resolverToken bool) *JobHandler {
	return &JobHandler{
		mutators:     mutators,
		validators:   validators,
		logger:       logger,
		resolveToken: resolverToken,
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
		job, w, err = mutator.Mutate(payload)
		j.logger.Trace("job mutate results", "mutator", mutator.Name(), "warnings", w, "error", err)
		if err != nil {
			return nil, nil, fmt.Errorf("error in job mutator %s: %v", mutator.Name(), err)
		}
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
		j.logger.Trace("job validate results", "validator", validator.Name(), "warnings", w, "error", err)
		if err != nil {
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
