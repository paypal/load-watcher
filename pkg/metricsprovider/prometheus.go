/*
Copyright 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metricsprovider

import (
	"context"
	"fmt"
	"github.com/prometheus/common/config"
	"os"
	"time"


	"github.com/paypal/load-watcher/pkg/watcher"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var (
	promHost    string
	promToken	string
	promTokenPresent = false
	node_metric_query = map[string]string{
		watcher.CPU : 	"instance:node_cpu:ratio",
		watcher.Memory : "instance:node_memory_utilization:ratio",
	}

)

const (
	// env variable that provides path to kube config file, if deploying from outside K8s cluster
	promClientName = "Prometheus"
	promHostKey = "PROM_HOST"
	promTokenKey = "PROM_TOKEN"
	prom_std_method = "stddev_over_time"
	prom_avg_method = "avg_over_time"
	prom_cpu_metric = "instance:node_cpu:ratio"
	prom_mem_metric = "instance:node_memory_utilisation:ratio"
	all_hosts = "all"
)

func init() {
	var promHostPresent bool

	promHost, promHostPresent = os.LookupEnv(promHostKey)
	promToken, promTokenPresent = os.LookupEnv(promTokenKey)
	if !promHostPresent {
		promHost = "http://prometheus-k8s:9090"
	}
}

type promClient struct {
	client api.Client
}

func NewPromClient() (watcher.FetcherClient, error) {
	var client api.Client
	var err error

	if !promTokenPresent {
		client, err = api.NewClient(api.Config{
			Address: promHost,
		})
	} else {
		client, err = api.NewClient(api.Config{
			Address: promHost,
			RoundTripper: config.NewBearerAuthRoundTripper(config.Secret(promToken), api.DefaultRoundTripper),
		})
	}

	if err != nil {
		fmt.Printf("Error creating prometheus client: %v\n", err)
	}

	return promClient{client}, err
}

func (s promClient) Name() string {
	return promClientName
}

func (s promClient) FetchHostMetrics(host string, window *watcher.Window) ([]watcher.Metric, error) {
	var metricList []watcher.Metric

	for _, method := range []string{prom_avg_method, prom_std_method} {
		for _, metric := range []string{prom_cpu_metric, prom_mem_metric} {
			promQuery := s.buildPromQuery(host, metric, method, window.Duration)
			promResults := s.getPromRsts(promQuery)
			curMetricMap := s.promRsts2MetricMap(promResults, metric, method, window.Duration)
			metricList = append(metricList, curMetricMap[host]...)
		}
	}

	return metricList, nil
}

// Fetch all host metrics for all methods and resource types
func (s promClient) FetchAllHostsMetrics(window *watcher.Window) (map[string][]watcher.Metric, error) {
	hostMetrics := make(map[string][]watcher.Metric)

	for _, method := range []string{prom_avg_method, prom_std_method} {
		for _, metric := range []string{prom_cpu_metric, prom_mem_metric} {
			promQuery := s.buildPromQuery(all_hosts, metric, method, window.Duration)
			promResults := s.getPromRsts(promQuery)
			curMetricMap := s.promRsts2MetricMap(promResults, metric, method, window.Duration)

			for k, v := range curMetricMap {
				hostMetrics[k] = append(hostMetrics[k], v...)
			}
		}
	}

	return hostMetrics, nil
}

func (s promClient) buildPromQuery(host string, metric string, method string, rollup string) string {
	var promQuery string
	if host == all_hosts {
		promQuery = fmt.Sprintf("%s(%s[%s])", method, metric, rollup)
	} else {
		promQuery = fmt.Sprintf("%s(%s{instance=\"%s\"}[%s])", method, metric, host, rollup)
	}

	return promQuery
}

func (s promClient) getPromRsts(promQuery string) model.Value {
	v1api := v1.NewAPI(s.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, warnings, err := v1api.Query(ctx, promQuery, time.Now())
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		os.Exit(1)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}
	fmt.Printf("Result:\n%v\n", results.Type())

	return results
}


func (s promClient) promRsts2MetricMap(promrst model.Value, metric string, method string, rollup string) map[string][]watcher.Metric {
	var metric_type string
	curMetrics := make(map[string][]watcher.Metric)

	if metric == prom_cpu_metric {
		metric_type = watcher.CPU
	} else {
		metric_type = watcher.Memory
	}

	switch promrst.(type) {
	case model.Vector:
		for _, result := range promrst.(model.Vector) {
			curMetric := watcher.Metric{metric, metric_type, method, rollup, float64(result.Value)}
			curHost := string(result.Metric["instance"])
			curMetrics[curHost] = append(curMetrics[curHost], curMetric)
		}
	default:
		panic(fmt.Sprintf("The Prometheus results should not be type: %v.", promrst.Type()))
	}

	return curMetrics
}