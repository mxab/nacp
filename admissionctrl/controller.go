package admissionctrl

// Admission Controller code copied from Nomad
// https://github.com/hashicorp/nomad/blob/v1.5.0-beta.1/nomad/job_endpoint_hooks.go

import (
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/nomad/structs"
)

type AdmissionController interface {
	Name() string
}

type JobMutator interface {
	AdmissionController
	Mutate(*structs.Job) (out *structs.Job, warnings []error, err error)
}

type JobValidator interface {
	AdmissionController
	Validate(*structs.Job) (warnings []error, err error)
}
type JobHandler struct {
	mutators   []JobMutator
	validators []JobValidator
	logger     hclog.Logger
}

func (j *JobHandler) ApplyAdmissionControllers(job *structs.Job) (out *structs.Job, warnings []error, err error) {
	// Mutators run first before validators, so validators view the final rendered job.
	// So, mutators must handle invalid jobs.
	out, warnings, err = j.admissionMutators(job)
	if err != nil {
		return nil, nil, err
	}

	validateWarnings, err := j.admissionValidators(job)
	if err != nil {
		return nil, nil, err
	}
	warnings = append(warnings, validateWarnings...)

	return out, warnings, nil
}

// admissionMutator returns an updated job as well as warnings or an error.
func (j *JobHandler) admissionMutators(job *structs.Job) (_ *structs.Job, warnings []error, err error) {
	var w []error
	for _, mutator := range j.mutators {
		job, w, err = mutator.Mutate(job)
		j.logger.Trace("job mutate results", "mutator", mutator.Name(), "warnings", w, "error", err)
		if err != nil {
			return nil, nil, fmt.Errorf("error in job mutator %s: %v", mutator.Name(), err)
		}
		warnings = append(warnings, w...)
	}
	return job, warnings, err
}

// admissionValidators returns a slice of validation warnings and a multierror
// of validation failures.
func (j *JobHandler) admissionValidators(origJob *structs.Job) ([]error, error) {
	// ensure job is not mutated
	job := origJob.Copy()

	var warnings []error
	var errs error

	for _, validator := range j.validators {
		w, err := validator.Validate(job)
		j.logger.Trace("job validate results", "validator", validator.Name(), "warnings", w, "error", err)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
		warnings = append(warnings, w...)
	}

	return warnings, errs

}
