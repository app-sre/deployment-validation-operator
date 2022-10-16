package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	dvconfig "github.com/app-sre/deployment-validation-operator/config"
	"github.com/app-sre/deployment-validation-operator/internal/options"
	"github.com/app-sre/deployment-validation-operator/pkg/apis"
	"github.com/app-sre/deployment-validation-operator/pkg/controller"
	dvo_prom "github.com/app-sre/deployment-validation-operator/pkg/prometheus"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/app-sre/deployment-validation-operator/version"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/go-logr/logr"
	osappsv1 "github.com/openshift/api/apps/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
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
		ClientQPS:   0.5,
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

	log.V(1).Info("Running with Options", opts.ToLogValues()...)

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

	log.Info("Initialize Scheme")

	scheme, err := initializeScheme()
	if err != nil {
		return nil, fmt.Errorf("initializing scheme: %w", err)
	}

	log.Info("Initialize Manager")

	mgrOpts, err := getManagerOptions(scheme, opts)
	if err != nil {
		return nil, fmt.Errorf("getting manager options: %w", err)
	}

	mgr, err := ctrl.NewManager(cfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("initializing manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding healthz check: %w", err)
	}

	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		return nil, fmt.Errorf("adding readyz check: %w", err)
	}

	log.Info("Registering Components")

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("initializing discovery client: %w", err)
	}

	grOpts, err := getReconcilierOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("getting reconcilier options: %w", err)
	}

	gr, err := controller.NewGenericReconciler(mgr.GetClient(), discoveryClient, grOpts...)
	if err != nil {
		return nil, fmt.Errorf("initializing generic reconciler: %w", err)
	}

	if err = gr.AddToManager(mgr); err != nil {
		return nil, fmt.Errorf("adding generic reconciler to manager: %w", err)
	}

	log.Info("Initializing Prometheus Registry")

	reg := prometheus.NewRegistry()

	log.Info(fmt.Sprintf("Initializing Prometheus metrics endpoint on %q", opts.MetricsEndpoint()))

	srv, err := dvo_prom.NewServer(reg, opts.MetricsPath, fmt.Sprintf(":%d", opts.MetricsPort))
	if err != nil {
		return nil, fmt.Errorf("initializing metrics server: %w", err)
	}

	if err := mgr.Add(srv); err != nil {
		return nil, fmt.Errorf("adding metrics server to manager: %w", err)
	}

	log.Info("Initializing Validation Engine")

	if err := validations.InitializeValidationEngine(opts.ConfigFile, reg); err != nil {
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

func getManagerOptions(scheme *k8sruntime.Scheme, opts options.Options) (ctrl.Options, error) {
	mgrOpts := ctrl.Options{
		HealthProbeBindAddress: opts.ProbeAddr,
		MetricsBindAddress:     "0", // disable controller-runtime managed prometheus endpoint
		// disable caching of everything
		NewClient: newClientWithOpts(opts),
		Scheme:    scheme,
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	// More: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/cache#MultiNamespacedCacheBuilder
	if nss := opts.WatchNamespaces; len(nss) > 1 {
		mgrOpts.NewCache = cache.MultiNamespacedCacheBuilder(nss)
	} else if len(nss) > 0 {
		mgrOpts.Namespace = nss[0]
	}

	return mgrOpts, nil
}

func newClientWithOpts(options options.Options) cluster.NewClientFunc {
	return func(_ cache.Cache, cfg *rest.Config, opts client.Options, _ ...client.Object) (client.Client, error) {
		cfg.QPS = options.ClientQPS

		return client.New(cfg, opts)
	}
}

func getReconcilierOptions(opts options.Options) ([]controller.GenericReconcilerOption, error) {
	var pattern *regexp.Regexp

	if patStr, ok := opts.GetNamespaceIgnorePattern(); ok {
		var err error

		pattern, err = regexp.Compile(patStr)
		if err != nil {
			return nil, fmt.Errorf("compiling namespace ignore pattern: %w", err)
		}
	}

	grOpts := []controller.GenericReconcilerOption{
		controller.WithLog{Log: logf.Log.WithName("controllers").WithName("generic-reconcilier")},
		controller.WithNamespaceCache{
			Cache: controller.NewWatchNamespacesCache(
				controller.WithIgnorePattern{Pattern: pattern},
			),
		},
	}

	if limit, ok := opts.GetResourceListLimit(); ok {
		grOpts = append(grOpts, controller.WithListLimit(limit))
	}

	return grOpts, nil
}
