package exoscale

import (
	"github.com/exoscale/egoscale"
)

// partialCommand represents an update command, it's made of
// the partial key which is expected to change and the
// request that has to be run.
type partialCommand struct {
	partial  string
	partials []string
	request  egoscale.Command
}
