package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

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

	"github.com/go-logr/logr"
	osappsv1 "github.com/openshift/api/apps/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"
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
		MetricsBindAddr:    ":8383",
		MetricsPath:        "metrics",
		MetricsServiceName: "deployment-validation-operator-metrics",
		ProbeAddr:          ":8081",
		ConfigFile:         "config/deployment-validation-operator-config.yaml",
	}

	if err := opts.Process(); err != nil {
		fmt.Fprintf(os.Stdout, "processing options: %v\n", err)

		os.Exit(1)
	}

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

	mgr, err := manager.New(cfg, mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("initializing manager: %w", err)
	}

	if err := setupProbes(mgr, opts); err != nil {
		return nil, fmt.Errorf("setting up probes: %w", err)
	}

	log.Info("Registering Components")

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("initializing discovery client: %w", err)
	}

	gr, err := controller.NewGenericReconciler(mgr.GetClient(), discoveryClient)
	if err != nil {
		return nil, fmt.Errorf("initializing generic reconciler: %w", err)
	}

	if err := gr.AddToManager(mgr); err != nil {
		return nil, fmt.Errorf("adding generic reconciler to manager: %w", err)
	}

	if err := setupComponents(log, mgr, opts); err != nil {
		return nil, fmt.Errorf("setting up components: %w", err)
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
		LeaderElection:             opts.EnableLeaderElection,
		LeaderElectionID:           "23h85e23.deployment-validation-operator-lock",
		LeaderElectionNamespace:    opts.LeaderElectionNamespace,
		LeaderElectionResourceLock: "leases",
		Namespace:                  ns,
		HealthProbeBindAddress:     opts.ProbeAddr,
		MetricsBindAddress:         "0", // disable controller-runtime managed prometheus endpoint
		// disable caching of everything
		NewClient: newClient,
		Scheme:    scheme,
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

func newClient(_ cache.Cache, cfg *rest.Config, opts client.Options, _ ...client.Object) (client.Client, error) {
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

func setupProbes(mgr manager.Manager, opts options.Options) error {
	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		return fmt.Errorf("adding healthz check: %w", err)
	}

	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		return fmt.Errorf("adding readyz check: %w", err)
	}

	return nil
}

func setupComponents(log logr.Logger, mgr manager.Manager, opts options.Options) error {
	log.Info("Initializing Prometheus Registry")

	reg, err := dvo_prom.NewRegistry()
	if err != nil {
		return fmt.Errorf("initializing prometheus registry: %w", err)
	}

	log.Info(fmt.Sprintf("Initializing Prometheus metrics endpoint on %q", opts.MetricsEndpoint()))

	svcURL := &url.URL{
		Scheme: "http",
		Host:   opts.MetricsServiceName,
	}
	if parts := strings.Split(opts.MetricsBindAddr, ":"); len(parts) > 0 {
		if len(parts) > 1 {
			svcURL.Host += parts[len(parts)-1]
		}
	}

	srv, err := dvo_prom.NewServer(reg,
		dvo_prom.WithMetricsAddr(opts.MetricsBindAddr),
		dvo_prom.WithMetricsPath(opts.MetricsPath),
		dvo_prom.WithServiceURL(svcURL.String()),
	)
	if err != nil {
		return fmt.Errorf("initializing metrics server: %w", err)
	}

	go func() {
		<-mgr.Elected()
		srv.Ready()
	}()

	if err := mgr.Add(srv); err != nil {
		return fmt.Errorf("adding metrics server to manager: %w", err)
	}

	log.Info("Initializing Validation Engine")

	if err := validations.InitializeValidationEngine(opts.ConfigFile, reg); err != nil {
		return fmt.Errorf("initializing validation engine: %w", err)
	}

	return nil
}
