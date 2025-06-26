package mutator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/mxab/nacp/admissionctrl/mutator/jsonpatcher"
	"github.com/mxab/nacp/admissionctrl/remoteutil"
	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
)

type JsonPatchWebhookMutator struct {
	name     string
	logger   *slog.Logger
	endpoint *url.URL
	method   string
}
type jsonPatchWebhookResponse struct {
	Patch    []interface{} `json:"patch"`
	Warnings []string      `json:"warnings"`
	Errors   []string      `json:"errors"`
}

func NewJsonPatchWebhookMutator(name string, endpoint string, method string, logger *slog.Logger) (*JsonPatchWebhookMutator, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	return &JsonPatchWebhookMutator{
		name:     name,
		logger:   logger,
		endpoint: u,
		method:   method,
	}, nil
}
func (j *JsonPatchWebhookMutator) Mutate(ctx context.Context, payload *types.Payload) (*api.Job, bool, []error, error) {
	jobJson, err := json.Marshal(payload)
	if err != nil {
		return nil, false, nil, err
	}

	req, err := http.NewRequest(j.method, j.endpoint.String(), bytes.NewBuffer(jobJson))
	if err != nil {
		return nil, false, nil, err
	}

	remoteutil.ApplyContextHeaders(req, payload)

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, false, nil, err
	}

	patchResponse := &jsonPatchWebhookResponse{}
	err = json.NewDecoder(res.Body).Decode(&patchResponse)
	if err != nil {
		return nil, false, nil, err
	}

	var warnings []error
	if len(patchResponse.Warnings) > 0 {
		j.logger.Debug("Got errors from rule", "rule", j.name, "warnings", patchResponse.Warnings, "job", payload.Job.ID)
		for _, warning := range patchResponse.Warnings {
			warnings = append(warnings, fmt.Errorf("%s", warning))
		}
	}

	patchedJob, mutated, err := jsonpatcher.PatchJob(payload.Job, patchResponse.Patch)
	if err != nil {
		return nil, false, nil, err
	}
	return patchedJob, mutated, warnings, nil

}
func (j *JsonPatchWebhookMutator) Name() string {
	return j.name
}
