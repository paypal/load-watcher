# Load Watcher

The load watcher is responsible for the cluster-wide aggregation of resource usage metrics like CPU, memory, network, and IO stats over time windows from a metrics provider like SignalFx, Prometheus, Kubernetes etc. developed for [Trimaran: Real Load Aware Scheduling](https://github.com/kubernetes-sigs/scheduler-plugins/blob/master/kep/61-Trimaran-real-load-aware-scheduling/README.md) in Kubernetes.
It stores the metrics in its local cache, which can be queried from scheduler plugins.

The following metrics provider clients are currently supported:

1) SignalFx
2) Kubernetes Metrics Server

Any new clients need to implement `FetcherClient` interface.