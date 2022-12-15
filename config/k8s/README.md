# Kubernetes Configuration

## Redis DB Index Allocaion

This is how redis database indexes *should* be allocated. The valid range is 0-15

| service  | redis index |
| -------- | ----------- |
| grafana  | 4           |
| registry | 5           |
