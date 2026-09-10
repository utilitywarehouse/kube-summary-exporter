# apparmor kustomize component

Adds `appArmorProfile: {type: RuntimeDefault}` to the kube-summary-exporter
Deployment container.

## When to use

The Kubernetes Pod Security Standards `restricted` policy requires the
`appArmorProfile` field to be `RuntimeDefault` or `Localhost` on Kubernetes
v1.30 or newer. The base deployment leaves it unset because setting it
unconditionally prevents a pod from starting on nodes where AppArmor is not
enabled in the kernel (the kubelet fails the pod with "Cannot enforce
AppArmor"). Enable this component only when both apply:

- your cluster enforces `restricted` natively (Pod Security admission), and
- your nodes have AppArmor enabled in the kernel.

## Usage

Add the component reference to your kustomization:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - <path-to>/manifests/base
components:
  - <path-to>/manifests/components/apparmor
```

Note that the security context on the base Deployment already passes every
other `restricted` check, so clusters enforcing `restricted` only through an
admission controller that does not require AppArmor (such as Kyverno) should
not use this component.
