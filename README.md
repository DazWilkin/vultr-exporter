# Prometheus Exporter for [Vultr](https://vultr.com)

[![build-containers](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml/badge.svg)](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/DazWilkin/vultr-exporter.svg)](https://pkg.go.dev/github.com/DazWilkin/vultr-exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/dazwilkin/vultr-exporter)](https://goreportcard.com/report/github.com/dazwilkin/vultr-exporter)

## Metrics

Metrics are all prefixed `vultr_`

|Name|Type|Description|
|----|----|-----------|
|`block_storage_up`|Counter|Number of Block Storage volumes|
|`block_storage_size`|Gauge|Size (GB) of Block Storage volumes|
|`exporter_build_info`|Counter|Build status (1=running)|
|`exporter_start_time`|Gauge|Start time (Unix epoch) of Exporter|
|`kubernetes_cluster_up`|Counter|Number of Kubernetes clusters|
|`kubernetes_node_pool`|Gauge|Number of Kubernetes cluster Node Pools|
|`kubernetes_node`|Gauge|Number of Kubernetes Cluster Nodes|
|`load_balancer_up`|Number of Load Balancers|
|`load_balancer_instances`|Number of Load Balancer instances|
|`reserved_ips_up`|Counter|Number of Reserved IPs|

## Image

+ `ghcr.io/dazwilkin/vultr-exporter:717ff3b08097eee49cb3734df4ea2c4bb37d246e`

## API Key

The Exporter needs access to your Vultr API Key

```bash
export API_KEY="[YOUR-API-KEY]"
```

## Image

+ `ghcr.io/dazwilkin/vultr-exporter:717ff3b08097eee49cb3734df4ea2c4bb37d246e`

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

IMAGE="ghcr.io/dazwilkin/vultr-exporter:717ff3b08097eee49cb3734df4ea2c4bb37d246e"

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

IMAGE="ghcr.io/dazwilkin/vultr-exporter:717ff3b08097eee49cb3734df4ea2c4bb37d246e"

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

`vultr-exporter` container images are being signed by Sigstore and may be verified:

```bash
cosign verify \
--key=./cosign.pub \
ghcr.io/dazwilkin/vultr-exporter:717ff3b08097eee49cb3734df4ea2c4bb37d246e
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

<hr/>
<br/>
<a href="https://www.buymeacoffee.com/dazwilkin" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/default-orange.png" alt="Buy Me A Coffee" height="41" width="174"></a>