package v1beta1

const (
	defaultOrgName         = "influxdata"
	defaultConfigName      = "default"
	defaultSecretName      = "influxdb-token"
	defaultSecretNamespace = "influxdb-system"
	defaultAddr            = "http://influxdb.influxdb-system.svc.cluster.local"
)

const (
	PermissionRead  = "read"
	PermissionWrite = "write"
)

// ResourceType const
const (
	ResourceTypeAuthorizations        = "authorizations"
	ResourceTypeBuckets               = "buckets"
	ResourceTypeChecks                = "checks"
	ResourceTypeDashboards            = "dashboards"
	ResourceTypeDbrp                  = "dbrp"
	ResourceTypeDocuments             = "documents"
	ResourceTypeLabels                = "labels"
	ResourceTypeNotebooks             = "notebooks"
	ResourceTypeNotificationEndpoints = "notificationEndpoints"
	ResourceTypeNotificationRules     = "notificationRules"
	ResourceTypeOrgs                  = "orgs"
	ResourceTypeScrapers              = "scrapers"
	ResourceTypeSecrets               = "secrets"
	ResourceTypeSources               = "sources"
	ResourceTypeTasks                 = "tasks"
	ResourceTypeTelegrafs             = "telegrafs"
	ResourceTypeUsers                 = "users"
	ResourceTypeVariables             = "variables"
	ResourceTypeViews                 = "views"
)

var resourceTypes = map[string]struct{}{
	ResourceTypeAuthorizations:        {},
	ResourceTypeBuckets:               {},
	ResourceTypeChecks:                {},
	ResourceTypeDashboards:            {},
	ResourceTypeDbrp:                  {},
	ResourceTypeDocuments:             {},
	ResourceTypeLabels:                {},
	ResourceTypeNotebooks:             {},
	ResourceTypeNotificationEndpoints: {},
	ResourceTypeNotificationRules:     {},
	ResourceTypeOrgs:                  {},
	ResourceTypeScrapers:              {},
	ResourceTypeSecrets:               {},
	ResourceTypeSources:               {},
	ResourceTypeTasks:                 {},
	ResourceTypeTelegrafs:             {},
	ResourceTypeUsers:                 {},
	ResourceTypeVariables:             {},
	ResourceTypeViews:                 {},
}

var permissionTypes = map[string]struct{}{
	PermissionRead:  {},
	PermissionWrite: {},
}
