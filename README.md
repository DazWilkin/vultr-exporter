# Prometheus Exporter for [Vultr](https://vultr.com)

[![GitHub Actions](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml/badge.svg)](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/DazWilkin/vultr-exporter.svg)](https://pkg.go.dev/github.com/DazWilkin/vultr-exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/dazwilkin/vultr-exporter)](https://goreportcard.com/report/github.com/dazwilkin/vultr-exporter)

## Metrics

Metrics are all prefixed `vultr_`

| Name                      | Type    | Description                                                           |
| ------------------------- | ------- | --------------------------------------------------------------------- |
| `account_balance`         | Gauge   | Account Balance                                                       |
| `account_bandwidth_value` | Gauge   | Account bandwidth metrics with period, metric type and unit as labels |
| `account_pending_charges` | Gauge   | Pending Charges                                                       |
| `billing_cost_usd`        | Gauge   | Total cost in USD per product instance                                |
| `billing_units`           | Gauge   | Number of units consumed per product instance                         |
| `block_storage_up`        | Counter | Number of Block Storage volumes                                       |
| `block_storage_size`      | Gauge   | Size (GB) of Block Storage volumes                                    |
| `exporter_build_info`     | Counter | Build status (1=running)                                              |
| `exporter_start_time`     | Gauge   | Start time (Unix epoch) of Exporter                                   |
| `kubernetes_cluster_up`   | Counter | Number of Kubernetes clusters                                         |
| `kubernetes_node_pool`    | Gauge   | Number of Kubernetes cluster Node Pools                               |
| `kubernetes_node_pool_nodes` | Gauge | Number of Kubernetes Cluster Nodes                                   |
| `load_balancer_up`        | Counter | Number of Load Balancers                                              |
| `load_balancer_instances` | Gauge   | Number of Load Balancer instances                                     |
| `reserved_ips_up`         | Counter | Number of Reserved IPs                                                |

### Account Bandwidth

Account Bandwidth metrics use a single metric `vultr_account_bandwidth_value` with labels to distinguish different types of measurements:

- `period`: Time period of the measurement
  - `current`: Current month to date
  - `previous`: Previous month
  - `projected`: Projected for current month
- `metric`: Type of measurement
  - `gb_in`: Ingress Bandwidth
  - `gb_out`: Egress Bandwidth
  - `total_instance_hours`: Total Instance Hours
  - `total_instance_count`: Total Instance Count
  - `instance_bandwidth_credits`: Instance Bandwidth Credits
  - `free_bandwidth_credits`: Free Bandwidth Credits
  - `purchased_bandwidth_credits`: Purchased Bandwidth Credits
  - `overage`: Bandwidth Overage
  - `overage_unit_cost`: Overage Unit Cost
  - `overage_cost`: Overage Cost
- `unit`: Unit of measurement (GB, hours, count, credits, USD)

### Billing

Billing metrics provide cost and usage information for each Vultr product instance:

| Name               | Type  | Labels                                      | Description                                     |
| ------------------ | ----- | ------------------------------------------- | ----------------------------------------------- |
| `billing_cost_usd` | Gauge | product, description                        | Total cost in USD for a product instance        |
| `billing_units`    | Gauge | product, description, unit_type, unit_price | Number of units consumed with price information |

The `product` and `description` labels uniquely identify each resource (e.g., specific Load Balancer, Instance, etc.).

### Block Storage

Block Storage metrics include the following labels:
- `id`: Block Storage ID
- `label`: Block Storage label
- `region`: Region where the Block Storage is located
- `status`: Current status of the Block Storage
- `block_type`: Type of Block Storage

### Prometheus Query Examples

Here are some useful PromQL queries:

```promql
# Total cost across all products
sum(vultr_billing_cost_usd)

# Cost by product type
sum(vultr_billing_cost_usd) by (product)

# Find most expensive resources
topk(5, vultr_billing_cost_usd)

# Total bandwidth usage current month
sum(vultr_account_bandwidth_value{period="current",metric=~"gb_.*"})

# Current month bandwidth by direction
sum(vultr_account_bandwidth_value{period="current"}) by (metric) 

# Resources with high unit prices
topk(5, vultr_billing_units * on(unit_price) scalar(vultr_billing_units{unit_price!="0"}))

# Projected vs current bandwidth usage
sum(vultr_account_bandwidth_value{metric=~"gb_.*"}) by (period)

# Total instance hours by period
vultr_account_bandwidth_value{metric="total_instance_hours"}

# Bandwidth overage cost projection
vultr_account_bandwidth_value{period="projected",metric="overage_cost"}

# Count of nodes by Kubernetes cluster
sum(vultr_kubernetes_node_pool_nodes) by (label)

# Block storage by type
sum(vultr_block_storage_size) by (block_type)
```

## Image

+ `ghcr.io/dazwilkin/vultr-exporter:dd9d5ac93d26fa43bb0058c8ceb49bee1ce285f8`

## API Key

The Exporter needs access to your Vultr API Key

```bash
export API_KEY="[YOUR-API-KEY]"
```

## Image

+ `ghcr.io/dazwilkin/vultr-exporter:dd9d5ac93d26fa43bb0058c8ceb49bee1ce285f8`

## API Key

The Exporter needs access to your Vultr API Key

```bash
export API_KEY="[YOUR-API-KEY]"
```

## Go

```bash
export API_KEY="[YOUR-API-KEY]"

go run ./cmd/server \
--endpoint=0.0.0.0:8080 \
--path=/metrics
```

## Container

```bash
API_KEY="[YOUR-API-KEY]"

IMAGE="ghcr.io/dazwilkin/vultr-exporter:dd9d5ac93d26fa43bb0058c8ceb49bee1ce285f8"

podman run \
--interactive --tty --rm \
--publish=8080:8080 \
--env=API_KEY=${API_KEY} \
${IMAGE} \
  --endpoint=0.0.0.0:8080 \
  --path=/metrics
```

## Kubernetes

> **NOTE** If running `vult-exporter` on VKE, ensure that you're `API_KEY` includes the public IP addresses of the cluster's nodes as these will be originating Vultr API requests. I think these access control changes can't be done programmatically.

```bash
API_KEY="[YOUR-API-KEY]"

IMAGE="ghcr.io/dazwilkin/vultr-exporter:dd9d5ac93d26fa43bb0058c8ceb49bee1ce285f8"

NAMESPACE="exporter"

kubectl create namespace ${NAMESPACE}

kubectl create secret generic vultr \
--namespace=${NAMESPACE} \
--from-literal=apiKey=${API_KEY}

echo "
apiVersion: v1
kind: List
metadata: {}
items:
  - kind: Service
    apiVersion: v1
    metadata:
      labels:
        app: vultr-exporter
      name: vultr-exporter
    spec:
      selector:
        app: vultr-exporter
      ports:
        - name: http
          port: 8080
          targetPort: 8080
  - kind: Deployment
    apiVersion: apps/v1
    metadata:
      labels:
        app: vultr-exporter
      name: vultr-exporter
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: vultr-exporter
      template:
        metadata:
          labels:
            app: vultr-exporter
        spec:
          containers:
            - name: vultr-exporter
              image: ${IMAGE}
              command:
              - /server
              args:
              - --endpoint=0.0.0.0:8080
              - --path=/metrics
              env:
                - name: API_KEY
                  valueFrom:
                    secretKeyRef:
                      name: vultr
                      key: apiKey
                      optional: false
              ports:
                - name: metrics
                  containerPort: 8080
          restartPolicy: Always
" | kubectl apply --filename=- --namespace=${NAMESPACE}

# Use your preferred HTTP Load-balancer
kubectl port-forward deployment/vultr-exporter 8080:8080 \
--namespace=${NAMESPACE}

# To use a Vultr Load balancer
# Replaces the service created above w/ a Vultr Load balancer
echo "
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/vultr-loadbalancer-protocol: "http"
  name: vultr-exporter
spec:
  type: LoadBalancer
  selector:
    app: vultr-exporter
  ports:
    - name: http
      port: 80
      targetPort: 8080
" | kubectl apply --filename=- --namespace=${NAMESPACE}
```

## Raspberry Pi 4

```bash
if [ "$(getconf LONG_BIT)" -eq 64 ]
then
  # 64-bit Raspian
  ARCH="GOARCH=arm64"
  TAG="arm64"
else
  # 32-bit Raspian
  ARCH="GOARCH=arm GOARM=7"
  TAG="arm32v7"
fi

podman build \
--build-arg=GOLANG_OPTIONS="CGO_ENABLED=0 GOOS=linux ${ARCH}" \
--build-arg=COMMIT=$(git rev-parse HEAD) \
--build-arg=VERSION=$(uname --kernel-release) \
--tag=ghcr.io/dazwilkin/vultr-exporter:${TAG} \
--file=./Dockerfile \
.
```

## Prometheus

```YAML
global:
  scrape_interval: 1m
  evaluation_interval: 1m

rule_files:
- "/etc/alertmanager/rules.yml"

  # Vultr Exporter
- job_name: "vultr-exporter"
  static_configs:
  - targets:
    - "localhost:8080"
```

## Alertmanager

```YAML
groups:
- name: vultr_exporter
  rules:
  - alert: vultr_kubernetes_cluster_up
    expr: vultr_kubernetes_cluster_up{} > 0
    for: 6h
    labels:
      severity: page
    annotations:
      summary: Vultr Kubernetes Engine clusters
```

## Sigstore

`vultr-exporter` container images are being signed by [Sigstore](https://www.sigstore.dev/) and may be verified:

```bash
cosign verify \
--key=./cosign.pub \
ghcr.io/dazwilkin/vultr-exporter:dd9d5ac93d26fa43bb0058c8ceb49bee1ce285f8
```

> **NOTE** cosign.pub may be downloaded [here](/cosign.pub)

To install cosign, e.g.:

```bash
go install github.com/sigstore/cosign/cmd/cosign@latest
```

## References

+ [Vultr API: List all Kubernetes Clusters](https://www.vultr.com/api/#operation/create-kubernetes-cluster)
+ [govultr SDK](https://github.com/vultr/govultr)
+ [pkg.go.dev: Cluster](https://pkg.go.dev/github.com/vultr/govultr/v2#Cluster)
+ [pkg.go.dev: NodePool](https://pkg.go.dev/github.com/vultr/govultr/v2#NodePool)

## Similar Exporters

+ [Prometheus Exporter for Azure](https://github.com/DazWilkin/azure-exporter)
+ [Prometheus Exporter for crt.sh](https://github.com/DazWilkin/crtsh-exporter)
+ [Prometheus Exporter for Fly.io](https://github.com/DazWilkin/fly-exporter)
+ [Prometheus Exporter for GoatCounter](https://github.com/DazWilkin/goatcounter-exporter)
+ [Prometheus Exporter for Google Cloud](https://github.com/DazWilkin/gcp-exporter)
+ [Prometheus Exporter for Koyeb](https://github.com/DazWilkin/koyeb-exporter)
+ [Prometheus Exporter for Linode](https://github.com/DazWilkin/linode-exporter)
+ [Prometheus Exporter for PorkBun](https://github.com/DazWilkin/porkbun-exporter)
+ [Prometheus Exporter for updown.io](https://github.com/DazWilkin/updown-exporter)
+ [Prometheus Exporter for Vultr](https://github.com/DazWilkin/vultr-exporter)

<hr/>
<br/>
<a href="https://www.buymeacoffee.com/dazwilkin" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>
