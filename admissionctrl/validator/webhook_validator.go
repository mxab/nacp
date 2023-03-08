package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
)

type WebhookValidator struct {
	endpoint *url.URL
	logger   hclog.Logger
	method   string
	name     string
}

type validationWebhookResponse struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func (w *WebhookValidator) Validate(job *api.Job) ([]error, error) {

	data, err := json.Marshal(job)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(w.method, w.endpoint.String(), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

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
func NewWebhookValidator(name string, endpoint string, method string, logger hclog.Logger) (*WebhookValidator, error) {
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
