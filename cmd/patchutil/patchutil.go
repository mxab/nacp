package main

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
)

func main() {

	fmt.Printf("Hello, world.\n")

	nomadClient, err := api.NewClient(&api.Config{})
	if err != nil {
		panic(err)
	}

}
