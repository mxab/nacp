package types

import (
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/config"
)

type Payload struct {
	Job     *api.Job               `json:"job"`
	Context *config.RequestContext `json:"context,omitempty"`
}
