package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	// Support auth providers in kubeconfig files
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var metricsNamespace = "kube_summary"

type Collectors struct {
	containerLogsInodesFree           *prometheus.GaugeVec
	containerLogsInodes               *prometheus.GaugeVec
	containerLogsInodesUsed           *prometheus.GaugeVec
	containerLogsAvailableBytes       *prometheus.GaugeVec
	containerLogsCapacityBytes        *prometheus.GaugeVec
	containerLogsUsedBytes            *prometheus.GaugeVec
	containerRootFsInodesFree         *prometheus.GaugeVec
	containerRootFsInodes             *prometheus.GaugeVec
	containerRootFsInodesUsed         *prometheus.GaugeVec
	containerRootFsAvailableBytes     *prometheus.GaugeVec
	containerRootFsCapacityBytes      *prometheus.GaugeVec
	containerRootFsUsedBytes          *prometheus.GaugeVec
	podEphemeralStorageAvailableBytes *prometheus.GaugeVec
	podEphemeralStorageCapacityBytes  *prometheus.GaugeVec
	podEphemeralStorageUsedBytes      *prometheus.GaugeVec
	podEphemeralStorageInodesFree     *prometheus.GaugeVec
	podEphemeralStorageInodes         *prometheus.GaugeVec
	podEphemeralStorageInodesUsed     *prometheus.GaugeVec
	nodeRuntimeImageFSAvailableBytes  *prometheus.GaugeVec
	nodeRuntimeImageFSCapacityBytes   *prometheus.GaugeVec
	nodeRuntimeImageFSUsedBytes       *prometheus.GaugeVec
	nodeRuntimeImageFSInodesFree      *prometheus.GaugeVec
	nodeRuntimeImageFSInodes          *prometheus.GaugeVec
	nodeRuntimeImageFSInodesUsed      *prometheus.GaugeVec
}

func newCollectors() *Collectors {
	return &Collectors{
		containerLogsInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes_free",
			Help:      "Number of available Inodes for logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerLogsInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes",
			Help:      "Number of Inodes for logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerLogsInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes_used",
			Help:      "Number of used Inodes for logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerLogsAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerLogsCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerLogsUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_used_bytes",
			Help:      "Number of bytes that are consumed by the container logs",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_free",
			Help:      "Number of available Inodes",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes",
			Help:      "Number of Inodes",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_used",
			Help:      "Number of used Inodes",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container",
		}, []string{"node", "pod", "namespace", "name"}),
		containerRootFsUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_used_bytes",
			Help:      "Number of bytes that are consumed by the container",
		}, []string{"node", "pod", "namespace", "name"}),
		podEphemeralStorageAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_available_bytes",
			Help:      "Number of bytes of Ephemeral storage that aren't consumed by the pod",
		}, []string{"node", "pod", "namespace"}),
		podEphemeralStorageCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_capacity_bytes",
			Help:      "Number of bytes of Ephemeral storage that can be consumed by the pod",
		}, []string{"node", "pod", "namespace"}),
		podEphemeralStorageUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_used_bytes",
			Help:      "Number of bytes of Ephemeral storage that are consumed by the pod",
		}, []string{"node", "pod", "namespace"}),
		podEphemeralStorageInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_free",
			Help:      "Number of available Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "namespace"}),
		podEphemeralStorageInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes",
			Help:      "Number of Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "namespace"}),
		podEphemeralStorageInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_used",
			Help:      "Number of used Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "namespace"}),
		nodeRuntimeImageFSAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_available_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that aren't consumed",
		}, []string{"node"}),
		nodeRuntimeImageFSCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_capacity_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that can be consumed",
		}, []string{"node"}),
		nodeRuntimeImageFSUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_used_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that are consumed",
		}, []string{"node"}),
		nodeRuntimeImageFSInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes_free",
			Help:      "Number of available Inodes for node Runtime ImageFS",
		}, []string{"node"}),
		nodeRuntimeImageFSInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes",
			Help:      "Number of Inodes for node Runtime ImageFS",
		}, []string{"node"}),
		nodeRuntimeImageFSInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes_used",
			Help:      "Number of used Inodes for node Runtime ImageFS",
		}, []string{"node"}),
	}
}

func (c *Collectors) register(registry *prometheus.Registry) {
	registry.MustRegister(
		c.containerLogsInodesFree,
		c.containerLogsInodes,
		c.containerLogsInodesUsed,
		c.containerLogsAvailableBytes,
		c.containerLogsCapacityBytes,
		c.containerLogsUsedBytes,
		c.containerRootFsInodesFree,
		c.containerRootFsInodes,
		c.containerRootFsInodesUsed,
		c.containerRootFsAvailableBytes,
		c.containerRootFsCapacityBytes,
		c.containerRootFsUsedBytes,
		c.podEphemeralStorageAvailableBytes,
		c.podEphemeralStorageCapacityBytes,
		c.podEphemeralStorageUsedBytes,
		c.podEphemeralStorageInodesFree,
		c.podEphemeralStorageInodes,
		c.podEphemeralStorageInodesUsed,
		c.nodeRuntimeImageFSAvailableBytes,
		c.nodeRuntimeImageFSCapacityBytes,
		c.nodeRuntimeImageFSUsedBytes,
		c.nodeRuntimeImageFSInodesFree,
		c.nodeRuntimeImageFSInodes,
		c.nodeRuntimeImageFSInodesUsed,
	)
}

// collectSummaryMetrics collects metrics from a /stats/summary response
func collectSummaryMetrics(summary *stats.Summary, collectors *Collectors) {
	nodeName := summary.Node.NodeName

	for _, pod := range summary.Pods {
		for _, container := range pod.Containers {
			if logs := container.Logs; logs != nil {
				if inodesFree := logs.InodesFree; inodesFree != nil {
					collectors.containerLogsInodesFree.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesFree))
				}
				if inodes := logs.Inodes; inodes != nil {
					collectors.containerLogsInodes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodes))
				}
				if inodesUsed := logs.InodesUsed; inodesUsed != nil {
					collectors.containerLogsInodesUsed.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesUsed))
				}
				if availableBytes := logs.AvailableBytes; availableBytes != nil {
					collectors.containerLogsAvailableBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*availableBytes))
				}
				if capacityBytes := logs.CapacityBytes; capacityBytes != nil {
					collectors.containerLogsCapacityBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*capacityBytes))
				}
				if usedBytes := logs.UsedBytes; usedBytes != nil {
					collectors.containerLogsUsedBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*usedBytes))
				}
			}
			if rootfs := container.Rootfs; rootfs != nil {
				if inodesFree := rootfs.InodesFree; inodesFree != nil {
					collectors.containerRootFsInodesFree.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesFree))
				}
				if inodes := rootfs.Inodes; inodes != nil {
					collectors.containerRootFsInodes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodes))
				}
				if inodesUsed := rootfs.InodesUsed; inodesUsed != nil {
					collectors.containerRootFsInodesUsed.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesUsed))
				}
				if availableBytes := rootfs.AvailableBytes; availableBytes != nil {
					collectors.containerRootFsAvailableBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*availableBytes))
				}
				if capacityBytes := rootfs.CapacityBytes; capacityBytes != nil {
					collectors.containerRootFsCapacityBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*capacityBytes))
				}
				if usedBytes := rootfs.UsedBytes; usedBytes != nil {
					collectors.containerRootFsUsedBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*usedBytes))
				}
			}
		}

		if ephemeralStorage := pod.EphemeralStorage; ephemeralStorage != nil {
			if ephemeralStorage.AvailableBytes != nil {
				collectors.podEphemeralStorageAvailableBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.AvailableBytes))
			}
			if ephemeralStorage.CapacityBytes != nil {
				collectors.podEphemeralStorageCapacityBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.CapacityBytes))
			}
			if ephemeralStorage.UsedBytes != nil {
				collectors.podEphemeralStorageUsedBytes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.UsedBytes))
			}
			if ephemeralStorage.InodesFree != nil {
				collectors.podEphemeralStorageInodesFree.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.InodesFree))
			}
			if ephemeralStorage.Inodes != nil {
				collectors.podEphemeralStorageInodes.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.Inodes))
			}
			if ephemeralStorage.InodesUsed != nil {
				collectors.podEphemeralStorageInodesUsed.WithLabelValues(nodeName, pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.InodesUsed))
			}
		}
	}

	if runtime := summary.Node.Runtime; runtime != nil {
		if runtime.ImageFs.AvailableBytes != nil {
			collectors.nodeRuntimeImageFSAvailableBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.AvailableBytes))
		}
		if runtime.ImageFs.CapacityBytes != nil {
			collectors.nodeRuntimeImageFSCapacityBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.CapacityBytes))
		}
		if runtime.ImageFs.UsedBytes != nil {
			collectors.nodeRuntimeImageFSUsedBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.UsedBytes))
		}
		if runtime.ImageFs.InodesFree != nil {
			collectors.nodeRuntimeImageFSInodesFree.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.InodesFree))
		}
		if runtime.ImageFs.Inodes != nil {
			collectors.nodeRuntimeImageFSInodes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.Inodes))
		}
		if runtime.ImageFs.InodesUsed != nil {
			collectors.nodeRuntimeImageFSInodesUsed.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.InodesUsed))
		}
	}
}

// nodeHandler returns metrics for the /stats/summary API of the given node
func nodeHandler(w http.ResponseWriter, r *http.Request, kubeClient *kubernetes.Clientset, collectors *Collectors) {
	node := mux.Vars(r)["node"]

	ctx, cancel := getTimeoutContext(r)
	defer cancel()

	summary, err := getNodeSummary(ctx, kubeClient, node)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying /stats/summary for %s: %v", node, err), http.StatusInternalServerError)
		return
	}

	registry := prometheus.NewRegistry()
	collectors.register(registry)
	collectSummaryMetrics(summary, collectors)

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

// allNodesHandler returns metrics for all nodes in the cluster
func allNodesHandler(w http.ResponseWriter, r *http.Request, kubeClient *kubernetes.Clientset, collectors *Collectors) {
	ctx, cancel := getTimeoutContext(r)
	defer cancel()

	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing nodes: %v", err), http.StatusInternalServerError)
		return
	}

	registry := prometheus.NewRegistry()
	collectors.register(registry)

	for _, node := range nodes.Items {
		summary, err := getNodeSummary(ctx, kubeClient, node.Name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error querying /stats/summary for %s: %v", node.Name, err), http.StatusInternalServerError)
			return
		}
		collectSummaryMetrics(summary, collectors)
	}

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

// getNodeSummary retrieves the summary for a single node
func getNodeSummary(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string) (*stats.Summary, error) {
	req := kubeClient.CoreV1().RESTClient().Get().Resource("nodes").Name(nodeName).SubResource("proxy").Suffix("stats/summary")
	resp, err := req.DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying /stats/summary for %s: %v", nodeName, err)
	}

	summary := &stats.Summary{}
	if err := json.Unmarshal(resp, summary); err != nil {
		return nil, fmt.Errorf("error unmarshaling /stats/summary response for %s: %v", nodeName, err)
	}

	return summary, nil
}

// getTimeoutContext returns a context with timeout based on the X-Prometheus-Scrape-Timeout-Seconds header
func getTimeoutContext(r *http.Request) (context.Context, context.CancelFunc) {
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		timeoutSeconds, err := strconv.ParseFloat(v, 64)
		if err == nil {
			return context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		}
	}
	return context.WithCancel(r.Context())
}

// newKubeClient returns a Kubernetes client (clientset) with configurable
// rate limits from a supplied kubeconfig path, the KUBECONFIG environment variable,
// the default config file location ($HOME/.kube/config), or from the in-cluster
// service account environment.
func newKubeClient(path string) (*kubernetes.Clientset, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if path != "" {
		loadingRules.ExplicitPath = path
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Set rate limits to reduce client-side throttling
	config.QPS = 100
	config.Burst = 200

	return kubernetes.NewForConfig(config)
}

var (
	flagListenAddress  = flag.String("listen-address", ":9779", "Listen address")
	flagKubeConfigPath = flag.String("kubeconfig", "", "Path of a kubeconfig file, if not provided the app will try $KUBECONFIG, $HOME/.kube/config or in cluster config")
)

func main() {
	flag.Parse()

	kubeClient, err := newKubeClient(*flagKubeConfigPath)
	if err != nil {
		fmt.Printf("[Error] Cannot create kube client: %v", err)
		os.Exit(1)
	}

	collectors := newCollectors()

	r := mux.NewRouter()
	r.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		allNodesHandler(w, r, kubeClient, collectors)
	})
	r.HandleFunc("/node/{node}", func(w http.ResponseWriter, r *http.Request) {
		nodeHandler(w, r, kubeClient, collectors)
	})
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
    <head><title>Kube Summary Exporter</title></head>
    <body>
        <h1>Kube Summary Exporter</h1>
        <p><a href="/nodes">Retrieve metrics for all nodes</a></p>
        <p><a href="/node/example-node">Retrieve metrics for 'example-node'</a></p>
        <p><a href="/metrics">Metrics</a></p>
    </body>
</html>`))
	})

	fmt.Printf("Listening on %s\n", *flagListenAddress)
	fmt.Printf("error: %v\n", http.ListenAndServe(*flagListenAddress, r))
}
