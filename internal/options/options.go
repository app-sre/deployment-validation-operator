package options

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	MetricsPort    int32
	MetricsPath    string
	ProbeAddr      string
	ConfigFile     string
	watchNamespace *string
	Zap            zap.Options
}

func (o *Options) MetricsEndpoint() string {
	return fmt.Sprintf("http://0.0.0.0:%d/%s", o.MetricsPort, o.MetricsPath)
}

func (o *Options) GetWatchNamespace() (string, bool) {
	if o.watchNamespace == nil {
		return "", false
	}

	return *o.watchNamespace, true
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

// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
// which specifies the Namespace to watch.
// An empty value means the operator is running with cluster scope.
const watchNamespaceEnvVar = "WATCH_NAMESPACE"

func (o *Options) processEnv() {
	if val, ok := os.LookupEnv(watchNamespaceEnvVar); ok {
		o.watchNamespace = &val
	}
}
