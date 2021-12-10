package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	dv_config "github.com/app-sre/deployment-validation-operator/config"
	"github.com/app-sre/deployment-validation-operator/pkg/apis"
	"github.com/app-sre/deployment-validation-operator/pkg/controller"
	dvo_prom "github.com/app-sre/deployment-validation-operator/pkg/prometheus"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/app-sre/deployment-validation-operator/version"

	"github.com/operator-framework/operator-lib/leader"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/spf13/pflag"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsPort       int32  = 8383
	metricsPath       string = "metrics"
	defaultConfigFile        = "config/deployment-validation-operator-config.yaml"
)

var log = logf.Log.WithName("DeploymentValidation")

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

func main() {
	var configFile string
	var inCluster bool

	// Make sure the operator name is what we want
	os.Setenv("OPERATOR_NAME", dv_config.OperatorName)

	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	opts := zap.Options{}
	opts.BindFlags(flag.CommandLine)

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// Add app specific flags
	appFlags := pflag.NewFlagSet("dvo", pflag.ExitOnError)
	appFlags.StringVar(&configFile, "config", defaultConfigFile, "Path to config file")
	appFlags.BoolVarP(&inCluster, "in-cluster", "", true, "Set to false if running outside a cluster")
	pflag.CommandLine.AddFlagSet(appFlags)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logger := zap.New(zap.UseFlagOptions(&opts))
	logf.SetLogger(logger)

	printVersion()

	namespace, err := getWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Failed to get config")
		os.Exit(1)
	}

	if inCluster {
		// leader.Become looks for the operator namespace using a hard-coded
		// path that assumes it is running inside a cluster and no workaround
		// is immediately obvious.  If running outside the cluster then the
		// file wouldn't normally exist and this call will fail.  If running
		// outside a cluster, then leader election probably isn't that
		// important anyway so we don't even try to do it.
		ctx := context.TODO()
		// Become the leader before proceeding
		err = leader.Become(ctx, "deployment-validation-operator-lock")
		if err != nil {
			log.Error(err, "Failed to get leader lock")
			os.Exit(1)
		}
	}

	// Set default manager options
	options := manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: "0", // disable controller-runtime managed prometheus endpoint
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "Failed to create manager")
		os.Exit(1)
	}

	log.Info("Registering Components")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "Failed to add api schemes")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddControllersToManager(mgr); err != nil {
		log.Error(err, "Failed to setup the controllers")
		os.Exit(1)
	}

	log.Info(fmt.Sprintf("Initializing Prometheus metrics endpoint on %s", getFullMetricsEndpoint()))
	dvo_prom.InitMetricsEndpoint(metricsPath, metricsPort)

	log.Info("Initializing Validation Engine")
	if err := validations.InitializeValidationEngine(configFile); err != nil {
		log.Error(err, "Failed to initialize validation engine")
		os.Exit(1)
	}

	log.Info("Starting")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func getFullMetricsEndpoint() string {
	return fmt.Sprintf("http://0.0.0.0:%d/%s", metricsPort, metricsPath)
}

func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}
