# Load Watcher

The load watcher is responsible for the cluster-wide aggregation of resource usage metrics like CPU, memory, network, and IO stats over time windows from a metrics provider like SignalFx, Prometheus, Kubernetes Metrics Server etc. developed for [Trimaran: Real Load Aware Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/61-Trimaran-real-load-aware-scheduling/README.md) in Kubernetes.
It stores the metrics in its local cache, which can be queried from scheduler plugins.

The following metrics provider clients are currently supported:

1) SignalFx
2) Kubernetes Metrics Server
3) Prometheus

These clients fetch CPU usage currently, support for other resources will be added later as needed.

# Tutorial

This tutorial will guide you to build load watcher Docker image, which can be deployed to work with Trimaran scheduler plugins.

The default `main.go` is configured to watch Kubernetes Metrics Server.
You can change this to any available metrics provider in `pkg/metricsprovider`.
To build a client for new metrics provider, you will need to implement `FetcherClient` interface.

First build load watcher binary with the following command in `main.go` file and save the built binary as `load-watcher`:

```
go build -o load-watcher main.go
```

If you are cross compiling for Linux 64 bit OS, use the following command:

```
env GOARCH=amd64 GOOS=linux go build -o load-watcher main.go
```

From the root folder, run the following commands to build docker image of load watcher, tag it and push to your docker repository:

```
docker build -t load-watcher .
docker tag load-watcher:latest <your-docker-repo>:latest
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

- To use the SignalFx client, please configure `METRICS_PROVIDER_NAME`, `METRICS_PROVIDER_ADDRESS` and `METRICS_PROVIDER_TOKEN` to `SignalFx`, SignalFx address and auth token respectively. Default value of address set is `https://api.signalfx.com` for Prometheus client.
  
## Deploy `load-watcher` as a service
To deploy `load-watcher` as a monitoring service in your Kubernetes cluster, you can run the following.
```bash
> kubectl create -f manifests/load-watcher-deployment.yaml
```

## Using `load-watcher` client
- `load-watcher-client.go` shows an example to use `load-watcher` packages as libraries in a client mode. When `load-watcher` is running as a
service exposing an endpoint in a cluster, a client, such as Trimaran plugins, can use its libraries to create a client getting the latest metrics. 