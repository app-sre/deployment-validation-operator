package osde2etests

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega" //nolint:golint
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachinerylabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

var _ = ginkgo.Describe("DVO", ginkgo.Ordered, func() {
	const (
		namespaceName  = "openshift-deployment-validation-operator"
		deploymentName = "deployment-validation-operator"
	)

	var k8s *resources.Resources

	ginkgo.BeforeAll(func() {
		// setup the k8s client
		cfg, err := config.GetConfig()
		Expect(err).Should(BeNil(), "failed to get kubeconfig")
		k8s, err = resources.New(cfg)
		Expect(err).Should(BeNil(), "resources.New error")
	})

	ginkgo.It("exists and is running", func(ctx context.Context) {
		const serviceName = "deployment-validation-operator-metrics"
		clusterRoles := []string{
			"deployment-validation-operator-og-admin",
			"deployment-validation-operator-og-edit",
			"deployment-validation-operator-og-view",
		}

		err := k8s.Get(ctx, namespaceName, namespaceName, &v1.Namespace{})
		Expect(err).Should(BeNil(), "unable to get namespace %s", namespaceName)

		err = k8s.Get(ctx, serviceName, namespaceName, &v1.Service{})
		Expect(err).Should(BeNil(), "unable to get service %s", serviceName)

		for _, clusterRoleName := range clusterRoles {
			err = k8s.Get(ctx, clusterRoleName, "", &rbacv1.ClusterRole{})
			Expect(err).Should(BeNil(), "unable to get clusterrole %s", clusterRoleName)
		}

		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: deploymentName, Namespace: namespaceName},
		}
		deploymentAvailable := conditions.New(k8s).
			DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, v1.ConditionTrue)
		err = wait.For(deploymentAvailable, wait.WithTimeout(10*time.Second))
		Expect(err).Should(BeNil(), "deployment %s never became available", deploymentName)
	})

	ginkgo.Context("validates", func() {
		var namespace *v1.Namespace

		ginkgo.BeforeEach(func(ctx context.Context) {
			namespace = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "osde2e-dvo-"}}
			err := k8s.Create(ctx, namespace)
			Expect(err).Should(BeNil(), "unable to create test namespace %s", namespace.GetName())

			labels := map[string]string{"app": "test"}
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{GenerateName: "osde2e-", Namespace: namespace.GetName()},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: labels},
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Labels: labels},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{Name: "pause", Image: "registry.k8s.io/pause:latest"},
							},
						},
					},
				},
			}
			err = k8s.Create(ctx, deployment)
			Expect(err).Should(BeNil(), "unable to create deployment %s", deployment.GetName())
			deploymentAvailable := conditions.New(k8s).
				DeploymentConditionMatch(deployment, appsv1.DeploymentAvailable, v1.ConditionTrue)
			err = wait.For(deploymentAvailable)
			Expect(err).Should(BeNil(), "deployment %s never became available", deployment.GetName())

			ginkgo.DeferCleanup(k8s.Delete, namespace)
		})

		ginkgo.It("new deployments", func(ctx context.Context) {
			//nolint:lll
			validationMsg := fmt.Sprintf("\"msg\":\"Set memory requests and limits for your container based on its requirements. Refer to https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#requests-and-limits for details.\",\"request.namespace\":%q", namespace.GetName())

			// Wait for the deployment logs to contain the validation message
			Eventually(ctx, func(ctx context.Context) (string, error) {
				pods := &v1.PodList{}
				lbls := apimachinerylabels.FormatLabels(map[string]string{"app": deploymentName})
				err := k8s.List(ctx, pods, resources.WithLabelSelector(lbls))
				if err != nil {
					return "", err
				}
				if len(pods.Items) < 1 {
					return "", fmt.Errorf("unable to find pod for deployment %s", deploymentName)
				}
				clientset, err := kubernetes.NewForConfig(k8s.GetConfig())
				if err != nil {
					return "", err
				}
				req := clientset.CoreV1().Pods(namespaceName).
					GetLogs(pods.Items[0].GetName(), &v1.PodLogOptions{})
				logs, err := req.DoRaw(ctx)
				if err != nil {
					return "", err
				}
				return string(logs), nil
			}).
				WithPolling(30 * time.Second).
				WithTimeout(10 * time.Minute).
				Should(ContainSubstring(validationMsg))
		})
	})
})
