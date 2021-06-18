module github.com/utilitywarehouse/kube-summary-exporter

go 1.16

require (
	github.com/gorilla/mux v1.8.0
	github.com/prometheus/client_golang v1.9.0
	k8s.io/client-go v0.21.1
	k8s.io/kubelet v0.21.1
)

replace (
	k8s.io/api => k8s.io/api v0.21.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.1
	k8s.io/client-go => k8s.io/client-go v0.21.1
	k8s.io/kubelet => k8s.io/kubelet v0.21.1
)
