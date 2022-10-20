package options

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type Options struct {
	EnableLeaderElection    bool
	LeaderElectionNamespace string
	MetricsBindAddr         string
	MetricsPath             string
	MetricsServiceName      string
	ProbeAddr               string
	ConfigFile              string
	watchNamespace          *string
	Zap                     zap.Options
}

func (o *Options) MetricsEndpoint() string {
	endpoint := &url.URL{
		Scheme: "http",
		Path:   o.MetricsPath,
	}

	if addr := o.MetricsBindAddr; strings.HasPrefix(addr, ":") {
		endpoint.Host = "0.0.0.0" + o.MetricsBindAddr
	} else {
		endpoint.Host = o.MetricsBindAddr
	}

	return endpoint.String()
}

func (o *Options) GetWatchNamespace() (string, bool) {
	if o.watchNamespace == nil {
		return "", false
	}

	return *o.watchNamespace, true
}

func (o *Options) Process() error {
	o.processFlags()
	o.processEnv()
	o.processSecrets()

	if err := o.validate(); err != nil {
		return fmt.Errorf("validating options: %w", err)
	}

	return nil
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
	flags.BoolVar(
		&o.EnableLeaderElection,
		"enable-leader-election", o.EnableLeaderElection,
		"Enables Leader Election when starting the manager.",
	)
	flags.StringVar(
		&o.LeaderElectionNamespace,
		"leader-election-namespace", o.LeaderElectionNamespace,
		"The namespace used by leader election resources.",
	)
	flags.StringVar(
		&o.MetricsBindAddr,
		"metrics-bind-address", o.MetricsBindAddr,
		"The address the metrics endpoint binds to.",
	)
	flags.StringVar(
		&o.ProbeAddr,
		"health-probe-bind-address", o.ProbeAddr,
		"The address the probe endpoint binds to.",
	)
	flags.StringVar(
		&o.MetricsServiceName,
		"metrics-service-name", o.MetricsServiceName,
		"Name of the service used to load balance metrics",
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

func (o *Options) processSecrets() {
	const (
		scrtsPath              = "/var/run/secrets"
		inClusterNamespacePath = scrtsPath + "/kubernetes.io/serviceaccount/namespace"
	)

	var namespace string

	if ns, err := os.ReadFile(inClusterNamespacePath); err == nil {
		// Avoid applying a garbage value if an error occurred
		namespace = string(ns)
	}

	if o.LeaderElectionNamespace == "" {
		o.LeaderElectionNamespace = namespace
	}
}

var errLeaderElectionNamespaceNotSet = errors.New("leader election namespace not set")

func (o *Options) validate() error {
	if !o.EnableLeaderElection {
		return nil
	}

	if o.LeaderElectionNamespace != "" {
		return nil
	}

	return errLeaderElectionNamespaceNotSet
}
