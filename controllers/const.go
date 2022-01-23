package controllers

const (
	finalizer                     = "influxdb.kubetrail.io/finalizer"
	label                         = "influxdb.kubetrail.io/group"
	reasonObjectInitialized       = "objectInitialized"
	reasonObjectMarkedForDeletion = "objectMarkedForDeletion"
	reasonFinalizerAdded          = "finalizerAdded"
	reasonSynced                  = "synced"
	reasonDeleted                 = "deleted"
	phasePending                  = "pending"
	phaseRunning                  = "running"
	phaseError                    = "error"
	phaseTerminating              = "terminating"
	conditionTypeObject           = "object"
	conditionTypeRuntime          = "runtime"
)
