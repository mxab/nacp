package mutator

import (
	"github.com/hashicorp/nomad/api"
)

type HelloMutator struct {
	MutatorName string
}

func (h *HelloMutator) Mutate(job *api.Job) (out *api.Job, warnings []error, err error) {

	if job.Meta == nil {
		job.Meta = make(map[string]string)
	}

	job.Meta["hello"] = "world"

	return job, nil, nil
}

func (h *HelloMutator) Name() string {
	return h.MutatorName
}
