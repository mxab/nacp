package mutator

import "github.com/hashicorp/nomad/nomad/structs"

type HelloMutator struct {
}

func (h *HelloMutator) Mutate(job *structs.Job) (out *structs.Job, warnings []error, err error) {

	if job.Meta == nil {
		job.Meta = make(map[string]string)
	}

	job.Meta["hello"] = "world"

	return job, nil, nil
}

func (h *HelloMutator) Name() string {
	return "hello"
}
