package utils

import (
	"os"

	"github.com/paypal/load-watcher/pkg/client"
	"github.com/paypal/load-watcher/pkg/metricsprovider"
)

const (
	MetricClientTypeKey		 = "METRIC_CLIENT"
	MetricEndpointKey    	 = "METRIC_HOST"
	MetricAuthTokenKey   	 = "METRIC_AUTH_TOKEN"
)

var (
	MetricProviderType     string
	MetricEndpoint         string
	MetricAuthToken        string
	metricProviderPresent  bool
	metricEndpointPresent  bool
	metricAuthTokenPresent bool

)

func InitEnvVars() {
	MetricProviderType, metricProviderPresent = os.LookupEnv(MetricClientTypeKey)
	if !metricProviderPresent {
		MetricProviderType = metricsprovider.K8sClientName
	}

	MetricEndpoint, metricEndpointPresent = os.LookupEnv(MetricEndpointKey)
	MetricAuthToken, metricAuthTokenPresent = os.LookupEnv(MetricAuthTokenKey)
	if !metricEndpointPresent {
		switch MetricProviderType {
		case metricsprovider.K8sClientName:
			MetricEndpoint = ""
		case metricsprovider.PromClientName:
			MetricEndpoint = metricsprovider.DefaultPrometheusEndpoint
		case metricsprovider.SignalFxClientName:
			MetricEndpoint = metricsprovider.DefaultSignalFxEndpoint
		case client.LoadWatcherClientName:
			MetricEndpoint = client.DefaultLoadWatcherClientEndpoint
		default:
			MetricEndpoint = ""
		}
	}

	if !metricAuthTokenPresent {
		MetricAuthToken = ""
	}
}

func NewClient(metricProviderType string, metricEndpoint string, metricAuthToken string) (client.Client, error) {
	if metricProviderType == client.LoadWatcherClientName {
		cl, err := client.NewLoadWatcherClient(metricEndpoint)
		return cl, err
	} else {
		cl, err := client.NewFetcherClient(metricProviderType, metricEndpoint, metricAuthToken)
		return cl, err
	}
}
