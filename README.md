# Prometheus Exporter for [Vultr](https://vultr.com)

> **NOTE** Currently only supports querying [Vultr Kubernetes Engine (VKE)](https://www.vultr.com/kubernetes/) clusters

[![build-containers](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml/badge.svg)](https://github.com/DazWilkin/vultr-exporter/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/DazWilkin/vultr-exporter.svg)](https://pkg.go.dev/github.com/DazWilkin/vultr-exporter)
[![Go Report Card](https://goreportcard.com/badge/github.com/dazwilkin/vultr-exporter)](https://goreportcard.com/report/github.com/dazwilkin/vultr-exporter)

## Image

+ `ghcr.io/dazwilkin/vultr-exporter:73800c719f1d4507e27ce29df16257a4fc978589`

## API Key

The Exporter needs access to your Vultr API Key

```bash
export API_KEY="[YOUR-API-KEY]"
```

## Metrics

|Name|Type|Description|
|----|----|-----------|
|`vultr_exporter_build_info`|Counter|Build status (1=running)|
|`vultr_exporter_kubernetes_cluster_up`|Gauge|Number of Kubernetes clusters|
|`vultr_exporter_kubernetes_node_pool`|Gauge|Number of Kubernetes cluster Node Pools|
|`vultr_exporter_kubernetes_node`|Gauge|Number of Kubernetes Cluster Nodes|

```bash
# HELP vultr_exporter_build_info A metric with a constant '1' value labeled by OS version, Go version, and the Git commit of the exporter
# TYPE vultr_exporter_build_info counter
vultr_exporter_build_info{git_commit="",go_version="go1.18",os_version=""} 1
# HELP vultr_exporter_kubernetes_cluster_up 1 if the cluster is running, 0 otherwise
# TYPE vultr_exporter_kubernetes_cluster_up counter
vultr_exporter_kubernetes_cluster_up{label="ackal",region="sea",status="active",version="v1.23.5+3"} 1
# HELP vultr_exporter_kubernetes_node Number of Nodes associated with the cluster
# TYPE vultr_exporter_kubernetes_node gauge
vultr_exporter_kubernetes_node{label="nodepool",plan="vc2-1c-2gb",status="active",tag="dev"} 1
# HELP vultr_exporter_kubernetes_node_pool Number of Node Pools associated with the cluster
# TYPE vultr_exporter_kubernetes_node_pool gauge
vultr_exporter_kubernetes_node_pool{label="ackal",region="sea",status="active",version="v1.23.5+3"} 1
# HELP vultr_exporter_start_time Exporter start time in Unix epoch seconds
# TYPE vultr_exporter_start_time gauge
vultr_exporter_start_time 1.653085629e+09```

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

IMAGE="ghcr.io/dazwilkin/vultr-exporter:73800c719f1d4507e27ce29df16257a4fc978589"

podman run \
--interactive --tty --rm \
--publish=8080:8080 \
--env=API_KEY=${API_KEY} \
${IMAGE} \
  --endpoint=0.0.0.0:8080 \
  --path=/metrics
```

## Kubernetes

```bash
API_KEY="[YOUR-API-KEY]"

IMAGE="ghcr.io/dazwilkin/vultr-exporter:ada6819cb8f9b7886a468b92ab83d04bb91c0967"

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
ghcr.io/dazwilkin/vultr-exporter:73800c719f1d4507e27ce29df16257a4fc978589
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