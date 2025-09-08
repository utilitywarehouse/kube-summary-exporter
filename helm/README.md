# Kube Summary Exporter Helm Chart

This Helm chart deploys kube-summary-exporter on a Kubernetes cluster using the Helm package manager.

## Prerequisites

- Kubernetes 1.16+
- Helm 3.0+
- RBAC enabled cluster (for ServiceAccount and ClusterRole)

## Installation

From this directory:

```bash
# Install the chart
helm install kube-summary-exporter ./kube-summary-exporter --namespace monitoring --create-namespace

# Or use the Makefile
make install
```

## Configuration

The following table lists the configurable parameters of the kube-summary-exporter chart and their default values.

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Container image repository | `quay.io/utilitywarehouse/kube-summary-exporter` |
| `image.pullPolicy` | Container image pull policy | `IfNotPresent` |
| `image.tag` | Container image tag | `"latest"` |
| `service.type` | Kubernetes service type | `ClusterIP` |
| `service.port` | Service port | `9779` |
| `config.listenAddress` | Address to listen on for HTTP requests | `":9779"` |
| `config.kubeconfig` | Path to kubeconfig file (empty for in-cluster) | `""` |
| `rbac.create` | Create RBAC resources | `true` |
| `serviceAccount.create` | Create service account | `true` |
| `serviceMonitor.enabled` | Create ServiceMonitor for Prometheus Operator | `true` |
| `serviceMonitorNodes.enabled` | Create ServiceMonitor for /nodes endpoint | `true` |

### ServiceMonitor Configuration

The chart includes two ServiceMonitors for Prometheus Operator:

1. **serviceMonitor**: Scrapes `/metrics` endpoint (exporter metrics)
2. **serviceMonitorNodes**: Scrapes `/nodes` endpoint (Kubernetes summary metrics)

Both ServiceMonitors are configured with the label `release: kube-prom-stack` by default. Update `serviceMonitor.additionalLabels.release` to match your Prometheus Operator release name.

## Usage Examples

### Custom Values Installation

Create a `values.yaml` file:

```yaml
image:
  tag: "v1.0.0"

resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 50m
    memory: 64Mi

serviceMonitor:
  additionalLabels:
    release: my-prometheus

nodeSelector:
  kubernetes.io/os: linux
```

Install with custom values:

```bash
helm install kube-summary-exporter ./kube-summary-exporter -f values.yaml --namespace monitoring --create-namespace
```

### Disable ServiceMonitors

If you don't use Prometheus Operator:

```bash
helm install kube-summary-exporter ./kube-summary-exporter \
  --set serviceMonitor.enabled=false \
  --set serviceMonitorNodes.enabled=false \
  --namespace monitoring --create-namespace
```

## Accessing the Application

After installation, you can access the kube-summary-exporter using port-forward:

```bash
export POD_NAME=$(kubectl get pods --namespace monitoring -l "app.kubernetes.io/name=kube-summary-exporter" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace monitoring port-forward $POD_NAME 9779:9779
```

Then visit:
- http://localhost:9779/ - Home page
- http://localhost:9779/nodes - Metrics for all nodes
- http://localhost:9779/node/{node-name} - Metrics for specific node
- http://localhost:9779/metrics - Exporter metrics

## Available Commands (Makefile)

From the helm directory, you can use:

```bash
make lint          # Lint the Helm chart
make template      # Render templates for debugging
make install       # Install the chart
make upgrade       # Upgrade the chart
make uninstall     # Uninstall the chart
make package       # Package the chart
make dry-run       # Dry run installation
make show-values   # Show chart values
make show-chart    # Show chart info
make test          # Test the release
make status        # Get release status
make list          # List releases
```

## Troubleshooting

### ServiceMonitor Not Picked Up

If Prometheus isn't discovering your ServiceMonitor:

1. Check your Prometheus serviceMonitorSelector labels:
   ```bash
   kubectl get prometheus -o yaml | grep -A 10 serviceMonitorSelector
   ```

2. Ensure the `release` label matches your Prometheus release name:
   ```yaml
   serviceMonitor:
     additionalLabels:
       release: your-prometheus-release-name
   ```

3. Verify namespace permissions and RBAC for Prometheus

### Pod Not Starting

Check the logs:
```bash
kubectl logs -n monitoring deployment/kube-summary-exporter
```

Common issues:
- RBAC permissions (check ServiceAccount and ClusterRole)
- Image pull issues (check imagePullSecrets)
- Resource constraints

## Metrics Exposed

The exporter provides detailed Kubernetes node and container metrics including:

- Container logs filesystem usage
- Container rootfs usage
- Node runtime ImageFS usage
- Pod ephemeral storage usage

See the [main README](../README.md) for a complete list of available metrics.

## Development

To contribute to this chart:

1. Make changes to templates or values
2. Test with `helm template` and `helm lint`
3. Verify installation works in a test cluster
4. Update this README if adding new configuration options

## License

This chart is part of the kube-summary-exporter project. See the main repository for license information.