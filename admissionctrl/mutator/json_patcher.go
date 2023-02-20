package mutator

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antchfx/jsonquery"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hashicorp/nomad/api"
)

type JSONPatcher struct {
}

func buildJsonPath(node *jsonquery.Node) string {
	var path string
	for node != nil {
		if node.Parent != nil {
			path = fmt.Sprintf("/%s%s", node.Data, path)
		} else {
			path = fmt.Sprintf("%s%s", node.Data, path)
		}
		node = node.Parent
	}
	return path
}
func (j *JSONPatcher) Mutate(job *api.Job) (out *api.Job, warnings []error, err error) {

	jobData, err := json.Marshal(job)
	if err != nil {
		return nil, nil, err
	}

	doc, err := jsonquery.Parse(strings.NewReader(string(jobData)))
	if err != nil {
		return nil, nil, err
	}

	Meta := jsonquery.FindOne(doc, "/Meta")

	// print(Meta)
	fmt.Println(Meta)
	Tasks := jsonquery.Find(doc, "//Tasks")

	for _, task := range Tasks {
		fmt.Println(buildJsonPath(task))
	}
	// print(Tasks)
	fmt.Println(Tasks)

	patchJSON := []byte(`[
		{
			"op": "add",
			"path": "/Meta",
			"value": {"foo": "bar"}
		}
	]`)

	patch, err := jsonpatch.DecodePatch(patchJSON)

	if err != nil {
		return nil, nil, err
	}

	modified, err := patch.Apply(jobData)
	if err != nil {
		return nil, nil, err
	}

	json.Unmarshal(modified, &job)

	return job, nil, nil
}
func (j *JSONPatcher) Name() string {
	return "jsonpatch"
}
