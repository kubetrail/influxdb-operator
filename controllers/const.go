package controllers

const (
	finalizer                     = "influxdb.kubetrail.io/finalizer"
	reasonObjectInitialized       = "objectInitialized"
	reasonObjectMarkedForDeletion = "objectMarkedForDeletion"
	reasonFinalizerAdded          = "finalizerAdded"
	reasonCreatedBucket           = "createdBucket"
	reasonDeletedBucket           = "deletedBucket"
	reasonCreatedOrganization     = "createdOrganization"
	reasonDeletedOrganization     = "deletedOrganization"
	reasonCreatedToken            = "createdToken"
	reasonDeletedToken            = "deletedToken"
	phasePending                  = "pending"
	phaseReady                    = "ready"
	phaseTerminating              = "terminating"
	conditionTypeObject           = "object"
	conditionTypeInfluxdb         = "influxdb"
)

const (
	configInfluxdb = "default"
	keyToken       = "token"
	keyTokenId     = "tokenId"
)
