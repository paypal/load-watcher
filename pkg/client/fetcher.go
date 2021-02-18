package client

import (
	"github.com/paypal/load-watcher/pkg/metricsprovider"
	"github.com/paypal/load-watcher/pkg/watcher"
)

type fetcherClient struct {
	Fetcher watcher.FetcherClient
}

func (c fetcherClient) Name() string {
	return c.Fetcher.Name()
}

func NewFetcherClient(metricProviderType string, metricEndpoint string, metricAuthToken string) (fetcherClient, error) {
	var fetcher watcher.FetcherClient
	var err error

	switch metricProviderType {
	case metricsprovider.PromClientName:
		fetcher, err = metricsprovider.NewPromClient(metricEndpoint, metricAuthToken)
	case metricsprovider.SignalFxClientName:
		fetcher, err = metricsprovider.NewSignalFxClient(metricEndpoint)
	default:
		fetcher, err = metricsprovider.NewMetricsServerClient()
	}

	return fetcherClient{fetcher}, err
}

func (c fetcherClient) GetRecentMetrics() (watcher.WatcherMetrics, error) {
	window := watcher.CurrentFifteenMinuteWindow()
	hostmetrics, err := c.Fetcher.FetchAllHostsMetrics(window)

	if err != nil {
		return watcher.WatcherMetrics{}, err
	}

	watchermetrics := watcher.MetricListMap2NodeMetricMap(hostmetrics, c.Name(), *window)
	return watchermetrics, err
}