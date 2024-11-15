package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/client_golang/prometheus"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

func Test_collectSummaryMetrics(t *testing.T) {
	expectedOut := `# HELP kube_summary_container_logs_available_bytes Number of bytes that aren't consumed by the container logs
# TYPE kube_summary_container_logs_available_bytes gauge
kube_summary_container_logs_available_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 9.0016837632e+10
# HELP kube_summary_container_logs_capacity_bytes Number of bytes that can be consumed by the container logs
# TYPE kube_summary_container_logs_capacity_bytes gauge
kube_summary_container_logs_capacity_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 1.01535985664e+11
# HELP kube_summary_container_logs_inodes Number of Inodes for logs
# TYPE kube_summary_container_logs_inodes gauge
kube_summary_container_logs_inodes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5474432e+07
# HELP kube_summary_container_logs_inodes_free Number of available Inodes for logs
# TYPE kube_summary_container_logs_inodes_free gauge
kube_summary_container_logs_inodes_free{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5355212e+07
# HELP kube_summary_container_logs_inodes_used Number of used Inodes for logs
# TYPE kube_summary_container_logs_inodes_used gauge
kube_summary_container_logs_inodes_used{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 1
# HELP kube_summary_container_logs_used_bytes Number of bytes that are consumed by the container logs
# TYPE kube_summary_container_logs_used_bytes gauge
kube_summary_container_logs_used_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 8192
# HELP kube_summary_container_rootfs_available_bytes Number of bytes that aren't consumed by the container
# TYPE kube_summary_container_rootfs_available_bytes gauge
kube_summary_container_rootfs_available_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 9.0016837632e+10
# HELP kube_summary_container_rootfs_capacity_bytes Number of bytes that can be consumed by the container
# TYPE kube_summary_container_rootfs_capacity_bytes gauge
kube_summary_container_rootfs_capacity_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 1.01535985664e+11
# HELP kube_summary_container_rootfs_inodes Number of Inodes
# TYPE kube_summary_container_rootfs_inodes gauge
kube_summary_container_rootfs_inodes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5474432e+07
# HELP kube_summary_container_rootfs_inodes_free Number of available Inodes
# TYPE kube_summary_container_rootfs_inodes_free gauge
kube_summary_container_rootfs_inodes_free{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5355212e+07
# HELP kube_summary_container_rootfs_inodes_used Number of used Inodes
# TYPE kube_summary_container_rootfs_inodes_used gauge
kube_summary_container_rootfs_inodes_used{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 14
# HELP kube_summary_container_rootfs_used_bytes Number of bytes that are consumed by the container
# TYPE kube_summary_container_rootfs_used_bytes gauge
kube_summary_container_rootfs_used_bytes{name="dev-server",namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 114688
# HELP kube_summary_pod_ephemeral_storage_available_bytes Number of bytes of Ephemeral storage that aren't consumed by the pod
# TYPE kube_summary_pod_ephemeral_storage_available_bytes gauge
kube_summary_pod_ephemeral_storage_available_bytes{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 9.0016837632e+10
# HELP kube_summary_pod_ephemeral_storage_capacity_bytes Number of bytes of Ephemeral storage that can be consumed by the pod
# TYPE kube_summary_pod_ephemeral_storage_capacity_bytes gauge
kube_summary_pod_ephemeral_storage_capacity_bytes{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 1.01535985664e+11
# HELP kube_summary_pod_ephemeral_storage_inodes Number of Inodes for pod Ephemeral storage
# TYPE kube_summary_pod_ephemeral_storage_inodes gauge
kube_summary_pod_ephemeral_storage_inodes{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5474432e+07
# HELP kube_summary_pod_ephemeral_storage_inodes_free Number of available Inodes for pod Ephemeral storage
# TYPE kube_summary_pod_ephemeral_storage_inodes_free gauge
kube_summary_pod_ephemeral_storage_inodes_free{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 2.5355212e+07
# HELP kube_summary_pod_ephemeral_storage_inodes_used Number of used Inodes for pod Ephemeral storage
# TYPE kube_summary_pod_ephemeral_storage_inodes_used gauge
kube_summary_pod_ephemeral_storage_inodes_used{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 63
# HELP kube_summary_pod_ephemeral_storage_used_bytes Number of bytes of Ephemeral storage that are consumed by the pod
# TYPE kube_summary_pod_ephemeral_storage_used_bytes gauge
kube_summary_pod_ephemeral_storage_used_bytes{namespace="mon",node="test.eu-west-1.compute.internal",pod="dev-server-0"} 1.33947392e+08
`

	d, err := os.ReadFile("test-summary.json")
	if err != nil {
		t.Fatal(err)
	}

	var summary stats.Summary
	collectors := newCollectors()

	err = json.Unmarshal(d, &summary)
	if err != nil {
		t.Fatal(err)
	}

	collectSummaryMetrics(&summary, collectors)

	tmpfile, err := os.CreateTemp("", "test-summary.prom")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	registry := prometheus.NewRegistry()
	collectors.register(registry)

	if err := prometheus.WriteToTextfile(tmpfile.Name(), registry); err != nil {
		t.Fatal(err)
	}

	fileBytes, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(string(fileBytes), expectedOut); diff != "" {
		t.Errorf("collectSummaryMetrics() metrics mismatch (-want +got):\n%s", diff)
	}
}
