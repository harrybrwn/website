# Kubernetes

Sometimes the HelmCharts (managed my k3s's built in `helm-controller`) will
refuse to be deleted. This is because of the `.metadata.finalizers` list. See
[this issue](https://github.com/k3s-io/helm-controller/issues/33). Use the
following command to remove the finalizes.
```
kubectl patch -n kube-system helmcharts.helm.cattle.io prometheus  --type='json' -p='[{"op": "replace", "path": "/metadata/finalizers", "value":[]}]'
```

## Uninstalling Longhorn

Some of the CRDs might get stuck and patching the `finalizers` will break
because it will call out to a webhook service. You can delete the webhook
configurations by looking at these resources.
```sh
kubectl -n longhorn-system get mutatingwebhookconfigurations
kubectl -n longhorn-system get validatingadmissionpolicies
kubectl -n longhorn-system get validatingadmissionpolicybindings
kubectl -n longhorn-system get validatingwebhookconfigurations
```
Then remove the resources by using `kubectl patch`.

