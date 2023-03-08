package mutator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/nomad/api"
)

type JsonPatchWebhookMutator struct {
	name string

	endpoint *url.URL
	method   string
}
type jsonPatchWebhookResponse struct {
	Patch    []interface{} `json:"patch"`
	Warnings []string      `json:"warnings"`
	Errors   []string      `json:"errors"`
}

func NewJsonPatchWebhookMutator(name string, endpoint *url.URL, method string) *JsonPatchWebhookMutator {
	return &JsonPatchWebhookMutator{
		name:     name,
		endpoint: endpoint,
		method:   method,
	}
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
