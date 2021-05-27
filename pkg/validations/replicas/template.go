package replicas

import (
	"fmt"

	"github.com/app-sre/deployment-validation-operator/pkg/stringutils"
	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	"github.com/app-sre/deployment-validation-operator/pkg/validations/replicas/internal/params"

	"golang.stackrox.io/kube-linter/pkg/check"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/diagnostic"

	"golang.stackrox.io/kube-linter/pkg/extract"
	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/objectkinds"

	"golang.stackrox.io/kube-linter/pkg/templates"
)

const (
	templateKey = "minimum-replicas"
)

func init() {
	templates.Register(check.Template{
		HumanName:   "Minimum recommended replicas not met",
		Key:         templateKey,
		Description: "Flag applications running fewer than recommended number of replicas",
		SupportedObjectKinds: config.ObjectKindsDesc{
			ObjectKinds: []string{objectkinds.DeploymentLike},
		},
		Parameters:             params.ParamDescs,
		ParseAndValidateParams: params.ParseAndValidate,
		Instantiate: params.WrapInstantiateFunc(func(p params.Params) (check.Func, error) {
			return func(_ lintcontext.LintContext, object lintcontext.Object) []diagnostic.Diagnostic {
				if !utils.IsController(object.K8sObject) {
					return nil
				}
				replicas, found := extract.Replicas(object.K8sObject)
				if !found {
					return nil
				}
				if int(replicas) >= p.MinReplicas {
					return nil
				}
				return []diagnostic.Diagnostic{
					{Message: fmt.Sprintf("object has %d %s but minimum required replicas is %d",
						replicas, stringutils.Ternary(replicas > 1, "replicas", "replica"),
						p.MinReplicas)},
				}
			}, nil
		}),
	})
}
