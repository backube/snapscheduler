package controller

import (
	"github.com/backube/SnapScheduler/pkg/controller/snapshotpolicy"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, snapshotpolicy.Add)
}
