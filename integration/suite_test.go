package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	internaltesting "github.com/app-sre/deployment-validation-operator/internal/testing"
	"github.com/app-sre/deployment-validation-operator/pkg/apis"
	osappsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	managerUser  = "deployment-validation-operator"
	managerGroup = "deployment-validation-operator"
)

var (
	_binPath        string
	_client         *internaltesting.TestClient
	_kubeConfigPath string
	_testEnv        *envtest.Environment
)

// To run these tests with an external cluster
// set the following environment variables:
// USE_EXISTING_CLUSTER=true
// KUBECONFIG=<path_to_kube.config>.
// The external cluster must have authentication
// enabled on the API server.
func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DVO suite")
}

var _ = BeforeSuite(func() {
	root, err := projectRoot()
	Expect(err).ToNot(HaveOccurred())

	By("Registering schemes")

	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).Should(Succeed())
	Expect(osappsv1.AddToScheme(scheme)).Should(Succeed())
	Expect(apis.AddToScheme(scheme)).Should(Succeed())

	By("Starting test environment")

	_testEnv = &envtest.Environment{
		Scheme: scheme,
	}

	cfg, err := _testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	DeferCleanup(cleanup(_testEnv))

	By("Initializing k8s client")

	client, err := client.New(cfg, client.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	_client = internaltesting.NewTestClient(client)

	By("Applying Cluster Scope RBAC")

	rbac, err := getClusterRBAC(managerGroup)
	Expect(err).ToNot(HaveOccurred())

	for _, obj := range rbac {
		_client.Create(context.Background(), obj)
	}

	By("Building manager binary")

	_binPath, err = gexec.BuildWithEnvironment(
		filepath.Join(root, "cmd", "manager"),
		[]string{"CGO_ENABLED=0"},
	)
	Expect(err).ToNot(HaveOccurred())

	By("writing kube.config")

	user, err := _testEnv.AddUser(
		envtest.User{
			Name:   managerUser,
			Groups: []string{managerGroup},
		},
		nil,
	)
	Expect(err).ToNot(HaveOccurred())

	configFile, err := os.CreateTemp("", "dvo-integration-*")
	Expect(err).ToNot(HaveOccurred())

	data, err := user.KubeConfig()
	Expect(err).ToNot(HaveOccurred())

	_, err = configFile.Write(data)
	Expect(err).ToNot(HaveOccurred())

	_kubeConfigPath = configFile.Name()
})

func cleanup(env *envtest.Environment) func() {
	return func() {
		By("Stopping the test environment")

		Expect(env.Stop()).Should(Succeed())

		By("Cleaning up test artifacts")

		gexec.CleanupBuildArtifacts()

		Expect(remove(_kubeConfigPath)).Should(Succeed())
	}
}

func usingExistingCluster() bool {
	return _testEnv.UseExistingCluster != nil && *_testEnv.UseExistingCluster
}
