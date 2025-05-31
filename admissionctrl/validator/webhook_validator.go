package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/mxab/nacp/admissionctrl/remoteutil"
	"github.com/mxab/nacp/admissionctrl/types"

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

func (w *WebhookValidator) Validate(payload *types.Payload) ([]error, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(w.method, w.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	remoteutil.ApplyContextHeaders(req, payload)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	valdationResult := &validationWebhookResponse{}
	err = json.NewDecoder(resp.Body).Decode(valdationResult)

	if err != nil {
		return nil, err
	}

	if len(valdationResult.Errors) > 0 {
		w.logger.Error("validation errors", "errors", valdationResult.Errors, "rule", w.name, "job", payload.Job.ID)
		oneError := &multierror.Error{}
		for _, e := range valdationResult.Errors {
			oneError = multierror.Append(oneError, fmt.Errorf("%v", e))
		}
		return nil, oneError
	}

	var warnings []error
	if len(valdationResult.Warnings) > 0 {

		for _, w := range valdationResult.Warnings {
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
	u, err := url.Parse(endpoint)
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
