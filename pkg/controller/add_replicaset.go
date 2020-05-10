package controller

import (
	"github.com/jmelis/dv-operator/pkg/controller/replicaset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, replicaset.Add)
}
