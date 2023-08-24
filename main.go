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

	apis "github.com/app-sre/deployment-validation-operator/api"
	dvconfig "github.com/app-sre/deployment-validation-operator/config"
	"github.com/app-sre/deployment-validation-operator/internal/options"
	"github.com/app-sre/deployment-validation-operator/pkg/configmap"
	"github.com/app-sre/deployment-validation-operator/pkg/controller"
	dvoProm "github.com/app-sre/deployment-validation-operator/pkg/prometheus"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/app-sre/deployment-validation-operator/version"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-logr/logr"
	osappsv1 "github.com/openshift/api/apps/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const operatorNameEnvVar = "OPERATOR_NAME"

func main() {
	// Make sure the operator name is what we want
	os.Setenv(operatorNameEnvVar, dvconfig.OperatorName)

	opts := options.Options{
		MetricsPort: 8383,
		MetricsPath: "metrics",
		ProbeAddr:   ":8081",
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

func setupManager(log logr.Logger, opts options.Options) (manager.Manager, error) {
	logVersion(log)

	log.Info("Load KubeConfig")

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	log.Info("Initialize Manager")

	mgr, err := initManager(log, opts, cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing manager: %w", err)
	}

	log.Info("Registering Components")

	log.Info("Initialize Prometheus Registry")

	reg := prometheus.NewRegistry()
	metrics, err := dvoProm.PreloadMetrics(reg)
	if err != nil {
		return nil, fmt.Errorf("preloading kube-linter metrics: %w", err)
	}

	log.Info(fmt.Sprintf("Initialize Prometheus metrics endpoint on %q", opts.MetricsEndpoint()))

	srv, err := dvoProm.NewServer(reg, opts.MetricsPath, fmt.Sprintf(":%d", opts.MetricsPort))
	if err != nil {
		return nil, fmt.Errorf("initializing metrics server: %w", err)
	}

	if err := mgr.Add(srv); err != nil {
		return nil, fmt.Errorf("adding metrics server to manager: %w", err)
	}

	log.Info("Initialize ConfigMap watcher")

	cmWatcher, err := configmap.NewWatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing configmap watcher: %w", err)
	}

	if err := mgr.Add(cmWatcher); err != nil {
		return nil, fmt.Errorf("adding configmap watcher to manager: %w", err)
	}

	log.Info("Initialize Validation Engine")

	validationEngine, err := validations.NewEngine(opts.ConfigFile, cmWatcher, metrics)
	if err != nil {
		return nil, fmt.Errorf("initializing validation engine: %w", err)
	}

	if err := mgr.Add(validationEngine); err != nil {
		return nil, fmt.Errorf("adding validationEngine to manager: %w", err)
	}

	log.Info("Initialize Reconciler")

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

func initializeScheme() (*k8sruntime.Scheme, error) {
	scheme := k8sruntime.NewScheme()

	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding client-go APIs to scheme: %w", err)
	}

	if err := osappsv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding OpenShift Apps V1 API to scheme: %w", err)
	}

	if err := apis.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("adding DVO APIs to scheme: %w", err)
	}

	return scheme, nil
}

var errWatchNamespaceNotSet = errors.New("'WatchNamespace' not set")

func getManagerOptions(scheme *k8sruntime.Scheme, opts options.Options) (manager.Options, error) {
	ns, ok := opts.GetWatchNamespace()
	if !ok {
		return manager.Options{}, errWatchNamespaceNotSet
	}

	mgrOpts := manager.Options{
		Namespace:              ns,
		HealthProbeBindAddress: opts.ProbeAddr,
		MetricsBindAddress:     "0", // disable controller-runtime managed prometheus endpoint
		// disable caching of everything
		NewClient: newClient,
		Scheme:    scheme,
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if strings.Contains(ns, ",") {
		mgrOpts.Cache.Namespaces = strings.Split(ns, ",")
	}

	return mgrOpts, nil
}

func newClient(cfg *rest.Config, opts client.Options) (client.Client, error) {
	qps, err := kubeClientQPS()
	if err != nil {
		return nil, err
	}

	cfg.QPS = qps

	return client.New(cfg, opts)
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

func initManager(log logr.Logger, opts options.Options, cfg *rest.Config) (manager.Manager, error) {
	log.Info("Initialize Scheme")
	scheme, err := initializeScheme()
	if err != nil {
		return nil, fmt.Errorf("initializing scheme: %w", err)
	}

	log.Info("Getting Manager Options")
	mgrOpts, err := getManagerOptions(scheme, opts)
	if err != nil {
		return nil, fmt.Errorf("getting manager options: %w", err)
	}

	mgr, err := manager.New(cfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("getting new manager: %w", err)
	}

	log.Info("Adding Healthz and Readyz checks")
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding healthz check: %w", err)
	}

	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding readyz check: %w", err)
	}

	return mgr, nil
}
