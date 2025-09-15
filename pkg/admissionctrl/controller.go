package admissionctrl

// Admission Controller code copied from Nomad and adjusted to use api.Job instead of structs.Job
// https://github.com/hashicorp/nomad/blob/v1.5.0-beta.1/nomad/job_endpoint_hooks.go

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mxab/nacp/o11y"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

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
	Mutate(context.Context, *types.Payload) (*api.Job, bool, []error, error)
}

type JobValidator interface {
	AdmissionController
	Validate(context.Context, *types.Payload) (warnings []error, err error)
}

type JobHandler struct {
	mutators     []JobMutator
	validators   []JobValidator
	resolveToken bool
	logger       *slog.Logger
	metrics      *Metrics
	tracer       trace.Tracer
}

func NewJobHandler(mutators []JobMutator, validators []JobValidator, logger *slog.Logger, resolverToken bool) *JobHandler {
	return &JobHandler{
		mutators:     mutators,
		validators:   validators,
		logger:       logger,
		resolveToken: resolverToken,
		metrics:      newMetrics(),
		tracer:       otel.Tracer("github.com/mxab/nacp"),
	}
}

func (j *JobHandler) ApplyAdmissionControllers(ctx context.Context, payload *types.Payload) (out *api.Job, warnings []error, err error) {
	// Mutators run first before validators, so validators view the final rendered job.
	// So, mutators must handle invalid jobs.

	ctx, span := j.tracer.Start(ctx, "admission.apply")
	defer span.End()

	span.SetAttributes(attribute.String("nomad.job.id", *payload.Job.ID))

	out, warnings, err = j.AdmissionMutators(ctx, payload)
	if err != nil {
		return nil, nil, err
	}

	validateWarnings, err := j.AdmissionValidators(ctx, payload)
	if err != nil {
		return nil, nil, err
	}
	warnings = append(warnings, validateWarnings...)

	return out, warnings, nil
}

// AdmissionMutators returns an updated job as well as warnings or an error.
func (j *JobHandler) AdmissionMutators(ctx context.Context, payload *types.Payload) (job *api.Job, warnings []error, err error) {

	ctx, span := j.tracer.Start(ctx, "mutators.process")
	defer span.End()
	var w []error
	job = payload.Job
	j.logger.DebugContext(ctx, "applying job mutators", "mutators", len(j.mutators), "job", payload.Job.ID)
	for _, mutator := range j.mutators {

		err = func() (err error) {

			jobId := *payload.Job.ID
			ctx, span := j.tracer.Start(ctx, fmt.Sprintf("mutate: %s", mutator.Name()), trace.WithAttributes(
				attribute.String("nomad.job.id", jobId),
				attribute.String("mutator.name", mutator.Name()),
			))

			defer span.End()

			j.logger.DebugContext(ctx, "applying job mutator", "mutator", mutator.Name(), "job", jobId)
			var mutated bool
			job, mutated, w, err = mutator.Mutate(ctx, &types.Payload{
				Job:     job,
				Context: payload.Context})
			if err != nil {
				span.SetStatus(codes.Error, "error in mutator")
				span.RecordError(err)

				j.metrics.mutatorErrorCount.Add(ctx, 1, mutator.Name())
				return fmt.Errorf("error in job mutator %s: %v", mutator.Name(), err)
			}
			if job == nil {
				span.SetStatus(codes.Error, "job mutator returned nil job")
				j.metrics.mutatorErrorCount.Add(ctx, 1, mutator.Name())
				return fmt.Errorf("job mutator %s returned nil job", mutator.Name())
			}
			if mutated {
				span.SetAttributes(attribute.Bool("mutated", true))
				j.metrics.mutatorMutationCount.Add(ctx, 1, mutator.Name())
			}
			j.metrics.mutatorWarningCount.Add(ctx, float64(len(w)), mutator.Name())

			j.logger.DebugContext(ctx, "job mutate results", "mutator", mutator.Name(), "warnings", w, "error", err)

			warnings = append(warnings, w...)
			return nil

		}()
		if err != nil {
			return nil, nil, err
		}
	}
	return job, warnings, err
}

// AdmissionValidators returns a slice of validation warnings and a multierror
// of validation failures.
func (j *JobHandler) AdmissionValidators(ctx context.Context, payload *types.Payload) ([]error, error) {
	// ensure job is not mutated

	ctx, span := j.tracer.Start(ctx, "validators.process")
	defer span.End()
	j.logger.DebugContext(ctx, "applying job validators", "validators", len(j.validators), "job", payload.Job.ID)
	job := copyJob(payload.Job)

	var warnings []error
	var errs error

	for _, validator := range j.validators {

		func() {
			jobId := *job.ID
			ctx, span := j.tracer.Start(ctx, fmt.Sprintf("validate: %s", validator.Name()), trace.WithAttributes(
				attribute.String("nomad.job.id", jobId),
				attribute.String("validator.name", validator.Name()),
			))
			defer span.End()
			j.logger.DebugContext(ctx, "applying job validator", "validator", validator.Name(), "job", jobId)
			w, err := validator.Validate(ctx, payload)
			j.metrics.validatorWarningCount.Add(ctx, float64(len(w)), validator.Name())
			j.logger.DebugContext(ctx, "job validate results", "job", jobId, "validator", validator.Name(), "warnings", w, "error", err)
			if err != nil {
				span.SetStatus(codes.Error, "error in validator")
				span.RecordError(err)
				j.metrics.validatorErrorCount.Add(ctx, 1, validator.Name())
				errs = multierror.Append(errs, err)
			}
			warnings = append(warnings, w...)
		}()
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
