package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	stats "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
)

// u64 returns a pointer to n; convenient for filling the *uint64 fields of
// stats.FsStats with sentinel values.
func u64(n uint64) *uint64 { return &n }

// pair is a (label name, label value) tuple used to build canonical label keys.
type pair struct{ n, v string }

// labelsKey returns a canonical, order-independent key for a metric's label
// set. Looking series up by label name (rather than positional order) is what
// lets the test catch a WithLabelValues(...) call that swaps two labels: the
// swapped series would be stored under wrong label values and the expected
// key would simply not be found.
func labelsKey(m *dto.Metric) string {
	pairs := append([]*dto.LabelPair(nil), m.Label...)
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].GetName() < pairs[j].GetName() })
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(p.GetName())
		b.WriteByte('=')
		b.WriteString(p.GetValue())
	}
	return b.String()
}

// key builds the same canonical label key the gather map uses, from name=value
// pairs in any order.
func key(pairs ...pair) string {
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].n < pairs[j].n })
	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(p.n)
		b.WriteByte('=')
		b.WriteString(p.v)
	}
	return b.String()
}

// gatherValues collects a registry into map[metric]map[labelsKey]value. Only
// gauges are extracted since every metric this exporter emits is a gauge.
func gatherValues(t *testing.T, reg *prometheus.Registry) map[string]map[string]float64 {
	t.Helper()
	fams, err := reg.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	out := make(map[string]map[string]float64, len(fams))
	for _, fam := range fams {
		mets := make(map[string]float64, len(fam.Metric))
		for _, m := range fam.Metric {
			if m.Gauge == nil {
				continue
			}
			mets[labelsKey(m)] = m.Gauge.GetValue()
		}
		out[fam.GetName()] = mets
	}
	return out
}

// fsStats builds a stats.FsStats populated with six distinct sentinel values
// derived from base, so a label-position bug would surface as the wrong value
// at the expected key rather than a coincidental match. usedBytes == base+3.
func fsStats(base uint64) stats.FsStats {
	return stats.FsStats{
		AvailableBytes: u64(base + 1),
		CapacityBytes:  u64(base + 2),
		UsedBytes:      u64(base + 3),
		InodesFree:     u64(base + 4),
		Inodes:         u64(base + 5),
		InodesUsed:     u64(base + 6),
	}
}

// fsPtr returns a pointer to a fully-populated FsStats with sentinel values
// derived from base.
func fsPtr(base uint64) *stats.FsStats { s := fsStats(base); return &s }

// buildSummary constructs a Summary for one node. Two pods (one carrying the
// sharedUID below) each run two containers; one container has nil rootfs to
// exercise the nil-skip path. Each pod has two volumes: one with a PVC ref
// and one without. The node carries an ImageFs stats block.
func buildSummary(nodeName string, sharedUID string) *stats.Summary {
	return &stats.Summary{
		Node: stats.NodeStats{
			NodeName: nodeName,
			Runtime: &stats.RuntimeStats{
				ImageFs: fsPtr(700),
			},
		},
		Pods: []stats.PodStats{
			{
				// All FsStats fields are nil pointers. collectSummaryMetrics
				// must emit **no** series for this pod: the per-field nil
				// guards inside collectFsStats/setGauge must skip every
				// metric so a broken guard that emits gauge=0 shows up as an
				// unexpected series.
				PodRef: stats.PodReference{Name: "pod-empty", Namespace: "ns-a", UID: "uid-empty"},
				Containers: []stats.ContainerStats{
					{Name: "c1", Logs: &stats.FsStats{}, Rootfs: &stats.FsStats{}},
				},
				EphemeralStorage: &stats.FsStats{},
			},
			{
				PodRef: stats.PodReference{Name: "pod-a", Namespace: "ns-a", UID: "uid-a"},
				Containers: []stats.ContainerStats{
					{Name: "c1", Logs: fsPtr(100), Rootfs: fsPtr(110)},
					{Name: "c2", Logs: fsPtr(200), Rootfs: fsPtr(210)},
				},
				VolumeStats: []stats.VolumeStats{
					{FsStats: fsStats(300), Name: "vol-a", PVCRef: &stats.PVCReference{Name: "pvc-a", Namespace: "ns-a"}},
					{FsStats: fsStats(310), Name: "vol-b"},
				},
				EphemeralStorage: fsPtr(400),
			},
			{
				// Reuses sharedUID across two nodes to prove the exporter keys
				// series per (node, pod, uid) and not by uid alone: a collision
				// would silently overwrite one node's metrics with the other's.
				PodRef: stats.PodReference{Name: "pod-shared", Namespace: "ns-a", UID: sharedUID},
				Containers: []stats.ContainerStats{
					{Name: "c1", Logs: fsPtr(500), Rootfs: fsPtr(510)},
					// rootfs nil on purpose: verifies the nil-skip path does not
					// emit container_rootfs_* for this container.
					{Name: "c2", Logs: fsPtr(520), Rootfs: nil},
				},
				VolumeStats: []stats.VolumeStats{
					{FsStats: fsStats(600), Name: "vol-a", PVCRef: &stats.PVCReference{Name: "pvc-shared", Namespace: "ns-a"}},
				},
				EphemeralStorage: fsPtr(410),
			},
		},
	}
}

// Test_collectSummaryMetrics verifies a representative subset of series per
// metric category are emitted with the expected values and labels. Asserting
// specific (metric, label-set) -> value pairs, rather than diffing the whole
// text exposition against a golden string, means adding a collector or a
// Prometheus library formatting change no longer breaks the test, while a
// WithLabelValues label-position bug shows up as "expected series not found"
// or a value mismatch. The fixture deliberately reuses one pod uid across
// two nodes and asserts both nodes' series survive independently.
func Test_collectSummaryMetrics(t *testing.T) {
	const (
		nodeA     = "node-a"
		nodeB     = "node-b"
		sharedUID = "shared-uid"
	)

	reg := prometheus.NewRegistry()
	collectors := newCollectors()
	collectors.register(reg)

	for _, sum := range []*stats.Summary{
		buildSummary(nodeA, sharedUID),
		buildSummary(nodeB, sharedUID),
	} {
		collectSummaryMetrics(sum, collectors)
	}

	got := gatherValues(t, reg)

	// gotVal returns the value for (metric, labelsKey) and whether present.
	gotVal := func(metric string, pairs ...pair) (float64, bool) {
		mets, ok := got[metric]
		if !ok {
			return 0, false
		}
		v, ok := mets[key(pairs...)]
		return v, ok
	}

	// expUsed is the sentinel usedBytes for a fully-populated FsStats built
	// from base (usedBytes == base+3).
	expUsed := func(base float64) float64 { return base + 3 }

	// must asserts (metric, labels) carries the expected value.
	must := func(metric string, labels []pair, want float64) {
		t.Helper()
		gotv, ok := gotVal(metric, labels...)
		if !ok {
			t.Errorf("metric %s missing for labels %s", metric, key(labels...))
			return
		}
		if gotv != want {
			t.Errorf("metric %s labels %s = %v, want %v", metric, key(labels...), gotv, want)
		}
	}

	// mustAbsent asserts the metric is absent for the given labels, used to
	// verify the nil-field skip path emits nothing.
	mustAbsent := func(metric string, labels []pair) {
		t.Helper()
		if _, ok := gotVal(metric, labels...); ok {
			t.Errorf("metric %s unexpectedly present for labels %s", metric, key(labels...))
		}
	}

	// Label builders for each metric family's label set.
	contLabels := func(node, pod, ns, uid, name string) []pair {
		return []pair{
			{"node", node}, {"namespace", ns}, {"name", name},
			{"pod", pod}, {"uid", uid},
		}
	}
	ephLabels := func(node, pod, ns, uid string) []pair {
		return []pair{{"node", node}, {"namespace", ns}, {"pod", pod}, {"uid", uid}}
	}
	volLabels := func(node, pod, ns, uid, vol, pvc, pvcNs string) []pair {
		return []pair{
			{"node", node}, {"namespace", ns}, {"name", vol},
			{"pod", pod}, {"uid", uid},
			{"persistentvolumeclaim", pvc}, {"pvc_namespace", pvcNs},
		}
	}

	// --- container logfs / rootfs: per (node, pod, container, field) sentinel ---
	for _, node := range []string{nodeA, nodeB} {
		// pod-a, container c1: logs.base=100, rootfs.base=110.
		must("kube_summary_container_logs_used_bytes", contLabels(node, "pod-a", "ns-a", "uid-a", "c1"), expUsed(100))
		must("kube_summary_container_rootfs_used_bytes", contLabels(node, "pod-a", "ns-a", "uid-a", "c1"), expUsed(110))
		// pod-a, container c2: logs=200, rootfs=210.
		must("kube_summary_container_logs_used_bytes", contLabels(node, "pod-a", "ns-a", "uid-a", "c2"), expUsed(200))
		must("kube_summary_container_rootfs_used_bytes", contLabels(node, "pod-a", "ns-a", "uid-a", "c2"), expUsed(210))
		// pod-shared, container c1: logs=500, rootfs=510.
		must("kube_summary_container_logs_used_bytes", contLabels(node, "pod-shared", "ns-a", sharedUID, "c1"), expUsed(500))
		must("kube_summary_container_rootfs_used_bytes", contLabels(node, "pod-shared", "ns-a", sharedUID, "c1"), expUsed(510))
		// pod-shared, container c2: logs=520 present, rootfs **absent** (nil).
		must("kube_summary_container_logs_used_bytes", contLabels(node, "pod-shared", "ns-a", sharedUID, "c2"), expUsed(520))
		mustAbsent("kube_summary_container_rootfs_used_bytes", contLabels(node, "pod-shared", "ns-a", sharedUID, "c2"))
	}

	// --- pod ephemeral storage ---
	for _, node := range []string{nodeA, nodeB} {
		must("kube_summary_pod_ephemeral_storage_used_bytes", ephLabels(node, "pod-a", "ns-a", "uid-a"), expUsed(400))
		must("kube_summary_pod_ephemeral_storage_used_bytes", ephLabels(node, "pod-shared", "ns-a", sharedUID), expUsed(410))
	}

	// --- pod volume storage: one with PVC, one without ---
	for _, node := range []string{nodeA, nodeB} {
		// pod-a, vol-a has a PVC.
		must("kube_summary_pod_volume_storage_used_bytes", volLabels(node, "pod-a", "ns-a", "uid-a", "vol-a", "pvc-a", "ns-a"), expUsed(300))
		// pod-a, vol-b has no PVC: empty pvc label values.
		must("kube_summary_pod_volume_storage_used_bytes", volLabels(node, "pod-a", "ns-a", "uid-a", "vol-b", "", ""), expUsed(310))
		// pod-shared, vol-a has a different PVC.
		must("kube_summary_pod_volume_storage_used_bytes", volLabels(node, "pod-shared", "ns-a", sharedUID, "vol-a", "pvc-shared", "ns-a"), expUsed(600))
	}

	// --- node runtime imagefs ---
	for _, node := range []string{nodeA, nodeB} {
		must("kube_summary_node_runtime_imagefs_used_bytes", []pair{{"node", node}}, expUsed(700))
	}

	// --- shared uid must not collide across nodes ---
	// pod-shared uses the same uid on both nodes; both series must survive
	// with distinct node labels. The per-node asserts above already enforce
	// this, but spell it out so the intent is explicit against future
	// refactors that might conflate series by uid alone.
	for _, node := range []string{nodeA, nodeB} {
		if _, ok := gotVal("kube_summary_container_rootfs_used_bytes", contLabels(node, "pod-shared", "ns-a", sharedUID, "c1")...); !ok {
			t.Errorf("shared-uid pod missing for node %s: exporter must key by (node,uid), not uid alone", node)
		}
	}

	// --- all-nil FsStats must emit no series ---
	// pod-empty has non-nil Logs/Rootfs/EphemeralStorage pointers but every
	// *uint64 field inside them is nil. No metric must be emitted for it; a
	// broken per-field nil guard that emits gauge=0 would show up here.
	for metric, mets := range got {
		for lk := range mets {
			if strings.Contains(lk, "pod=pod-empty") || strings.Contains(lk, "uid=uid-empty") {
				t.Errorf("metric %s unexpectedly emitted for all-nil FsStats pod: %s", metric, lk)
			}
		}
	}
}

// Test_collectSummaryMetrics_Concurrent exercises the shared-Collectors write
// path concurrently to confirm GaugeVecs are safe under -race. The production
// allNodesHandler fans out scrapes in goroutines but collects sequentially;
// this test nonetheless hammers the collectors from many goroutines to guard
// against a future refactor that collects concurrently.
func Test_collectSummaryMetrics_Concurrent(t *testing.T) {
	const n = 32
	reg := prometheus.NewRegistry()
	collectors := newCollectors()
	collectors.register(reg)

	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			uid := "uid-" + strconv.Itoa(i%4) // reuse some uids to force contention
			summary := buildSummary(fmt.Sprintf("node-%d", i), uid)
			collectSummaryMetrics(summary, collectors)
		}(i)
	}
	wg.Wait()

	// Gather must succeed without erroring; the exact values are irrelevant
	// here (the -race detector failing the build is the point).
	if _, err := reg.Gather(); err != nil {
		t.Fatalf("Gather after concurrent collect: %v", err)
	}
}
