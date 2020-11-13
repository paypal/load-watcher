# Load Watcher

The load watcher is responsible for the cluster-wide aggregation of resource usage metrics like CPU, memory, network, and IO stats over time windows from a metrics provider like SignalFx, Prometheus, K8s etc.
It stores them in its local cache, which can be queried from scheduler plugins.