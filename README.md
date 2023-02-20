# kube-summary-exporter

Exports prometheus metrics for the Kubernetes Summary API.

This exists because of: https://github.com/google/cadvisor/issues/2785

## Run locally

To run exporter locally run `go run ./...`

This will run server on default port `9779`

Visiting http://localhost:9779/node/example-node will return metrics for the
node 'example-node'. App will look for `example-node` in the `current-context`
cluster set in kube config.

[Here's an example scrape config.](manifests/scrap-config.yaml)

## Metrics

| Metric                                             | Description                                                          | Labels               |
|----------------------------------------------------|----------------------------------------------------------------------|----------------------|
| kube_summary_container_logs_inodes_free            | Number of available Inodes for logs                                  | pod, namespace, name |
| kube_summary_container_logs_inodes                 | Number of Inodes for logs                                            | pod, namespace, name |
| kube_summary_container_logs_inodes_used            | Number of used Inodes for logs                                       | pod, namespace, name |
| kube_summary_container_logs_available_bytes        | Number of bytes that aren't consumed by the container logs           | pod, namespace, name |
| kube_summary_container_logs_capacity_bytes         | Number of bytes that can be consumed by the container logs           | pod, namespace, name |
| kube_summary_container_logs_used_bytes             | Number of bytes that are consumed by the container logs              | pod, namespace, name |
| kube_summary_container_rootfs_inodes_free          | Number of available Inodes                                           | pod, namespace, name |
| kube_summary_container_rootfs_inodes               | Number of Inodes                                                     | pod, namespace, name |
| kube_summary_container_rootfs_inodes_used          | Number of used Inodes                                                | pod, namespace, name |
| kube_summary_container_rootfs_available_bytes      | Number of bytes that aren't consumed by the container                | pod, namespace, name |
| kube_summary_container_rootfs_capacity_bytes       | Number of bytes that can be consumed by the container                | pod, namespace, name |
| kube_summary_container_rootfs_used_bytes           | Number of bytes that are consumed by the container                   | pod, namespace, name |
| kube_summary_pod_ephemeral_storage_available_bytes | Number of bytes of Ephemeral storage that aren't consumed by the pod | pod, namespace       |
| kube_summary_pod_ephemeral_storage_capacity_bytes  | Number of bytes of Ephemeral storage that can be consumed by the pod | pod, namespace       |
| kube_summary_pod_ephemeral_storage_used_bytes      | Number of bytes of Ephemeral storage that are consumed by the pod    | pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes_free     | Number of available Inodes for pod Ephemeral storage                 | pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes          | Number of Inodes for pod Ephemeral storage                           | pod, namespace       |
| kube_summary_pod_ephemeral_storage_inodes_used     | Number of used Inodes for pod Ephemeral storage                      | pod, namespace       |
