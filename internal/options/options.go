package options

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	ClientQPS              float32
	MetricsPort            int32 // bit alignment takes precedence over lexicographic order here
	ConfigFile             string
	MetricsPath            string
	namespaceIgnorePattern *string
	ProbeAddr              string
	resourceListLimit      *int
	WatchNamespaces        []string
	Zap                    zap.Options
}

func (o *Options) MetricsEndpoint() string {
	return fmt.Sprintf("http://0.0.0.0:%d/%s", o.MetricsPort, o.MetricsPath)
}

func (o *Options) GetNamespaceIgnorePattern() (string, bool) {
	if o.namespaceIgnorePattern == nil {
		return "", false
	}

	return *o.namespaceIgnorePattern, true
}

func (o *Options) GetResourceListLimit() (int, bool) {
	if o.resourceListLimit == nil {
		return 0, false
	}

	return *o.resourceListLimit, true
}

func (o *Options) Process() {
	o.processFlags()
	o.processEnv()
}

func (o *Options) processFlags() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	o.Zap.BindFlags(flag.CommandLine)

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Add app specific flags
	flags := pflag.NewFlagSet("dvo", pflag.ExitOnError)
	flags.StringVar(
		&o.ConfigFile,
		"config", o.ConfigFile,
		"Path to config file",
	)
	flags.StringVar(
		&o.ProbeAddr,
		"health-probe-bind-address", o.ProbeAddr,
		"The address the probe endpoint binds to.",
	)

	pflag.CommandLine.AddFlagSet(flags)

	pflag.Parse()
}

const (
	// clientQPSEnvVar specifies the environment variable which
	// may contain an alternate value for client QPS.
	clientQPSEnvVar = "KUBECLIENT_QPS"
	// namespaceIgnorePatternEnvVar specifies the environment variable
	// which may contain a regex pattern to apply when choosing
	// namespaces to filter from reconciliation.
	namespaceIgnorePatternEnvVar = "NAMESPACE_IGNORE_PATTERN"
	// resourcesPerListQueryEnvVar specifies the environment variable
	// which may contain an alternate value for list limits applied
	// when listing resources during reconciliation.
	resorucesPerListQueryEnvVar = "RESOURCES_PER_LIST_QUERY"
	// watchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	watchNamespaceEnvVar = "WATCH_NAMESPACE"
)

func (o *Options) processEnv() {
	if val, ok := os.LookupEnv(clientQPSEnvVar); ok {
		if f, err := strconv.ParseFloat(val, 32); err == nil {
			o.ClientQPS = float32(f)
		}
	}

	if val, ok := os.LookupEnv(namespaceIgnorePatternEnvVar); ok {
		o.namespaceIgnorePattern = &val
	}

	if val, ok := os.LookupEnv(resorucesPerListQueryEnvVar); ok {
		o.resourceListLimit = maybeParseint(val)
	}

	if val, ok := os.LookupEnv(watchNamespaceEnvVar); ok {
		o.WatchNamespaces = strings.Split(val, ",")
	}
}

func maybeParseint(s string) *int {
	if i, err := strconv.Atoi(s); err == nil {
		return &i
	}

	return nil
}

func (o *Options) ToLogValues() []interface{} {
	return []interface{}{
		"ClientQPS", o.ClientQPS,
		"ConfigFile", o.ConfigFile,
		"MetricsPort", o.MetricsPort,
		"MetricsPath", o.MetricsPath,
		"NamespaceIgnorePattern", handleUnset(o.GetNamespaceIgnorePattern()),
		"ProbeAddress", o.ProbeAddr,
		"ResourceListLimit", handleUnset(o.GetResourceListLimit()),
		"WatchNamespaces", o.WatchNamespaces,
	}
}

func handleUnset(val interface{}, ok bool) interface{} {
	if !ok {
		return "<nil>"
	}

	return val
}
