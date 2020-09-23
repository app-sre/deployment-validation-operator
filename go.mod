module github.com/app-sre/dv-operator

go 1.13

require (
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/google/go-cmp v0.5.2 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mcuadros/go-defaults v1.2.0
	github.com/onsi/ginkgo v1.13.0 // indirect
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/prometheus/client_golang v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/tools v0.0.0-20200812195022-5ae4c3c160a0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	honnef.co/go/tools v0.0.1-2020.1.5 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
