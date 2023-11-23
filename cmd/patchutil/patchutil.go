package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"

	"github.com/hashicorp/nomad/api"
	"github.com/wI2L/jsondiff"
)

const job string = `
{
    "Region": null,
    "Namespace": null,
    "ID": "my-app",
    "Name": "my-app",
    "Type": null,

}
`
const omitemptyregex = `"\w+"\s*:\s*(""|null|0|false),?\s*`

func main() {

	result := regexp.MustCompile(omitemptyregex).ReplaceAllString(job, "")
	fmt.Println(result)
	nomadClient, err := api.NewClient(&api.Config{})
	if err != nil {
		panic(err)
	}

	// read two contents of two files
	path := "misc/hashitalk_deploy2023/demos/demo3/ref"

	beforeJob, err := do(nomadClient, path+"/before.nomad")
	if err != nil {
		panic(err)
	}
	afterJob, err := do(nomadClient, path+"/after.nomad")
	if err != nil {
		panic(err)
	}

	patch, err := jsondiff.Compare(beforeJob, afterJob)
	if err != nil {
		panic(err)
	}
	_, err = json.MarshalIndent(patch, "", "    ")
	if err != nil {
		panic(err)
	}
	//os.Stdout.Write(b)

}
func do(nomadClient *api.Client, path string) (*api.Job, error) {
	jobContent, err := os.ReadFile(path)

	if err != nil {
		panic(err)
	}

	job, err := nomadClient.Jobs().ParseHCL(string(jobContent), false)
	if err != nil {
		return nil, err
	}

	jobJson, err := json.MarshalIndent(job, "", "  ")
	// remove all line matching: "\w+"\s*:\s*""|null|0|false,? (poor man's omitempty)
	//jobJson = regexp.MustCompile(omitemptyregex).ReplaceAll(jobJson, []byte(""))

	if err != nil {
		return nil, err
	}

	os.WriteFile(path+".json", jobJson, 0644)

	return job, nil
}
