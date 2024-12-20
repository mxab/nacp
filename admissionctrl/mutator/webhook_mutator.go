package mutator

import (
	"bytes"
	"encoding/json"
	"github.com/mxab/nacp/config"
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
	// Add context headers if available
	if ctx := req.Context(); ctx != nil {
		if reqCtx, ok := ctx.Value("request_context").(*config.RequestContext); ok {
			if reqCtx.ClientIP != "" {
				req.Header.Set("X-Forwarded-For", reqCtx.ClientIP)
			}
			if reqCtx.AccessorID != "" {
				req.Header.Set("X-Nomad-Accessor-ID", reqCtx.AccessorID)
			}
		}
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
