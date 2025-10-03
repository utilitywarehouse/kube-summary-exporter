# Makefile for kube-summary-exporter Helm chart

CHART_NAME = kube-summary-exporter
RELEASE_NAME = kube-summary-exporter
NAMESPACE = monitoring

.PHONY: lint template install upgrade uninstall package dry-run

# Lint the Helm chart
lint:
	helm lint $(CHART_NAME)

# Render templates for debugging
template:
	helm template $(RELEASE_NAME) $(CHART_NAME) --debug

# Install the chart
install:
	helm install $(RELEASE_NAME) $(CHART_NAME) --namespace $(NAMESPACE) --create-namespace

# Upgrade the chart
upgrade:
	helm upgrade $(RELEASE_NAME) $(CHART_NAME) --namespace $(NAMESPACE)

# Uninstall the chart  
uninstall:
	helm uninstall $(RELEASE_NAME) --namespace $(NAMESPACE)

# Package the chart
package:
	helm package $(CHART_NAME)

# Dry run installation
dry-run:
	helm install $(RELEASE_NAME) $(CHART_NAME) --namespace $(NAMESPACE) --dry-run --debug

# Show chart values
show-values:
	helm show values $(CHART_NAME)

# Show chart info
show-chart:
	helm show chart $(CHART_NAME)

# Test the release
test:
	helm test $(RELEASE_NAME) --namespace $(NAMESPACE)

# Get status
status:
	helm status $(RELEASE_NAME) --namespace $(NAMESPACE)

# List releases
list:
	helm list --namespace $(NAMESPACE)