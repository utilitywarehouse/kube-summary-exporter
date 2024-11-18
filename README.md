# kube-summary-exporter

Exports Prometheus metrics for the Kubernetes Summary API.

This exists because of: https://github.com/google/cadvisor/issues/2785

Docker / Podman image available:
`quay.io/utilitywarehouse/kube-summary-exporter`

All available tags:
https://quay.io/repository/utilitywarehouse/kube-summary-exporter?tab=tags

## Run locally

To run exporter locally run `go run ./...`

This will run server on default port `9779`

Visiting http://localhost:9779/node/{node-name} will return metrics for the
specified node. The app will look for the node in the `current-context`
cluster set in kube config.

You can also visit http://localhost:9779/nodes to retrieve metrics for all nodes in the cluster.

[Here's an example scrape config.](manifests/scrape-config.yaml)

## Endpoints

- `/`: Home page with links to other endpoints
- `/nodes`: Metrics for all nodes in the cluster
- `/node/{node}`: Metrics for a specific node
- `/metrics`: Prometheus metrics about the exporter itself

## Command-line Flags

- `--listen-address`: The address to listen on for HTTP requests (default ":9779")
- `--kubeconfig`: Path to a kubeconfig file (if not provided, the app will try $KUBECONFIG, $HOME/.kube/config, or in-cluster config)

## Metrics

| Metric                                             | Description                                                          | Labels                     |
| -------------------------------------------------- | -------------------------------------------------------------------- | -------------------------- |
| kube_summary_container_logs_available_bytes        | Number of bytes that aren't consumed by the container logs           | node, pod, namespace, name |
| kube_summary_container_logs_capacity_bytes         | Number of bytes that can be consumed by the container logs           | node, pod, namespace, name |
| kube_summary_container_logs_inodes                 | Number of Inodes for logs                                            | node, pod, namespace, name |
| kube_summary_container_logs_inodes_free            | Number of available Inodes for logs                                  | node, pod, namespace, name |
| kube_summary_container_logs_inodes_used            | Number of used Inodes for logs                                       | node, pod, namespace, name |
| kube_summary_container_logs_used_bytes             | Number of bytes that are consumed by the container logs              | node, pod, namespace, name |
| kube_summary_container_rootfs_available_bytes      | Number of bytes that aren't consumed by the container                | node, pod, namespace, name |
| kube_summary_container_rootfs_capacity_bytes       | Number of bytes that can be consumed by the container                | node, pod, namespace, name |
| kube_summary_container_rootfs_inodes               | Number of Inodes                                                     | node, pod, namespace, name |
| kube_summary_container_rootfs_inodes_free          | Number of available Inodes                                           | node, pod, namespace, name |
| kube_summary_container_rootfs_inodes_used          | Number of used Inodes                                                | node, pod, namespace, name |
| kube_summary_container_rootfs_used_bytes           | Number of bytes that are consumed by the container                   | node, pod, namespace, name |
| kube_summary_node_runtime_imagefs_available_bytes  | Number of bytes of node Runtime ImageFS that aren't consumed         | node                       |
| kube_summary_node_runtime_imagefs_capacity_bytes   | Number of bytes of node Runtime ImageFS that can be consumed         | node                       |
| kube_summary_node_runtime_imagefs_inodes           | Number of Inodes for node Runtime ImageFS                            | node                       |
| kube_summary_node_runtime_imagefs_inodes_free      | Number of available Inodes for node Runtime ImageFS                  | node                       |
| kube_summary_node_runtime_imagefs_inodes_used      | Number of used Inodes for node Runtime ImageFS                       | node                       |
| kube_summary_node_runtime_imagefs_used_bytes       | Number of bytes of node Runtime ImageFS that are consumed            | node                       |
| kube_summary_pod_ephemeral_storage_available_bytes | Number of bytes of Ephemeral storage that aren't consumed by the pod | node, pod, namespace       |
| kube_summary_pod_ephemeral_storage_capacity_bytes  | Number of bytes of Ephemeral storage that can be consumed by the pod | node, pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes          | Number of Inodes for pod Ephemeral storage                           | node, pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes_free     | Number of available Inodes for pod Ephemeral storage                 | node, pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes_used     | Number of used Inodes for pod Ephemeral storage                      | node, pod, namespace       |
| kube_summary_pod_ephemeral_storage_used_bytes      | Number of bytes of Ephemeral storage that are consumed by the pod    | node, pod, namespace       |

## Development

### Running Tests

To run the tests, use the following command:
```
go test ./...
```

The main test file (`main_test.go`) includes a test for the `collectSummaryMetrics` function, which verifies that the metrics are collected correctly from a sample JSON file (`test-summary.json`).
