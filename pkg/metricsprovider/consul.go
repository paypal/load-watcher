package metricsprovider

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/paypal/load-watcher/pkg/watcher"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

const (
	defaultMetricCheckID = "metrics"
)

var (
	metricCheckID string
	consulAddress string
)

/**
This client relies on following Consul health check script that is installed on each node:
{
      "checks": [
        {
          "id": "metrics",
          "name": "metrics",
          "args": ["sh", "-c", "uptime | awk '{print $(NF-2)$(NF-1)$(NF)}' && nproc"],
          "interval": "60s"
        }
     ]
}
Here load average from uptime is used as CPU Utilisation metric.
*/

type consulClient struct {
	client *api.Client
}

type healthCheckOutput struct {
	oneMinute     float64
	fiveMinute    float64
	fifteenMinute float64
	capacity      float64
}

func init() {
	consulAddress = os.Getenv("CONSUL_MASTER_ADDRESS")
	metricCheckID = os.Getenv("CONSUL_CHECK_ID")
	if metricCheckID == "" {
		metricCheckID = defaultMetricCheckID
	}
}

func NewConsulClient(config *api.Config) (watcher.FetcherClient, error) {
	if config == nil {
		config = api.DefaultConfig()
	}
	if consulAddress != "" {
		config.Address = consulAddress
	}
	c, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return consulClient{client: c}, nil
}

func (c consulClient) FetchHostMetrics(host string, window *watcher.Window) ([]watcher.Metric, error) {
	var metrics = []watcher.Metric{}
	healthChecks, _, err := c.client.Health().Node(host, nil)
	log.Debugf("found following health checks: %v", healthChecks)
	if err != nil {
		return metrics, err
	}
	for _, health := range healthChecks {
		if health.CheckID != metricCheckID {
			continue
		}
		var metric = watcher.Metric{Name: "load_average", Type: watcher.CPU}
		output, err := parseHealthCheckOutput(health.Output)
		if err != nil {
			return metrics, err
		}
		switch window.Duration {
		case watcher.FifteenMinutes:
			metric.Value = output.fifteenMinute / output.capacity
		// Note: this mapping is different as consul health check depends on uptime, which provides 15m, 5m and 1m load averages
		// Since 15m is our first preference and rest are fall backs, it is okay.
		case watcher.TenMinutes:
			metric.Value = output.fiveMinute / output.capacity
		case watcher.FiveMinutes:
			metric.Value = output.oneMinute / output.capacity
		}
		metric.Value = 100 * metric.Value
		metrics = append(metrics, metric)
		break
	}
	return metrics, nil
}

func (c consulClient) FetchAllHostsMetrics(window *watcher.Window) (map[string][]watcher.Metric, error) {
	metrics := make(map[string][]watcher.Metric)
	nodes, _, err := c.client.Catalog().Nodes(nil)
	if err != nil {
		return metrics, err
	}
	for _, node := range nodes {
		nodeMetrics, err := c.FetchHostMetrics(node.Node, window)
		if err != nil {
			log.Errorf("error while fetching metrics for host %v: %v", node.Node, err)
		} else {
			metrics[node.Node] = nodeMetrics
		}
	}
	return metrics, nil
}


// Sample output "0.1,0.1,0.1\n2"
func parseHealthCheckOutput(rawOutput string) (healthCheckOutput, error) {
	var output healthCheckOutput
	rawOutput = strings.ReplaceAll(rawOutput, ",", " ")
	_, err := fmt.Sscan(rawOutput, &output.oneMinute, &output.fiveMinute, &output.fifteenMinute, &output.capacity)
	if err != nil {
		return output, err
	}
	return output, nil
}
