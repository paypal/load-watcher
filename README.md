# Load Watcher [![Go Reference](https://pkg.go.dev/badge/github.com/paypal/load-watcher.svg)](https://pkg.go.dev/github.com/paypal/load-watcher) ![CI Build Status](https://github.com/paypal/load-watcher/actions/workflows/ci.yml/badge.svg) [![Automated Release Notes by gren](https://img.shields.io/badge/%F0%9F%A4%96-release%20notes-00B2EE.svg)](https://github-tools.github.io/github-release-notes/)

The load watcher is responsible for the cluster-wide aggregation of resource usage metrics like CPU, memory, network, and IO stats over time windows from a metrics provider like SignalFx, Prometheus, Kubernetes Metrics Server etc. developed for [Trimaran: Real Load Aware Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/61-Trimaran-real-load-aware-scheduling/README.md) in Kubernetes.
It stores the metrics in its local cache, which can be queried from scheduler plugins.

The following metrics provider clients are currently supported:

1) SignalFx
2) Kubernetes Metrics Server
3) Prometheus
4) Datadog

These clients fetch CPU usage currently, support for other resources will be added later as needed.

## Compatibility Matrix

The following table outlines the compatibility between `load-watcher` releases, supported `scheduler-plugins` (Trimaran) versions, and the corresponding Kubernetes / `client-go` versions:

| `load-watcher` Version | `scheduler-plugins` (Trimaran) | Target Kubernetes Version | `client-go` Version |
| :--- | :--- | :--- | :--- |
| **v0.2.4** | v0.31.x – v0.35.x+ | v1.31.x – v1.35.x+ | v0.31.2 |
| **v0.2.3** | v0.26.x – v0.30.x | v1.26.x – v1.30.x | v0.22.3 |
| **v0.2.2** | v0.23.x – v0.25.x | v1.23.x – v1.25.x | v0.22.3 |
| **v0.2.1** | v0.22.x | v1.22.x | v0.22.3 |
| **v0.2.0** | v0.21.x | v1.21.x | v0.19.0 |
| **v0.1.1** | v0.19.x – v0.20.x | v1.19.x – v1.20.x | v0.19.0 |
| **v0.1.0** | Initial release | v1.19.x | v0.19.0 |

# Tutorial

This tutorial will guide you to build load watcher Docker image, which can be deployed to work with Trimaran scheduler plugins.

The default `main.go` is configured to watch Kubernetes Metrics Server.
You can change this to any available metrics provider in `pkg/metricsprovider`.
To build a client for new metrics provider, you will need to implement `FetcherClient` interface.

## Building behind a corporate TLS proxy

Some corporate environments may fail during Go module downloads or base image pulls with errors such as:

```text
x509: certificate signed by unknown authority

Export the corporate root CA certificate:

security find-certificate -c "PayPal Crypto Mgt Corp Root CA" -p \
  /Library/Keychains/System.keychain > paypal-crypto-root-ca.crt

From the root folder, build the load-watcher image using Docker BuildKit:

docker buildx build \
  --platform linux/amd64 \
  --secret id=corp_ca,src="$(pwd)/paypal-crypto-root-ca.crt" \
  --build-arg BUILD_SPEC=build.amd64 \
  --build-arg GOPROXY=direct \
  --build-arg GOSUMDB=off \
  --load \
  -t load-watcher:<version> \
  .

Notes:

corp_ca is an optional BuildKit secret used to install the corporate root CA inside the builder container.
GOPROXY=direct and GOSUMDB=off can help in environments where access to proxy.golang.org is restricted.
linux/arm64 builds succeed locally. linux/amd64 builds on Apple Silicon/Rancher Desktop can intermittently hit Go compiler segmentation faults under emulation, so amd64 images should be validated on CI or an amd64 runner.

Tag the docker image and push to your docker repository:

```
docker tag load-watcher:<version> <your-docker-repo>:<version>
docker push <your-docker-repo>
```

Note that load watcher runs on default port 2020. Once deployed, you can use the following API to read watcher metrics:

```
GET /watcher
```

This will return metrics for all nodes. A query parameter to filter by host can be added with `host`.

## Metrics Provider Configuration
- By default Kubernetes Metrics Server client is configured. Set `KUBE_CONFIG` env var to your kubernetes client configuration file path if running out of cluster.

- To use the Prometheus client, please configure environment variables `METRICS_PROVIDER_NAME`, `METRICS_PROVIDER_ADDRESS` and `METRICS_PROVIDER_TOKEN` to `Prometheus`, Prometheus address and auth token. Please do not set `METRICS_PROVIDER_TOKEN` if no authentication 
  is needed to access the Prometheus APIs. Default value of address set is `http://prometheus-k8s:9090` for Prometheus client.

- To use the SignalFx client, please configure environment variables `METRICS_PROVIDER_NAME`, `METRICS_PROVIDER_ADDRESS` and `METRICS_PROVIDER_TOKEN` to `SignalFx`, SignalFx address and auth token respectively. Default value of address set is `https://api.signalfx.com` for SignalFx client.
  
- To use the Datadog client, configure `METRICS_PROVIDER_NAME`, `METRICS_PROVIDER_ADDRESS`, `METRICS_PROVIDER_TOKEN`, `METRICS_PROVIDER_APP_KEY`, `DATADOG_ClUSTER_FILTER`, and `DATADOG_CLUSTER_NAME`.

## Deploy `load-watcher` as a service
To deploy `load-watcher` as a monitoring service in your Kubernetes cluster, you should replace the values in the `[]` with your own cluster monitoring stack and then you can run the following.
```bash
> kubectl create -f manifests/load-watcher-deployment.yaml
```

## Using `load-watcher` client
- `load-watcher-client.go` shows an example to use `load-watcher` packages as libraries in a client mode. When `load-watcher` is running as a
service exposing an endpoint in a cluster, a client, such as Trimaran plugins, can use its libraries to create a client getting the latest metrics.
