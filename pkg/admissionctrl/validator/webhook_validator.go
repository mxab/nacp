package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/mxab/nacp/pkg/admissionctrl/remoteutil"
	"github.com/mxab/nacp/pkg/admissionctrl/types"

	"github.com/hashicorp/go-multierror"
)

type WebhookValidator struct {
	endpoint *url.URL
	logger   *slog.Logger
	method   string
	name     string
}

type validationWebhookResponse struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func (w *WebhookValidator) Validate(ctx context.Context, payload *types.Payload) ([]error, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, w.method, w.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	remoteutil.ApplyContextHeaders(req, payload)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := remoteutil.NewInstrumentedClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	validationResult := &validationWebhookResponse{}
	if err := remoteutil.DecodeJSONResponse(resp, validationResult); err != nil {
		return nil, err
	}

	if len(validationResult.Errors) > 0 {
		w.logger.Error("validation errors", "errors", validationResult.Errors, "rule", w.name, "job", payload.Job.ID)
		oneError := &multierror.Error{}
		for _, e := range validationResult.Errors {
			oneError = multierror.Append(oneError, fmt.Errorf("%v", e))
		}
		return nil, oneError
	}

	var warnings []error
	if len(validationResult.Warnings) > 0 {

		for _, w := range validationResult.Warnings {
			warnings = append(warnings, fmt.Errorf("%v", w))
		}
		return warnings, nil

	}
	return warnings, nil
}
func (w *WebhookValidator) Name() string {
	return w.name
}
func NewWebhookValidator(name string, endpoint string, method string, logger *slog.Logger) (*WebhookValidator, error) {
	u, err := remoteutil.ParseEndpoint(endpoint)
	if err != nil {
		return nil, err
	}
	return &WebhookValidator{
		name:     name,
		logger:   logger,
		endpoint: u,
		method:   method,
	}, nil
}
