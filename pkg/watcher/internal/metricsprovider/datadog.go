/*
Copyright 2024 PayPal

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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"net/http"
	"os"
	"strings"

	"github.com/paypal/load-watcher/pkg/watcher"
	log "github.com/sirupsen/logrus"
)

const (
	// Datadog Request Params
	DefaultDatadogAddress    = "datadoghq.com"
	datadogHostFilter        = "host:"
	datadogClusterFilter     = "cluster_name:"
	datadogHostNameSuffixKey = "DATADOG_HOST_NAME_SUFFIX"
	datadogClusterName       = "DATADOG_CLUSTER_NAME"
	// Datadog Query Params
	datadogOneMinuteResolutionMs   = 60000
	datadogCpuUtilizationMetric    = "max:cpu.utilization"
	datadogMemoryUtilizationMetric = "max:memory.utilization"
	datadogRollup                  = "60"
)

type datadogClient struct {
	client         http.Client
	authToken      string
	applicationKey string
	datadogAddress string
	hostNameSuffix string
	clusterName    string
}

// This method creates a new datadog client based on the environment variables
func NewDatadogClient(opts watcher.MetricsProviderOpts) (watcher.MetricsProviderClient, error) {
	if opts.Name != watcher.DatadogClientName {
		return nil, fmt.Errorf("metric provider name should be %v, found %v", watcher.DatadogClientName, opts.Name)
	}
	tlsConfig := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: opts.InsecureSkipVerify}, // TODO(lawwong): Figure out a secure way to let users add SSL certs
	}
	hostNameSuffix, _ := os.LookupEnv(datadogHostNameSuffixKey)
	clusterName, _ := os.LookupEnv(datadogClusterName)
	var datadogAddress, datadogAuthToken, datadogApplicationKey = DefaultDatadogAddress, "", ""
	if opts.Address != "" {
		datadogAddress = opts.Address
	}
	if opts.AuthToken != "" {
		datadogAuthToken = opts.AuthToken
	}
	if opts.ApplicationKey != "" {
		datadogApplicationKey = opts.ApplicationKey
	}
	if datadogApplicationKey == "" {
		log.Fatalf("No application key found to connect with datadog server")
	}
	if datadogAuthToken == "" {
		log.Fatalf("No api key found to connect with datadog server")
	}
	return datadogClient{client: http.Client{
		Timeout:   httpClientTimeout,
		Transport: tlsConfig},
		authToken:      datadogAuthToken,
		applicationKey: datadogApplicationKey,
		datadogAddress: datadogAddress,
		hostNameSuffix: hostNameSuffix,
		clusterName:    clusterName}, nil
}

func (s datadogClient) Name() string {
	return watcher.DatadogClientName
}

// This function fetches metrics for a host during the watcher.window.
// It returns an array of watcher.Metric
func (s datadogClient) FetchHostMetrics(host string, window *watcher.Window) ([]watcher.Metric, error) {
	log.Debugf("fetching metrics for host %v", host)
	metricsMap, err := s.getMetricsHelper(window, host)
	if err == nil {
		// Get the value from the map with only 1 key, hostname
		for _, metrics := range metricsMap {
			return metrics, err
		}
	}
	return []watcher.Metric{}, err
}

// This function fetches all hosts metrics during the watcher.window.
// It returns a map of hostname and an array of watcher.Metric
func (s datadogClient) FetchAllHostsMetrics(window *watcher.Window) (map[string][]watcher.Metric, error) {
	return s.getMetricsHelper(window, "*")
}

func (s datadogClient) Health() (int, error) {
	return ping(s.client, "https://"+s.datadogAddress)
}

// Simple ping utility to a given URL
// Returns -1 if unhealthy, 0 if healthy along with error if any
func ping(client http.Client, url string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	if resp == nil {
		return -1, fmt.Errorf("No response from datadog server: %v", url)
	}
	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("received response code: %v", resp.StatusCode)
	}
	return 0, nil
}

// This function adds metadata for watcher.Metric
func addDatadogMetadata(metric *watcher.Metric, isCPU bool) {
	if metric != nil {
		metric.Operator = watcher.Average
		metric.Rollup = "rollup(max, " + datadogRollup + ")"
		if isCPU {
			metric.Name = datadogCpuUtilizationMetric
			metric.Type = watcher.CPU
		} else {
			metric.Name = datadogMemoryUtilizationMetric
			metric.Type = watcher.Memory
		}
	}
}

// This method constructs datadog query for CPU and memory metrics for all/a host(s)
// It returns a map of hostname and array of watcher.Metric
func (s datadogClient) getMetricsHelper(window *watcher.Window, host string) (map[string][]watcher.Metric, error) {
	ctx := context.WithValue(
		context.Background(),
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: s.authToken,
			},
			"appKeyAuth": {
				Key: s.applicationKey,
			},
		},
	)
	hostFilter := datadogHostFilter + host + s.hostNameSuffix
	clusterFilter := datadogClusterFilter + s.clusterName
	filter := "{" + hostFilter + ", " + clusterFilter + "}"
	rollup := ".rollup(max, " + datadogRollup + ")"
	cpuquery := datadogCpuUtilizationMetric + filter + " by {host}" + rollup
	memoryquery := datadogMemoryUtilizationMetric + filter + " by {host}" + rollup
	cpuqueryname := "cpuqueryname"
	memoryqueryname := "memoryqueryname"
	body := datadogV2.TimeseriesFormulaQueryRequest{
		Data: datadogV2.TimeseriesFormulaRequest{
			Attributes: datadogV2.TimeseriesFormulaRequestAttributes{
				From:     window.Start * 1000,
				Interval: datadog.PtrInt64(datadogOneMinuteResolutionMs),
				Queries: []datadogV2.TimeseriesQuery{
					datadogV2.TimeseriesQuery{
						MetricsTimeseriesQuery: &datadogV2.MetricsTimeseriesQuery{
							Name:       &cpuqueryname,
							DataSource: datadogV2.METRICSDATASOURCE_METRICS,
							Query:      cpuquery,
						}},
					datadogV2.TimeseriesQuery{
						MetricsTimeseriesQuery: &datadogV2.MetricsTimeseriesQuery{
							Name:       &memoryqueryname,
							DataSource: datadogV2.METRICSDATASOURCE_METRICS,
							Query:      memoryquery,
						}},
				},
				To: window.End * 1000,
			},
			Type: datadogV2.TIMESERIESFORMULAREQUESTTYPE_TIMESERIES_REQUEST,
		},
	}
	configuration := datadog.NewConfiguration()
	configuration.SetUnstableOperationEnabled("v2.QueryTimeseriesData", true)
	apiClient := datadog.NewAPIClient(configuration)
	apiClient.Cfg.Host = s.datadogAddress
	api := datadogV2.NewMetricsApi(apiClient)
	resp, r, err := api.QueryTimeseriesData(ctx, body)

	if err != nil {
		return make(map[string][]watcher.Metric), fmt.Errorf("Error when calling `MetricsApi.QueryTimeseriesData`: %v\n", err)
	}
	if r == nil {
		return make(map[string][]watcher.Metric), errors.New("No response from getting matrix in Datadog API")
	}
	if r.StatusCode != http.StatusOK {
		return make(map[string][]watcher.Metric), fmt.Errorf("received status code %v for metric resp", r.StatusCode)
	}

	responseContent, _ := json.MarshalIndent(resp, "", "  ")
	log.Debugf("Response from MetricsApi.QueryTimeseriesData:\n%s\n", responseContent)

	return getMetricsFromTimeSeriesResponse(resp)
}

// This method parses the datadogV2 time series response and return a map with key hostname, value an array of watcher.Metric
func getMetricsFromTimeSeriesResponse(resp datadogV2.TimeseriesFormulaQueryResponse) (map[string][]watcher.Metric, error) {
	metrics := make(map[string][]watcher.Metric)
	timeSeriesData, ok := resp.GetDataOk()
	if !ok {
		return metrics, errors.New("Error when getting data from timeseries response.")
	}
	if !timeSeriesData.HasAttributes() {
		return metrics, errors.New("No metrics found from timeseries response.")
	}
	log.Debugf("Response from TimeseriesData:\n%v\n", timeSeriesData)
	timeSeriesDataAttr, ok := timeSeriesData.GetAttributesOk()
	if !ok {
		return metrics, errors.New("Error when getting attributes from timeseries data.")
	}
	timeSeriesDataSeriesPtr, ok := timeSeriesDataAttr.GetSeriesOk()
	if !ok {
		return metrics, errors.New("Error when getting series from timeseries attributes.")
	}
	if timeSeriesDataSeriesPtr == nil || len(*timeSeriesDataSeriesPtr) == 0 {
		return metrics, errors.New("No series from timeseries attributes.")
	}
	timeSeriesDataValuesPtr, ok := timeSeriesDataAttr.GetValuesOk()
	if !ok {
		return metrics, errors.New("Error when getting values from timeseries attributes.")
	}
	if timeSeriesDataValuesPtr == nil || len(*timeSeriesDataValuesPtr) == 0 {
		return metrics, errors.New("No values from timeseries attributes.")
	}

	hosts := make([]string, len(*timeSeriesDataSeriesPtr))
	isCPU := make([]bool, len(*timeSeriesDataSeriesPtr))
	index := 0
	// Populate host array and corresponding query index
	for _, timeSeriesDataSeries := range *timeSeriesDataSeriesPtr {
		queryIndex, ok := timeSeriesDataSeries.GetQueryIndexOk()
		if !ok {
			log.Error("Error when getting query index from timeseries series.")
			continue
		}
		if queryIndex == nil {
			log.Error("No query index from timeseries series.")
			continue
		}
		isCPU[index] = (*queryIndex == 0)
		groupTagsPtr, ok := timeSeriesDataSeries.GetGroupTagsOk()
		if !ok {
			log.Error("Error when getting group tags from timeseries series.")
			continue
		}
		if groupTagsPtr == nil || len(*groupTagsPtr) == 0 {
			log.Error("No group tags from timeseries series.")
			continue
		}
		for _, groupTags := range *groupTagsPtr {
			log.Debugf("%v\n", groupTags)
			hosts[index] = getHostName(groupTags)
			index++
		}
	}

	if len(hosts) != len(*timeSeriesDataValuesPtr) {
		errMsg := "Number of group tags does not match number of values in timeseries series."
		log.Error(errMsg)
		return metrics, errors.New(errMsg)
	}
	// Find the average across returned values per 1 minute resolution
	// Build a hostname map of array of metrics [CPU, memory]
	hostIndex := 0
	for _, timesSeriesDataValues := range *timeSeriesDataValuesPtr {
		sum := 0.0
		count := 0.0
		for _, timesSeriesDataValue := range timesSeriesDataValues {
			if timesSeriesDataValue != nil {
				sum += *timesSeriesDataValue
				count += 1
			}
		}
		fetchedMetric := watcher.Metric{Value: sum / count}
		addDatadogMetadata(&fetchedMetric, isCPU[hostIndex])
		metrics[hosts[hostIndex]] = append(metrics[hosts[hostIndex]], fetchedMetric)
		hostIndex++
	}
	return metrics, nil
}

// This function checks and extracts node name from its FQDN if present
// hostname is in the format of host:<hostname>, e.g. host:alpha.dev.k8s.com
// It assumes that node names themselves don't contain "."
// Example: alpha.dev.k8s.com is returned as alpha
func getHostName(hostname string) string {
	index := strings.Index(hostname, ":")
	if index != -1 {
		hostname = hostname[index+1:]
		index = strings.Index(hostname, ".")
		if index != -1 {
			return hostname[:index]
		}
	}
	return hostname
}

/*
*
Sample input:

{
  "data": {
    "type": "timeseries_request",
    "attributes": {
      "to": 1728623905000,
      "from": 1728623005000,
      "queries": [
        {
          "name": "query1",
          "data_source": "metrics",
          "query": "max:cpu.utilization{host:<host>, cluster_name:<cluster_name>} by {host}.rollup(max, 60) "
        },
        {
          "name": "query2",
          "data_source": "metrics",
          "query": "max:memory.utilization{host:<host>, cluster_name:<cluster_name>} by {host}.rollup(max, 60)"
        }
      ],
      "interval": 900
    }
  }
}
*/

/*
*
Sample metricData output:

{
    "data": {
        "id": "0",
        "type": "timeseries_response",
        "attributes": {
            "series": [
                {
                    "group_tags": [
                        "host:<host>"
                    ],
                    "query_index": 0,
                    "unit": null
                },
                {
                    "group_tags": [
                        "host:<host>>"
                    ],
                    "query_index": 1,
                    "unit": null
                }
            ],
            "times": [
                1728623040000,
                1728623100000,
                1728623160000,
                1728623220000,
                1728623280000,
                1728623340000,
                1728623460000,
                1728623520000,
                1728623580000,
                1728623640000,
                1728623700000,
                1728623760000,
                1728623820000,
                1728623880000
            ],
            "values": [
                [
                    5.830664,
                    5.848869,
                    5.923058,
                    5.425694,
                    6.387427,
                    7.290057,
                    5.766315,
                    7.192848,
                    5.513073,
                    null,
                    5.528908,
                    5.296647,
                    5.937974,
                    5.272351
                ],
                [
                    30.498568,
                    30.460526,
                    30.457855,
                    30.474593,
                    30.470759,
                    30.476932,
                    30.489024,
                    30.494981,
                    30.51441,
                    30.520453,
                    30.512126,
                    30.516842,
                    30.493316,
                    30.49882
                ]
            ]
        }
    }
}
*/
