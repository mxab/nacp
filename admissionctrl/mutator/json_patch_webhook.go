package mutator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
)

type JsonPatchWebhookMutator struct {
	name     string
	logger   hclog.Logger
	endpoint *url.URL
	method   string
}
type jsonPatchWebhookResponse struct {
	Patch    []interface{} `json:"patch"`
	Warnings []string      `json:"warnings"`
	Errors   []string      `json:"errors"`
}

func NewJsonPatchWebhookMutator(name string, endpoint string, method string, logger hclog.Logger) (*JsonPatchWebhookMutator, error) {
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
func (j *JsonPatchWebhookMutator) Mutate(job *api.Job) (*api.Job, []error, error) {

	jobJson, err := json.Marshal(job)
	if err != nil {
		return nil, nil, err
	}
	httpClient := &http.Client{}

	req, err := http.NewRequest(j.method, j.endpoint.String(), bytes.NewBuffer(jobJson))
	if err != nil {
		return nil, nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	patchResponse := &jsonPatchWebhookResponse{}
	err = json.NewDecoder(res.Body).Decode(&patchResponse)
	if err != nil {
		return nil, nil, err
	}

	var warnings []error
	for _, warning := range patchResponse.Warnings {
		warnings = append(warnings, fmt.Errorf(warning))
	}

	patchJson, err := json.Marshal(patchResponse.Patch)
	if err != nil {
		return nil, nil, err
	}
	patch, err := jsonpatch.DecodePatch(patchJson)
	if err != nil {
		return nil, nil, err
	}
	j.logger.Debug("Got patch fom rule", "rule", j.name, "patch", string(patchJson))
	patchedJobJson, err := patch.Apply(jobJson)

	if err != nil {
		return nil, nil, err
	}
	var patchedJob api.Job
	err = json.Unmarshal(patchedJobJson, &patchedJob)
	if err != nil {
		return nil, nil, err
	}
	return &patchedJob, warnings, nil

}
func (j *JsonPatchWebhookMutator) Name() string {
	return j.name
}
