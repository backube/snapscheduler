package controller

import (
	"github.com/backube/snap-scheduler/pkg/controller/snapshotschedule"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, snapshotschedule.Add)
}
