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
	var respData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&respData)

	if err != nil {
		return nil, err
	}

	errors, ok := respData["result"].(map[string]interface{})["errors"].([]interface{})
	if ok && len(errors) > 0 {
		oneError := &multierror.Error{}
		for _, e := range errors {

			oneError = multierror.Append(oneError, fmt.Errorf("%v", e))
		}
		return nil, oneError
	}

	warnings, ok := respData["result"].(map[string]interface{})["warnings"].([]interface{})
	var warns []error
	if ok && len(warnings) > 0 {

		for _, w := range warnings {
			warns = append(warns, fmt.Errorf("%v", w))
		}
		return warns, nil

	}
	return warns, nil
}
