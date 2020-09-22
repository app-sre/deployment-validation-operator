module github.com/app-sre/dv-operator

go 1.13

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b // indirect
	github.com/golangci/golangci-lint v1.31.0 // indirect
	github.com/mcuadros/go-defaults v1.2.0
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/prometheus/client_golang v1.6.0
	github.com/sirupsen/logrus v1.6.0 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.17.4
	k8s.io/apiextensions-apiserver v0.17.4 // indirect
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kube-aggregator v0.17.3 // indirect
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
