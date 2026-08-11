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

package watcher

import (
	"os"
	"strings"
)

const (
	K8sClientName      = "KubernetesMetricsServer"
	PromClientName     = "Prometheus"
	SignalFxClientName = "SignalFx"
	DatadogClientName  = "Datadog"

	MetricsProviderNameKey    = "METRICS_PROVIDER_NAME"
	MetricsProviderAddressKey = "METRICS_PROVIDER_ADDRESS"
	MetricsProviderTokenKey   = "METRICS_PROVIDER_TOKEN"
	MetricsProviderAppKey     = "METRICS_PROVIDER_APP_KEY"
	InsecureSkipVerify        = "INSECURE_SKIP_VERIFY"
	MetricsProviderHeadersKey = "METRICS_PROVIDER_HEADERS"
)

var (
	EnvMetricProviderOpts MetricsProviderOpts
)

// parseHeadersFromEnv parses headers from environment variable
// Expected format: "key1=value1,key2=value2,key3=value3"
// Example: "http_x_oauth_bypass_token=token123,X-API-Key=key456"
func parseHeadersFromEnv(headersEnv string) map[string]string {
	headers := make(map[string]string)
	if headersEnv == "" {
		return headers
	}

	// Split by comma to get individual header pairs
	headerPairs := strings.Split(headersEnv, ",")
	for _, pair := range headerPairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// Split by = to get key and value
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				headers[key] = value
			}
		}
	}

	return headers
}

func init() {
	var ok bool
	EnvMetricProviderOpts.Name, ok = os.LookupEnv(MetricsProviderNameKey)
	if !ok {
		EnvMetricProviderOpts.Name = K8sClientName
	}
	EnvMetricProviderOpts.Address, ok = os.LookupEnv(MetricsProviderAddressKey)
	EnvMetricProviderOpts.AuthToken, ok = os.LookupEnv(MetricsProviderTokenKey)
	EnvMetricProviderOpts.ApplicationKey, ok = os.LookupEnv(MetricsProviderAppKey)
	insecureVerify, _ := os.LookupEnv(InsecureSkipVerify)
	if strings.ToLower(insecureVerify) == "true" {
		EnvMetricProviderOpts.InsecureSkipVerify = true
	} else {
		EnvMetricProviderOpts.InsecureSkipVerify = false
	}

	// Parse headers from environment variable
	headersEnv, _ := os.LookupEnv(MetricsProviderHeadersKey)
	EnvMetricProviderOpts.Headers = parseHeadersFromEnv(headersEnv)
}

// Interface to be implemented by any metrics provider client to interact with Watcher
type MetricsProviderClient interface {
	// Return the client name
	Name() string
	// Fetch metrics for given host
	FetchHostMetrics(host string, window *Window) ([]Metric, error)
	// Fetch metrics for all hosts
	FetchAllHostsMetrics(window *Window) (map[string][]Metric, error)
	// Get metric provider server health status
	// Returns 0 if healthy, -1 if unhealthy along with error if any
	Health() (int, error)
}

// Generic metrics provider options
type MetricsProviderOpts struct {
	Name               string
	Address            string
	AuthToken          string
	ApplicationKey     string
	InsecureSkipVerify bool
	Headers            map[string]string
}
