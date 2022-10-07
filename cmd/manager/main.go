package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	dv_config "github.com/app-sre/deployment-validation-operator/config"
	"github.com/app-sre/deployment-validation-operator/pkg/apis"
	"github.com/app-sre/deployment-validation-operator/pkg/controller"
	dvo_prom "github.com/app-sre/deployment-validation-operator/pkg/prometheus"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/app-sre/deployment-validation-operator/version"

	"github.com/go-logr/logr"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const operatorNameEnvVar = "OPERATOR_NAME"

func main() {
	// Make sure the operator name is what we want
	os.Setenv(operatorNameEnvVar, dv_config.OperatorName)

	opts := options{
		MetricsPort: 8383,
		MetricsPath: "metrics",
		ConfigFile:  "config/deployment-validation-operator-config.yaml",
	}

	opts.Process()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.New(zap.UseFlagOptions(&opts.Zap)))

	log := logf.Log.WithName("DeploymentValidation")

	log.Info("Setting Up Manager")

	mgr, err := setupManager(log, opts)
	if err != nil {
		fail(log, err, "Unexpected error occurred while setting up manager")
	}

	log.Info("Starting Manager")

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		fail(log, err, "Unexpected error occurred while running manager")
	}
}

func setupManager(log logr.Logger, opts options) (manager.Manager, error) {
	logVersion(log)

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	mgrOpts, err := getManagerOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("getting manager options: %w", err)
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("initializing manager: %w", err)
	}

	log.Info("Registering Components")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, fmt.Errorf("adding APIs to scheme: %w", err)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("initializing discovery client: %w", err)
	}

	gr, err := controller.NewGenericReconciler(mgr.GetClient(), discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("initializing generic reconciler: %w", err)
	}

	if err = gr.AddToManager(mgr); err != nil {
		return nil, fmt.Errorf("adding generic reconciler to manager: %w", err)
	}

	log.Info(fmt.Sprintf("Initializing Prometheus metrics endpoint on %q", opts.MetricsEndpoint()))
	dvo_prom.InitMetricsEndpoint(opts.MetricsPath, opts.MetricsPort)

	log.Info("Initializing Validation Engine")
	if err := validations.InitializeValidationEngine(opts.ConfigFile); err != nil {
		return nil, fmt.Errorf("initializing validation engine: %w", err)
	}

	return mgr, nil
}

func fail(log logr.Logger, err error, msg string) {
	log.Error(err, msg)

	os.Exit(1)
}

func logVersion(log logr.Logger) {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
}

var errWatchNamespaceNotSet = errors.New("'WatchNamespace' not set")

func getManagerOptions(opts options) (manager.Options, error) {
	ns, ok := opts.GetWatchNamespace()
	if !ok {
		return manager.Options{}, errWatchNamespaceNotSet
	}

	mgrOpts := manager.Options{
		Namespace:          ns,
		MetricsBindAddress: "0", // disable controller-runtime managed prometheus endpoint
		// disable caching of everything
		ClientBuilder: &newUncachedClientBuilder{},
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(ns, ",") {
		mgrOpts.Namespace = ""
		mgrOpts.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(ns, ","))
	}

	return mgrOpts, nil
}

type newUncachedClientBuilder struct {
	uncached []client.Object
}

func (n *newUncachedClientBuilder) WithUncached(objs ...client.Object) manager.ClientBuilder {
	n.uncached = append(n.uncached, objs...)
	return n
}

func (n *newUncachedClientBuilder) Build(
	cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
	// Directly use the API client, without wrapping it in a delegatingClient for cache access.
	qps, err := kubeClientQPS()
	if err != nil {
		return nil, err
	}
	config.QPS = qps
	return client.New(config, options)
}

func kubeClientQPS() (float32, error) {
	qps := controller.DefaultKubeClientQPS
	envVal, ok := os.LookupEnv(controller.EnvKubeClientQPS)
	if !ok {
		return qps, nil
	}
	val, err := strconv.ParseFloat(envVal, 32)
	if err != nil {
		return 0.0, err
	}
	qps = float32(val)
	return qps, err
}
