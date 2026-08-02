package bundle

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mxab/nacp/pkg/admissionctrl/opa"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/o11y"
	"github.com/open-policy-agent/opa/v1/sdk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Outcome labels how a decision ended, for metrics.
const (
	outcomeAllow = "allow"
	outcomeDeny  = "deny"
	outcomeError = "error"
)

var decisionDuration = func() o11y.NacpOpaDecisionDuration {
	instrument, err := o11y.NewNacpOpaDecisionDuration(otel.Meter("nacp.opa.bundle"))
	if err != nil {
		panic(err)
	}
	return instrument
}()

// Decide evaluates path against this instance's active bundles and returns the
// parsed decision document.
//
// Everything the bundle adapters need in common lives here: the evaluation
// deadline, strict builtin errors, and the provenance that makes an admission
// outcome traceable back to a specific bundle revision.
func (i *Instance) Decide(ctx context.Context, path string, payload *types.Payload) (*opa.Decision, error) {
	// The deadline bounds evaluation only where rego reaches a cancellation
	// check (http.send, large iterations); a tight loop in a trivial rule can
	// still outrun it. It is a backstop, not a hard limit.
	if i.decisionTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, i.decisionTimeout)
		defer cancel()
	}

	span := trace.SpanFromContext(ctx)
	started := time.Now()

	result, err := i.opa.Decision(ctx, sdk.DecisionOptions{
		Input: payload,
		Path:  path,
		// A builtin that errors must fail the decision. Left off, the rule
		// silently evaluates to undefined and the policy stops applying.
		StrictBuiltinErrors: true,
	})
	if err != nil {
		i.record(ctx, path, started, outcomeError)
		if sdk.IsUndefinedErr(err) {
			return nil, fmt.Errorf("decision path %q is undefined in the active bundle of opa_bundle %q", path, i.id)
		}
		return nil, fmt.Errorf("failed to perform policy decision: %w", err)
	}

	attrs := []attribute.KeyValue{
		attribute.String("opa.bundle.source", i.id),
		attribute.String("opa.decision.path", path),
		attribute.String("opa.decision.id", result.ID),
	}
	logAttrs := []any{
		slog.String("opa.bundle.source", i.id),
		slog.String("opa.decision.path", path),
		slog.String("opa.decision.id", result.ID),
	}
	for name, provenance := range result.Provenance.Bundles {
		key := "opa.bundle." + name + ".revision"
		attrs = append(attrs, attribute.String(key, provenance.Revision))
		logAttrs = append(logAttrs, slog.String(key, provenance.Revision))
	}
	span.SetAttributes(attrs...)

	decision, err := opa.ParseDecision(result.Result, opa.Strict)
	if err != nil {
		i.record(ctx, path, started, outcomeError)
		i.logger.DebugContext(ctx, "OPA bundle decision was not a valid decision document", logAttrs...)
		return nil, err
	}

	outcome := outcomeAllow
	if len(decision.Errors) > 0 {
		outcome = outcomeDeny
	}
	i.record(ctx, path, started, outcome)
	i.logger.DebugContext(ctx, "OPA bundle decision",
		append(logAttrs,
			slog.String("opa.decision.outcome", outcome),
			slog.Any("errors", decision.Errors),
			slog.Any("warnings", decision.Warnings),
		)...)

	return decision, nil
}

func (i *Instance) record(ctx context.Context, path string, started time.Time, outcome string) {
	decisionDuration.Record(ctx, time.Since(started).Seconds(), i.id, outcome, path)
}
