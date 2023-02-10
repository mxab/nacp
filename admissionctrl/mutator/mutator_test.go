package mutator

import (
	"testing"

	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
)

func TestHelloMutator(t *testing.T) {
	mutator := &HelloMutator{}

	//read the json file and unmarshal it into the job struct
	//this is the job that we will be mutating
	job := testutil.ReadJob(t)

	job, _, _ = mutator.Mutate(job)

	assert.Equal(t, "world", job.Meta["hello"])

}
