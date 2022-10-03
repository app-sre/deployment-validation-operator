package controller

import "time"

const (
	// default interval to run validations in.
	// A 10 percent jitter will be added to the reconcile interval between reconcilers,
	// so that not all reconcilers will not send list requests simultaneously.
	defaultReconcileInterval = 5 * time.Minute

	// default number of resources retrieved from the api server per list request
	// the usage of list-continue mechanism ensures that the memory consumption
	// by this operator always stays under a desired threshold irrespective of the
	// number of resource instances for any kubernetes resource
	defaultListLimit = 5

	// EnvNamespaceIgnorePattern sets the pattern for ignoring namespaces from the list of namespaces
	// that are in the validate list of this operator
	EnvNamespaceIgnorePattern string = "NAMESPACE_IGNORE_PATTERN"

	// EnvValidationCheckInterval overrides defaultReconcileInterval
	EnvValidationCheckInterval string = "VALIDATION_CHECK_INTERVAL"

	// EnvResorucesPerListQuery overrides defaultListLimit
	EnvResorucesPerListQuery string = "RESOURCES_PER_LIST_QUERY"
)
