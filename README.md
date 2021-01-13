# kube-summary-exporter

[![Build Status](https://drone.prod.merit.uw.systems/api/badges/utilitywarehouse/kube-summary-exporter/status.svg)](https://drone.prod.merit.uw.systems/utilitywarehouse/kube-summary-exporter)

Exports prometheus metrics for the Kubernetes Summary API.

## Usage

Visiting http://localhost:9779/node/example-node will return metrics for the
node 'example-node'.

Here's an example scrape config. This assumes that the exporter is available at `kube-summary-exporter:9779`.

```
  - job_name: "kubernetes-summary"
    kubernetes_sd_configs:
      - role: node
    relabel_configs:
      - action: labelmap
        regex: __meta_kubernetes_node_label_(.+)
      - source_labels: [__meta_kubernetes_node_name]
        regex: (.+)
        target_label: __metrics_path__
        replacement: /node/${1}
      - target_label: __address__
        replacement: kube-summary-exporter:9779
```

## Metrics

| Metric                                        | Description                                                | Labels               |
| --------------------------------------------- | ---------------------------------------------------------- | -------------------- |
| kube_summary_container_logs_inodes_free       | Number of available Inodes for logs                        | pod, namespace, name |
| kube_summary_container_logs_inodes            | Number of Inodes for logs                                  | pod, namespace, name |
| kube_summary_container_logs_inodes_used       | Number of used Inodes for logs                             | pod, namespace, name |
| kube_summary_container_logs_available_bytes   | Number of bytes that aren't consumed by the container logs | pod, namespace, name |
| kube_summary_container_logs_capacity_bytes    | Number of bytes that can be consumed by the container logs | pod, namespace, name |
| kube_summary_container_logs_used_bytes        | Number of bytes that are consumed by the container logs    | pod, namespace, name |
| kube_summary_container_rootfs_inodes_free     | Number of available Inodes                                 | pod, namespace, name |
| kube_summary_container_rootfs_inodes          | Number of Inodes                                           | pod, namespace, name |
| kube_summary_container_rootfs_inodes_used     | Number of used Inodes                                      | pod, namespace, name |
| kube_summary_container_rootfs_available_bytes | Number of bytes that aren't consumed by the container      | pod, namespace, name |
| kube_summary_container_rootfs_capacity_bytes  | Number of bytes that can be consumed by the container      | pod, namespace, name |
| kube_summary_container_rootfs_used_bytes      | Number of bytes that are consumed by the container         | pod, namespace, name |
