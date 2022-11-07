package integration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	internaltesting "github.com/app-sre/deployment-validation-operator/internal/testing"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func nameGenerator(pfx string) func() string {
	i := 0

	return func() string {
		name := fmt.Sprintf("%s-%d", pfx, i)

		i++

		return name
	}
}

func getClusterRBAC(group string) ([]client.Object, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, "deploy", "openshift", "cluster-role.yaml")

	role, err := internaltesting.LoadUnstructuredFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading role: %w", err)
	}

	return []client.Object{
		role,
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: role.GetName(),
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: group,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: role.GetName(),
			},
		},
	}, nil
}

func getRBAC(namespace, group string) ([]client.Object, error) {
	root, err := projectRoot()
	if err != nil {
		return nil, err
	}

	role, err := internaltesting.LoadUnstructuredFromFile(filepath.Join(root, "deploy", "openshift", "role.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading role: %w", err)
	}

	role.SetNamespace(namespace)

	return []client.Object{
		role,
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      role.GetName(),
				Namespace: namespace,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind: "Group",
					Name: group,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "Role",
				Name: role.GetName(),
			},
		},
	}, nil
}

func projectRoot() (string, error) {
	var buf bytes.Buffer

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Stdout = &buf
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("determining top level directory from git: %w", errSetup)
	}

	return strings.TrimSpace(buf.String()), nil
}

var errSetup = errors.New("test setup failed")

func remove(path string) error {
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		return nil
	}

	return os.Remove(path)
}
