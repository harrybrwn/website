# Kubernetes Configuration

## Redis DB Index Allocation

This is how redis database indexes *should* be allocated. The valid range is 0-15

| service  | redis index | note             |
| -------- | ----------- | ----             |
| mastodon | 0           | Not configurable |
| grafana  | 4           | |
| registry | 5           | |

# Resources

- [kubernetes object definitions](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/)