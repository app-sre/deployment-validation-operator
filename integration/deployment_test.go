package integration

import (
	"context"
	"errors"
	"os/exec"

	internaltesting "github.com/app-sre/deployment-validation-operator/internal/testing"
	io_prometheus_client "github.com/prometheus/client_model/go"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ErrNilMetric = errors.New("nil metric")

var _ = Describe("deployments", func() {
	var (
		ctx          context.Context
		cancel       context.CancelFunc
		namespace    string
		namespaceGen = nameGenerator("deployment-test-namespace")
	)

	prom := internaltesting.NewPromClient()

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		namespace = namespaceGen()

		By("Starting manager")

		manager := exec.Command(_binPath,
			"--kubeconfig", _kubeConfigPath,
		)

		manager.Env = []string{
			"WATCH_NAMESPACE=" + namespace,
		}

		session, err := Start(manager, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())

		By("Creating the watch namespace")

		ns := newNamespace(namespace)

		_client.Create(ctx, &ns)

		rbac, err := getRBAC(namespace, managerGroup)
		Expect(err).ToNot(HaveOccurred())

		for _, obj := range rbac {
			_client.Create(ctx, obj)
		}

		DeferCleanup(func() {
			cancel()

			By("Stopping the managers")

			session.Interrupt()

			if usingExistingCluster() {
				By("Deleting watch namspace")

				_client.Delete(ctx, &ns)
			}
		})
	})

	When("created with missing resource requirements", func() {
		It("should generate metrics", func() {
			_client.Create(ctx, newDeployment("test", namespace))

			var (
				deploymentMemReqMetric *io_prometheus_client.Metric
				deploymentCPUReqMetric *io_prometheus_client.Metric
			)

			Eventually(func() error {
				metrics, err := prom.GetDVOMetrics(ctx, "http://localhost:8383/metrics")
				if err != nil {
					return err
				}

				memReqMetrics := metrics["unset_memory_requirements"]
				deploymentMemReqMetric = searchableMetrics(
					memReqMetrics,
				).FindDVOMetric("Deployment", "test", namespace)
				if deploymentMemReqMetric == nil {
					return ErrNilMetric
				}

				cpuReqMetrics := metrics["unset_cpu_requirements"]
				deploymentCPUReqMetric = searchableMetrics(
					cpuReqMetrics,
				).FindDVOMetric("Deployment", "test", namespace)
				if deploymentCPUReqMetric == nil {
					return ErrNilMetric
				}

				return nil
			}, "10s").Should(Succeed())

			Expect(deploymentMemReqMetric.Gauge.GetValue()).To(Equal(float64(1)))
			Expect(deploymentCPUReqMetric.Gauge.GetValue()).To(Equal(float64(1)))
		})
	})

})

func newNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

func newDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test",
							Image: "test",
						},
					},
				},
			},
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }

type searchableMetrics []*io_prometheus_client.Metric

func (ms searchableMetrics) FindDVOMetric(kind, name, namespace string) *io_prometheus_client.Metric {
	for _, m := range ms {
		labels := searchableLables(m.GetLabel())

		if labels.Contains("kind", kind) &&
			labels.Contains("name", name) &&
			labels.Contains("namespace", namespace) {
			return m
		}
	}

	return nil
}

type searchableLables []*io_prometheus_client.LabelPair

func (ls searchableLables) Contains(name, val string) bool {
	for _, l := range ls {
		if l.GetName() != name {
			continue
		}

		if l.GetValue() == val {
			return true
		}
	}

	return false
}
