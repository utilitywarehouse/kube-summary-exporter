package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"

	// Support auth providers in kubeconfig files
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var metricsNamespace = "kube_summary"

// collectSummaryMetrics collects metrics from a /stats/summary response
func collectSummaryMetrics(summary *stats.Summary, registry *prometheus.Registry) {
	var (
		containerLogsInodesFree = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes_free",
			Help:      "Number of available Inodes for logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerLogsInodes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes",
			Help:      "Number of Inodes for logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerLogsInodesUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes_used",
			Help:      "Number of used Inodes for logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerLogsAvailableBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerLogsCapacityBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerLogsUsedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_used_bytes",
			Help:      "Number of bytes that are consumed by the container logs",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsInodesFree = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_free",
			Help:      "Number of available Inodes",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsInodes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes",
			Help:      "Number of Inodes",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsInodesUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_used",
			Help:      "Number of used Inodes",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsAvailableBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsCapacityBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		containerRootFsUsedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_used_bytes",
			Help:      "Number of bytes that are consumed by the container",
		},
			[]string{
				"pod",
				"namespace",
				"name",
			},
		)
		podEphemeralStorageAvailableBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_available_bytes",
			Help:      "Number of bytes of Ephemeral storage that aren't consumed by the pod",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		podEphemeralStorageCapacityBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_capacity_bytes",
			Help:      "Number of bytes of Ephemeral storage that can be consumed by the pod",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		podEphemeralStorageUsedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_used_bytes",
			Help:      "Number of bytes of Ephemeral storage that are consumed by the pod",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		podEphemeralStorageInodesFree = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_free",
			Help:      "Number of available Inodes for pod Ephemeral storage",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		podEphemeralStorageInodes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes",
			Help:      "Number of Inodes for pod Ephemeral storage",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		podEphemeralStorageInodesUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_used",
			Help:      "Number of used Inodes for pod Ephemeral storage",
		},
			[]string{
				"pod",
				"namespace",
			},
		)
		nodeRuntimeImageFSAvailableBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_available_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that aren't consumed",
		},
			[]string{
				"node",
			},
		)
		nodeRuntimeImageFSCapacityBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_capacity_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that can be consumed",
		},
			[]string{
				"node",
			},
		)
		nodeRuntimeImageFSUsedBytes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_used_bytes",
			Help:      "Number of bytes of node Runtime ImageFS that are consumed",
		},
			[]string{
				"node",
			},
		)
		nodeRuntimeImageFSInodesFree = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes_free",
			Help:      "Number of available Inodes for node Runtime ImageFS",
		},
			[]string{
				"node",
			},
		)
		nodeRuntimeImageFSInodes = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes",
			Help:      "Number of Inodes for node Runtime ImageFS",
		},
			[]string{
				"node",
			},
		)
		nodeRuntimeImageFSInodesUsed = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "node_runtime_imagefs_inodes_used",
			Help:      "Number of used Inodes for node Runtime ImageFS",
		},
			[]string{
				"node",
			},
		)
	)
	registry.MustRegister(
		containerLogsInodesFree,
		containerLogsInodes,
		containerLogsInodesUsed,
		containerLogsAvailableBytes,
		containerLogsCapacityBytes,
		containerLogsUsedBytes,
		containerRootFsInodesFree,
		containerRootFsInodes,
		containerRootFsInodesUsed,
		containerRootFsAvailableBytes,
		containerRootFsCapacityBytes,
		containerRootFsUsedBytes,
		podEphemeralStorageAvailableBytes,
		podEphemeralStorageCapacityBytes,
		podEphemeralStorageUsedBytes,
		podEphemeralStorageInodesFree,
		podEphemeralStorageInodes,
		podEphemeralStorageInodesUsed,
		nodeRuntimeImageFSAvailableBytes,
		nodeRuntimeImageFSCapacityBytes,
		nodeRuntimeImageFSUsedBytes,
		nodeRuntimeImageFSInodesFree,
		nodeRuntimeImageFSInodes,
		nodeRuntimeImageFSInodesUsed,
	)

	for _, pod := range summary.Pods {
		for _, container := range pod.Containers {
			if logs := container.Logs; logs != nil {
				if inodesFree := logs.InodesFree; inodesFree != nil {
					containerLogsInodesFree.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesFree))
				}
				if inodes := logs.Inodes; inodes != nil {
					containerLogsInodes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodes))
				}
				if inodesUsed := logs.InodesUsed; inodesUsed != nil {
					containerLogsInodesUsed.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesUsed))
				}
				if availableBytes := logs.AvailableBytes; availableBytes != nil {
					containerLogsAvailableBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*availableBytes))
				}
				if capacityBytes := logs.CapacityBytes; capacityBytes != nil {
					containerLogsCapacityBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*capacityBytes))
				}
				if usedBytes := logs.UsedBytes; usedBytes != nil {
					containerLogsUsedBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*usedBytes))
				}
			}
			if rootfs := container.Rootfs; rootfs != nil {
				if inodesFree := rootfs.InodesFree; inodesFree != nil {
					containerRootFsInodesFree.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesFree))
				}
				if inodes := rootfs.Inodes; inodes != nil {
					containerRootFsInodes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodes))
				}
				if inodesUsed := rootfs.InodesUsed; inodesUsed != nil {
					containerRootFsInodesUsed.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*inodesUsed))
				}
				if availableBytes := rootfs.AvailableBytes; availableBytes != nil {
					containerRootFsAvailableBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*availableBytes))
				}
				if capacityBytes := rootfs.CapacityBytes; capacityBytes != nil {
					containerRootFsCapacityBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*capacityBytes))
				}
				if usedBytes := rootfs.UsedBytes; usedBytes != nil {
					containerRootFsUsedBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace, container.Name).Set(float64(*usedBytes))
				}
			}
		}

		if ephemeralStorage := pod.EphemeralStorage; ephemeralStorage != nil {
			if ephemeralStorage.AvailableBytes != nil {
				podEphemeralStorageAvailableBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.AvailableBytes))
			}
			if ephemeralStorage.CapacityBytes != nil {
				podEphemeralStorageCapacityBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.CapacityBytes))
			}
			if ephemeralStorage.UsedBytes != nil {
				podEphemeralStorageUsedBytes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.UsedBytes))
			}
			if ephemeralStorage.InodesFree != nil {
				podEphemeralStorageInodesFree.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.InodesFree))
			}
			if ephemeralStorage.Inodes != nil {
				podEphemeralStorageInodes.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.Inodes))
			}
			if ephemeralStorage.InodesUsed != nil {
				podEphemeralStorageInodesUsed.WithLabelValues(pod.PodRef.Name, pod.PodRef.Namespace).Set(float64(*ephemeralStorage.InodesUsed))
			}
		}
	}

	if runtime := summary.Node.Runtime; runtime != nil {
		nodeName := summary.Node.NodeName
		if runtime.ImageFs.AvailableBytes != nil {
			nodeRuntimeImageFSAvailableBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.AvailableBytes))
		}
		if runtime.ImageFs.CapacityBytes != nil {
			nodeRuntimeImageFSCapacityBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.CapacityBytes))
		}
		if runtime.ImageFs.UsedBytes != nil {
			nodeRuntimeImageFSUsedBytes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.UsedBytes))
		}
		if runtime.ImageFs.InodesFree != nil {
			nodeRuntimeImageFSInodesFree.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.InodesFree))
		}
		if runtime.ImageFs.Inodes != nil {
			nodeRuntimeImageFSInodes.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.Inodes))
		}
		if runtime.ImageFs.InodesUsed != nil {
			nodeRuntimeImageFSInodesUsed.WithLabelValues(nodeName).Set(float64(*runtime.ImageFs.InodesUsed))
		}
	}

}

// nodeHandler returns metrics for the /stats/summary API of the given node
func nodeHandler(w http.ResponseWriter, r *http.Request, kubeClient *kubernetes.Clientset) {
	node := mux.Vars(r)["node"]

	// If a timeout is configured via the Prometheus header, add it to the request.
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		var err error
		timeoutSeconds, err := strconv.ParseFloat(v, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error parsing timeout from X-Prometheus-Scrape-Timeout-Seconds=%s: %v", html.EscapeString(v), err), http.StatusInternalServerError)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		defer cancel()
		r = r.WithContext(ctx)
	}

	req := kubeClient.CoreV1().RESTClient().Get().Resource("nodes").Name(node).SubResource("proxy").Suffix("stats/summary")
	resp, err := req.DoRaw(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying /stats/summary for %s: %v", html.EscapeString(node), err), http.StatusInternalServerError)
		return
	}

	summary := &stats.Summary{}
	if err := json.Unmarshal(resp, summary); err != nil {
		http.Error(w, fmt.Sprintf("Error unmarshaling /stats/summary response for %s: %v", html.EscapeString(node), err), http.StatusInternalServerError)
		return
	}

	registry := prometheus.NewRegistry()

	collectSummaryMetrics(summary, registry)

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

// newKubeClient returns a Kubernetes client (clientset) from the supplied
// kubeconfig path, the KUBECONFIG environment variable, the default config file
// location ($HOME/.kube/config) or from the in-cluster service account environment.
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

	r := mux.NewRouter()
	r.HandleFunc("/node/{node}", func(w http.ResponseWriter, r *http.Request) {
		nodeHandler(w, r, kubeClient)
	})
	r.Handle("/metrics", promhttp.Handler())
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
    <head><title>Kube Summary Exporter</title></head>
    <body>
        <h1>Kube Summary Exporter</h1>
        <p><a href="node/example-node">Retrieve metrics for 'example-node'</a></p>
        <p><a href="metrics">Metrics</a></p>
    </body>
</html>`))
	})

	fmt.Printf("Listening on %s\n", *flagListenAddress)
	fmt.Printf("error: %v\n", http.ListenAndServe(*flagListenAddress, r))
}
