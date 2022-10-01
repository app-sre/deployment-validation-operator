module github.com/app-sre/deployment-validation-operator

go 1.16

require (
	cloud.google.com/go v0.74.0 // indirect
	github.com/Azure/go-autorest/autorest v0.11.18 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20191106031601-ce3c9ade29de // indirect
	github.com/mcuadros/go-defaults v1.2.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/onsi/ginkgo v1.15.0 // indirect
	github.com/onsi/gomega v1.10.5 // indirect
	github.com/operator-framework/operator-lib v0.4.1
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/prometheus/client_golang v1.12.0
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	go.uber.org/atomic v1.7.0 // indirect
	golang.org/x/mod v0.4.2 // indirect
	golang.stackrox.io/kube-linter v0.0.0-20210923173231-2a83cbe3dec2
	gopkg.in/ini.v1 v1.57.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	github.com/go-openapi/spec => github.com/go-openapi/spec v0.0.0-20180415031709-bcff419492ee
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v1.7.2
	github.com/prometheus-operator/prometheus-operator => github.com/prometheus-operator/prometheus-operator v0.46.0
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v1.9.0
	k8s.io/api => k8s.io/api v0.20.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.20.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.4
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.20.4
	k8s.io/client-go => k8s.io/client-go v0.20.4
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
