package mutator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/hashicorp/nomad/api"
)

type WebhookMutator struct {
	name     string
	endpoint *url.URL
	method   string
}

func (w *WebhookMutator) Mutate(job *api.Job) (out *api.Job, warnings []error, err error) {

	data, err := json.Marshal(job)
	if err != nil {
		return nil, nil, err
	}
	buffer := bytes.NewBuffer(data)

	req, err := http.NewRequest(w.method, w.endpoint.String(), buffer)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	newJob := &api.Job{}
	err = json.NewDecoder(resp.Body).Decode(newJob)
	if err != nil {
		return nil, nil, err
	}
	return newJob, nil, nil
}
func (w *WebhookMutator) Name() string {
	return w.name
}

func NewWebhookMutator(name string, endpoint *url.URL, method string) *WebhookMutator {

	return &WebhookMutator{
		name:     name,
		endpoint: endpoint,
		method:   method,
	}
}
