package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
)

type WebhookValidator struct {
	endpoint *url.URL
	method   string
	name     string
}

type validationWebhookResponse struct {
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func (w *WebhookValidator) Validate(job *api.Job) ([]error, error) {
	//calls and endpoint with a job, returns a json response with result.errors and result.warnings fields
	//if result.errors is not empty, return an error
	//if result.warnings is not empty, return a all warnings

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

	//check if result.errors is not empty, return an error
	//check if result.warnings is not empty, return a all warnings
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
