/*
Copyright 2020 PayPal

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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/paypal/load-watcher/pkg/watcher"
	log "github.com/sirupsen/logrus"
)

const (
	// SignalFX Request Params
	DefaultSignalFxAddress = "https://api.signalfx.com"
	signalFxMetricsAPI     = "/v1/timeserieswindow"
	// SignalFx adds a suffix to hostnames if configured
	signalFxHostNameSuffix = ".group.region.gcp.com"
	signalFxHostFilter     = "host:"

	// SignalFX Query Params
	oneMinuteResolutionMs   = 60000
	cpuUtilizationMetric    = `sf_metric:"cpu.utilization"`
	memoryUtilizationMetric = `sf_metric:"memory.utilization"`
	AND                     = "AND"

	// Miscellaneous
	httpClientTimeout = 55 * time.Second
)

type signalFxClient struct {
	client          http.Client
	authToken       string
	signalFxAddress string
}

func NewSignalFxClient(opts watcher.MetricsProviderOpts) (watcher.MetricsProviderClient, error) {
	if opts.Name != watcher.SignalFxClientName {
		return nil, fmt.Errorf("metric provider name should be %v, found %v", watcher.SignalFxClientName, opts.Name)
	}
	tlsConfig := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // TODO(aqadeer): Figure out a secure way to let users add SSL certs
	}

	var signalFxAddress, signalFxAuthToken = DefaultSignalFxAddress, ""
	if opts.Address != "" {
		signalFxAddress = opts.Address
	}
	if opts.AuthToken != "" {
		signalFxAuthToken = opts.AuthToken
	}
	if signalFxAuthToken == "" {
		log.Fatalf("No auth token found to connect with SignalFx server")
	}
	return signalFxClient{client: http.Client{
		Timeout:   httpClientTimeout,
		Transport: tlsConfig},
		authToken:       signalFxAuthToken,
		signalFxAddress: signalFxAddress}, nil
}

func (s signalFxClient) Name() string {
	return watcher.SignalFxClientName
}

func (s signalFxClient) FetchHostMetrics(host string, window *watcher.Window) ([]watcher.Metric, error) {
	log.Debugf("fetching metrics for host %v", host)
	var metrics []watcher.Metric
	hostQuery := signalFxHostFilter + host + signalFxHostNameSuffix

	for _, metric := range []string{cpuUtilizationMetric, memoryUtilizationMetric} {
		uri, err := s.buildMetricURL(hostQuery, metric, window)
		if err != nil {
			return metrics, err
		}
		req, _ := http.NewRequest(http.MethodGet, uri.String(), nil)
		req.Header.Set("X-SF-Token", s.authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := s.client.Do(req)
		if err != nil {
			return metrics, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return metrics, fmt.Errorf("received status code: %v", resp.StatusCode)
		}
		var res interface{}
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return metrics, err
		}

		var fetchedMetric watcher.Metric
		// Added default operator and rollup for signalfx client.
		fetchedMetric.Operator = watcher.Average
		fetchedMetric.Rollup = window.Duration
		if metric == cpuUtilizationMetric {
			fetchedMetric.Name = cpuUtilizationMetric
			fetchedMetric.Type = watcher.CPU
		} else {
			fetchedMetric.Name = memoryUtilizationMetric
			fetchedMetric.Type = watcher.Memory
		}
		fetchedMetric.Value, err = decodeMetricsPayload(res)
		if err != nil {
			return metrics, err
		}
		metrics = append(metrics, fetchedMetric)
	}

	return metrics, nil
}

// TODO(aqadeer): Fetching metrics for all hosts is not possible currently via timeserieswindow SignalFx API
func (s signalFxClient) FetchAllHostsMetrics(*watcher.Window) (map[string][]watcher.Metric, error) {
	return nil, errors.New("This function is not yet implemented")
}

func (s signalFxClient) buildMetricURL(host string, metric string, window *watcher.Window) (uri *url.URL, err error) {
	uri, err = url.Parse(s.signalFxAddress + signalFxMetricsAPI)
	if err != nil {
		return nil, err
	}
	q := uri.Query()

	builder := strings.Builder{}
	builder.WriteString(host)
	builder.WriteString(fmt.Sprintf(" %v ", AND))
	builder.WriteString(metric)
	q.Set("query", builder.String())

	q.Set("startMs", strconv.FormatInt(window.Start, 10))
	q.Set("endMs", strconv.FormatInt(window.End, 10))
	q.Set("resolution", strconv.Itoa(oneMinuteResolutionMs))
	uri.RawQuery = q.Encode()
	return
}

/**
Sample payload:
{
  "data": {
    "Ehql_bxBgAc": [
      [
        1600213380000,
        84.64246793530153
      ]
    ]
  },
  "errors": []
}
*/
func decodeMetricsPayload(payload interface{}) (float64, error) {
	var data interface{}
	data = payload.(map[string]interface{})["data"]
	if data == nil {
		return -1, errors.New("unexpected payload: missing data field")
	}
	keyMap, ok := data.(map[string]interface{})
	if !ok {
		return -1, errors.New("unable to deserialise data field")
	}

	var values []interface{}
	if len(keyMap) == 0 {
		return -1, errors.New("no values found")
	}
	for _, v := range keyMap {
		values, ok = v.([]interface{})
		if !ok {
			return -1, errors.New("unable to deserialise values")
		}
		break
	}
	if len(values) == 0 {
		return -1, errors.New("no metric value array could be decoded")
	}

	var timestampUtilisation []interface{}
	// Choose the latest window out of multiple values returned
	timestampUtilisation, ok = values[len(values)-1].([]interface{})
	if !ok {
		return -1, errors.New("unable to deserialise metric values")
	}
	return timestampUtilisation[1].(float64), nil
}
