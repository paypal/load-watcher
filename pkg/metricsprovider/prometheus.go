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
	"os"
	"time"

	"github.com/paypal/load-watcher/pkg/watcher"
	log "github.com/sirupsen/logrus"
	"github.com/prometheus/common/config"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
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
	promClientName = "Prometheus"
	promHostKey = "PROM_HOST"
	promTokenKey = "PROM_TOKEN"
	promStd = "stddev_over_time"
	promAvg = "avg_over_time"
	promCpuMetric = "instance:node_cpu:ratio"
	promMemMetric = "instance:node_memory_utilisation:ratio"
	allHosts = "all"
	hostMetricKey = "instance"
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
		log.Errorf("Error creating prometheus client: %v\n", err)
		return nil, err
	}

	return promClient{client}, err
}

func (s promClient) Name() string {
	return promClientName
}

func (s promClient) FetchHostMetrics(host string, window *watcher.Window) ([]watcher.Metric, error) {
	var metricList []watcher.Metric
	var anyerr error
	for _, method := range []string{promAvg, promStd} {
		for _, metric := range []string{promCpuMetric, promMemMetric} {
			promQuery := s.buildPromQuery(host, metric, method, window.Duration)
			promResults, err := s.getPromResults(promQuery)

			if err != nil {
				log.Errorf("Error querying Prometheus for query %v: %v\n", promQuery, err)
				anyerr = err
				continue
			}

			curMetricMap := s.promResults2MetricMap(promResults, metric, method, window.Duration)
			metricList = append(metricList, curMetricMap[host]...)
		}
	}

	return metricList, anyerr
}

// Fetch all host metrics with different operators (avg_over_time, stddev_over_time) and diffrent resource types (CPU, Memory)
func (s promClient) FetchAllHostsMetrics(window *watcher.Window) (map[string][]watcher.Metric, error) {
	hostMetrics := make(map[string][]watcher.Metric)
	var anyerr error
	for _, method := range []string{promAvg, promStd} {
		for _, metric := range []string{promCpuMetric, promMemMetric} {
			promQuery := s.buildPromQuery(allHosts, metric, method, window.Duration)
			promResults, err := s.getPromResults(promQuery)

			if err != nil {
				log.Errorf("Error querying Prometheus for query %v: %v\n", promQuery, err)
				anyerr = err
				continue
			}

			curMetricMap := s.promResults2MetricMap(promResults, metric, method, window.Duration)

			for k, v := range curMetricMap {
				hostMetrics[k] = append(hostMetrics[k], v...)
			}
		}
	}

	return hostMetrics, anyerr
}

func (s promClient) buildPromQuery(host string, metric string, method string, rollup string) string {
	var promQuery string
	if host == allHosts {
		promQuery = fmt.Sprintf("%s(%s[%s])", method, metric, rollup)
	} else {
		promQuery = fmt.Sprintf("%s(%s{%s=\"%s\"}[%s])", method, metric, hostMetricKey, host, rollup)
	}

	return promQuery
}

func (s promClient) getPromResults(promQuery string) (model.Value, error) {
	v1api := v1.NewAPI(s.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, warnings, err := v1api.Query(ctx, promQuery, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		log.Warnf("Warnings: %v\n", warnings)
	}
	log.Debugf("Result:\n%v\n", results)
	return results, nil
}

func (s promClient) promResults2MetricMap(promresults model.Value, metric string, method string, rollup string) map[string][]watcher.Metric {
	var metric_type string
	curMetrics := make(map[string][]watcher.Metric)

	if metric == promCpuMetric {
		metric_type = watcher.CPU
	} else {
		metric_type = watcher.Memory
	}

	switch promresults.(type) {
	case model.Vector:
		for _, result := range promresults.(model.Vector) {
			curMetric := watcher.Metric{metric, metric_type, method, rollup, float64(result.Value)}
			curHost := string(result.Metric[hostMetricKey])
			curMetrics[curHost] = append(curMetrics[curHost], curMetric)
		}
	default:
		log.Errorf("Error: The Prometheus results should not be type: %v.\n", promresults.Type())
	}

	return curMetrics
}