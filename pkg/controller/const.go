package controller

const (

	// defaultKubeClientQPS defines the default Queries Per Second (QPS) of the kubeclient used by the operator
	DefaultKubeClientQPS = float32(0.5)

	// default number of resources retrieved from the api server per list request
	// the usage of list-continue mechanism ensures that the memory consumption
	// by this operator always stays under a desired threshold irrespective of the
	// number of resource instances for any kubernetes resource
	defaultListLimit = 20

	// EnvKubeClientQPS overrides defaultKubeClientQPS
	EnvKubeClientQPS string = "KUBECLIENT_QPS"

	// EnvResorucesPerListQuery overrides defaultListLimit
	EnvResorucesPerListQuery string = "RESOURCES_PER_LIST_QUERY"

	// EnvNamespaceIgnorePattern sets the pattern for ignoring namespaces from the list of namespaces
	// that are in the validate list of this operator
	EnvNamespaceIgnorePattern string = "NAMESPACE_IGNORE_PATTERN"
)
