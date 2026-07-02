package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
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

const defaultScrapeTimeout = 60 * time.Second

var (
	flagKubeConfigPath = flag.String("kubeconfig", "", "Path of a kubeconfig file, if not provided the app will try $KUBECONFIG, $HOME/.kube/config or in cluster config")
	flagListenAddress  = flag.String("listen-address", ":9779", "Listen address")
	metricsNamespace   = "kube_summary"

	// errorLog is used for promhttp.HandlerOpts.ErrorLog so registry
	// exposition errors are observable instead of silently dropped.
	errorLog = log.New(os.Stderr, "", log.LstdFlags)

	// scrapeErrorsTotal counts per-node /stats/summary scrape failures. It is
	// registered on the default registry so it is exposed on /metrics and
	// operators can alert on partial scrapes instead of them being silently
	// dropped.
	scrapeErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metricsNamespace,
		Subsystem: "exporter",
		Name:      "scrape_errors_total",
		Help:      "Total number of errors scraping a node's /stats/summary",
	}, []string{"node"})

	// lastScrapeDurationSeconds records the duration of the most recent
	// /stats/summary scrape per node, in seconds.
	lastScrapeDurationSeconds = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metricsNamespace,
		Subsystem: "exporter",
		Name:      "last_scrape_duration_seconds",
		Help:      "Duration of the last scrape of a node's /stats/summary in seconds",
	}, []string{"node"})
)

func init() {
	prometheus.MustRegister(scrapeErrorsTotal, lastScrapeDurationSeconds)
}

// handlerOpts configures promhttp to keep emitting remaining metrics even when
// a single metric errors, and to surface exposition errors via errorLog.
var handlerOpts = promhttp.HandlerOpts{
	ErrorHandling: promhttp.ContinueOnError,
	ErrorLog:      errorLog,
}

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
	podVolumeStorageAvailableBytes    *prometheus.GaugeVec
	podVolumeStorageCapacityBytes     *prometheus.GaugeVec
	podVolumeStorageUsedBytes         *prometheus.GaugeVec
	podVolumeStorageInodesFree        *prometheus.GaugeVec
	podVolumeStorageInodes            *prometheus.GaugeVec
	podVolumeStorageInodesUsed        *prometheus.GaugeVec
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
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerLogsInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes",
			Help:      "Number of Inodes for logs",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerLogsInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_inodes_used",
			Help:      "Number of used Inodes for logs",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerLogsAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container logs",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerLogsCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container logs",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerLogsUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_logs_used_bytes",
			Help:      "Number of bytes that are consumed by the container logs",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_free",
			Help:      "Number of available Inodes",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes",
			Help:      "Number of Inodes",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_inodes_used",
			Help:      "Number of used Inodes",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_available_bytes",
			Help:      "Number of bytes that aren't consumed by the container",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_capacity_bytes",
			Help:      "Number of bytes that can be consumed by the container",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		containerRootFsUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "container_rootfs_used_bytes",
			Help:      "Number of bytes that are consumed by the container",
		}, []string{"node", "pod", "uid", "namespace", "name"}),
		podEphemeralStorageAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_available_bytes",
			Help:      "Number of bytes of Ephemeral storage that aren't consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace"}),
		podEphemeralStorageCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_capacity_bytes",
			Help:      "Number of bytes of Ephemeral storage that can be consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace"}),
		podEphemeralStorageUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_used_bytes",
			Help:      "Number of bytes of Ephemeral storage that are consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace"}),
		podEphemeralStorageInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_free",
			Help:      "Number of available Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "uid", "namespace"}),
		podEphemeralStorageInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes",
			Help:      "Number of Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "uid", "namespace"}),
		podEphemeralStorageInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_ephemeral_storage_inodes_used",
			Help:      "Number of used Inodes for pod Ephemeral storage",
		}, []string{"node", "pod", "uid", "namespace"}),
		podVolumeStorageAvailableBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_available_bytes",
			Help:      "Number of bytes of Volume storage that aren't consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
		podVolumeStorageCapacityBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_capacity_bytes",
			Help:      "Number of bytes of Volume storage that can be consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
		podVolumeStorageUsedBytes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_used_bytes",
			Help:      "Number of bytes of Volume storage that are consumed by the pod",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
		podVolumeStorageInodesFree: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_inodes_free",
			Help:      "Number of available Inodes for pod Volume storage",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
		podVolumeStorageInodes: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_inodes",
			Help:      "Number of Inodes for pod Volume storage",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
		podVolumeStorageInodesUsed: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "pod_volume_storage_inodes_used",
			Help:      "Number of used Inodes for pod Volume storage",
		}, []string{"node", "pod", "uid", "namespace", "name", "persistentvolumeclaim", "pvc_namespace"}),
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
		c.podVolumeStorageAvailableBytes,
		c.podVolumeStorageCapacityBytes,
		c.podVolumeStorageUsedBytes,
		c.podVolumeStorageInodesFree,
		c.podVolumeStorageInodes,
		c.podVolumeStorageInodesUsed,
		c.nodeRuntimeImageFSAvailableBytes,
		c.nodeRuntimeImageFSCapacityBytes,
		c.nodeRuntimeImageFSUsedBytes,
		c.nodeRuntimeImageFSInodesFree,
		c.nodeRuntimeImageFSInodes,
		c.nodeRuntimeImageFSInodesUsed,
	)
}

// fsCollectors groups the six GaugeVecs that mirror the fields of a
// stats.FsStats, in the order availableBytes, capacityBytes, usedBytes,
// inodesFree, inodes, inodesUsed.
type fsCollectors struct {
	availableBytes *prometheus.GaugeVec
	capacityBytes  *prometheus.GaugeVec
	usedBytes      *prometheus.GaugeVec
	inodesFree     *prometheus.GaugeVec
	inodes         *prometheus.GaugeVec
	inodesUsed     *prometheus.GaugeVec
}

// collectFsStats sets all six collectors from a single FsStats using the same
// label values for every series. Nil fields are skipped.
func collectFsStats(fs *stats.FsStats, c fsCollectors, labels []string) {
	setGauge(c.availableBytes, labels, fs.AvailableBytes)
	setGauge(c.capacityBytes, labels, fs.CapacityBytes)
	setGauge(c.usedBytes, labels, fs.UsedBytes)
	setGauge(c.inodesFree, labels, fs.InodesFree)
	setGauge(c.inodes, labels, fs.Inodes)
	setGauge(c.inodesUsed, labels, fs.InodesUsed)
}

// setGauge sets vec for the given labels to v if v is non-nil.
func setGauge(vec *prometheus.GaugeVec, labels []string, v *uint64) {
	if v != nil {
		vec.WithLabelValues(labels...).Set(float64(*v))
	}
}

// collectSummaryMetrics collects metrics from a /stats/summary response
func collectSummaryMetrics(summary *stats.Summary, collectors *Collectors) {
	nodeName := summary.Node.NodeName

	logsCs := fsCollectors{
		availableBytes: collectors.containerLogsAvailableBytes,
		capacityBytes:  collectors.containerLogsCapacityBytes,
		usedBytes:      collectors.containerLogsUsedBytes,
		inodesFree:     collectors.containerLogsInodesFree,
		inodes:         collectors.containerLogsInodes,
		inodesUsed:     collectors.containerLogsInodesUsed,
	}
	rootfsCs := fsCollectors{
		availableBytes: collectors.containerRootFsAvailableBytes,
		capacityBytes:  collectors.containerRootFsCapacityBytes,
		usedBytes:      collectors.containerRootFsUsedBytes,
		inodesFree:     collectors.containerRootFsInodesFree,
		inodes:         collectors.containerRootFsInodes,
		inodesUsed:     collectors.containerRootFsInodesUsed,
	}
	ephemeralCs := fsCollectors{
		availableBytes: collectors.podEphemeralStorageAvailableBytes,
		capacityBytes:  collectors.podEphemeralStorageCapacityBytes,
		usedBytes:      collectors.podEphemeralStorageUsedBytes,
		inodesFree:     collectors.podEphemeralStorageInodesFree,
		inodes:         collectors.podEphemeralStorageInodes,
		inodesUsed:     collectors.podEphemeralStorageInodesUsed,
	}
	volumeCs := fsCollectors{
		availableBytes: collectors.podVolumeStorageAvailableBytes,
		capacityBytes:  collectors.podVolumeStorageCapacityBytes,
		usedBytes:      collectors.podVolumeStorageUsedBytes,
		inodesFree:     collectors.podVolumeStorageInodesFree,
		inodes:         collectors.podVolumeStorageInodes,
		inodesUsed:     collectors.podVolumeStorageInodesUsed,
	}
	imageFsCs := fsCollectors{
		availableBytes: collectors.nodeRuntimeImageFSAvailableBytes,
		capacityBytes:  collectors.nodeRuntimeImageFSCapacityBytes,
		usedBytes:      collectors.nodeRuntimeImageFSUsedBytes,
		inodesFree:     collectors.nodeRuntimeImageFSInodesFree,
		inodes:         collectors.nodeRuntimeImageFSInodes,
		inodesUsed:     collectors.nodeRuntimeImageFSInodesUsed,
	}

	nodeLabels := []string{nodeName}

	for _, pod := range summary.Pods {
		podLabels := []string{nodeName, pod.PodRef.Name, pod.PodRef.UID, pod.PodRef.Namespace}

		for _, container := range pod.Containers {
			containerLabels := []string{nodeName, pod.PodRef.Name, pod.PodRef.UID, pod.PodRef.Namespace, container.Name}
			if container.Logs != nil {
				collectFsStats(container.Logs, logsCs, containerLabels)
			}
			if container.Rootfs != nil {
				collectFsStats(container.Rootfs, rootfsCs, containerLabels)
			}
		}

		if pod.EphemeralStorage != nil {
			collectFsStats(pod.EphemeralStorage, ephemeralCs, podLabels)
		}

		for _, volume := range pod.VolumeStats {
			pvcName, pvcNamespace := "", ""
			if volume.PVCRef != nil {
				pvcName = volume.PVCRef.Name
				pvcNamespace = volume.PVCRef.Namespace
			}
			volumeLabels := []string{nodeName, pod.PodRef.Name, pod.PodRef.UID, pod.PodRef.Namespace, volume.Name, pvcName, pvcNamespace}
			collectFsStats(&volume.FsStats, volumeCs, volumeLabels)
		}
	}

	if runtime := summary.Node.Runtime; runtime != nil && runtime.ImageFs != nil {
		collectFsStats(runtime.ImageFs, imageFsCs, nodeLabels)
	}
}

// nodeHandler returns metrics for the /stats/summary API of the given node
func nodeHandler(w http.ResponseWriter, r *http.Request, kubeClient *kubernetes.Clientset) {
	node := mux.Vars(r)["node"]

	ctx, cancel := timeoutContext(r)
	defer cancel()

	start := time.Now()
	summary, err := nodeSummary(ctx, kubeClient, node)
	lastScrapeDurationSeconds.WithLabelValues(node).Set(time.Since(start).Seconds())
	if err != nil {
		scrapeErrorsTotal.WithLabelValues(node).Inc()
		http.Error(w, fmt.Sprintf("Error querying /stats/summary for %s: %v", node, err), http.StatusInternalServerError)
		return
	}

	collectors := newCollectors()
	registry := prometheus.NewRegistry()
	collectors.register(registry)
	collectSummaryMetrics(summary, collectors)

	h := promhttp.HandlerFor(registry, handlerOpts)
	h.ServeHTTP(w, r)
}

// allNodesHandler returns metrics for all nodes in the cluster
func allNodesHandler(w http.ResponseWriter, r *http.Request, kubeClient *kubernetes.Clientset) {
	ctx, cancel := timeoutContext(r)
	defer cancel()

	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		http.Error(w, fmt.Sprintf("Error listing nodes: %v", err), http.StatusInternalServerError)
		return
	}

	collectors := newCollectors()
	registry := prometheus.NewRegistry()
	collectors.register(registry)

	type result struct {
		summary  *stats.Summary
		node     string
		err      error
		duration time.Duration
	}

	results := make(chan result, len(nodes.Items))
	var wg sync.WaitGroup

	// Process each node concurrently
	for _, node := range nodes.Items {
		wg.Add(1)
		go func(n string) {
			defer wg.Done()

			// Each nodeSummary call gets the shared context (with timeout)
			start := time.Now()
			summary, err := nodeSummary(ctx, kubeClient, n)
			results <- result{
				summary:  summary,
				node:     n,
				err:      err,
				duration: time.Since(start),
			}
		}(node.Name)
	}

	// Close channel when all node scrapes finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Consume results
	for res := range results {
		lastScrapeDurationSeconds.WithLabelValues(res.node).Set(res.duration.Seconds())
		if res.err != nil {
			// Record the failure and DO NOT fail the whole scrape
			scrapeErrorsTotal.WithLabelValues(res.node).Inc()
			errorLog.Printf("Error scraping %s: %v", res.node, res.err)
			continue
		}
		collectSummaryMetrics(res.summary, collectors)
	}

	// Return all aggregated metrics
	h := promhttp.HandlerFor(registry, handlerOpts)
	h.ServeHTTP(w, r)
}

// nodeSummary retrieves the summary for a single node
func nodeSummary(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string) (*stats.Summary, error) {
	req := kubeClient.CoreV1().RESTClient().Get().Resource("nodes").Name(nodeName).SubResource("proxy").Suffix("stats/summary")
	resp, err := req.DoRaw(ctx)
	if err != nil {
		return nil, fmt.Errorf("error querying /stats/summary for %s: %w", nodeName, err)
	}

	summary := &stats.Summary{}
	if err := json.Unmarshal(resp, summary); err != nil {
		return nil, fmt.Errorf("error unmarshaling /stats/summary response for %s: %w", nodeName, err)
	}

	return summary, nil
}

// timeoutContext returns a context with a scrape timeout. The timeout is taken
// from the X-Prometheus-Scrape-Timeout-Seconds header when present, otherwise
// defaultScrapeTimeout is applied so a hung kubelet proxy cannot block the
// scrape indefinitely.
func timeoutContext(r *http.Request) (context.Context, context.CancelFunc) {
	if v := r.Header.Get("X-Prometheus-Scrape-Timeout-Seconds"); v != "" {
		if timeoutSeconds, err := strconv.ParseFloat(v, 64); err == nil {
			return context.WithTimeout(r.Context(), time.Duration(timeoutSeconds*float64(time.Second)))
		}
	}
	return context.WithTimeout(r.Context(), defaultScrapeTimeout)
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

func main() {
	flag.Parse()

	kubeClient, err := newKubeClient(*flagKubeConfigPath)
	if err != nil {
		errorLog.Printf("[Error] Cannot create kube client: %v", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		allNodesHandler(w, r, kubeClient)
	})
	r.HandleFunc("/node/{node}", func(w http.ResponseWriter, r *http.Request) {
		nodeHandler(w, r, kubeClient)
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
