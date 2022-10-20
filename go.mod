module github.com/app-sre/deployment-validation-operator

go 1.16

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/mcuadros/go-defaults v1.2.0
	github.com/onsi/ginkgo/v2 v2.3.1
	github.com/onsi/gomega v1.22.0
	github.com/openshift/api v3.9.0+incompatible
	github.com/prometheus/client_golang v1.12.0
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.32.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.1
	go.uber.org/multierr v1.6.0
	golang.org/x/crypto v0.0.0-20220427172511-eb4f295cb31f // indirect
	golang.stackrox.io/kube-linter v0.0.0-20210928184316-5e1ead387f43
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.3
)
