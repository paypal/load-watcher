package metricsprovider

import (
	"github.com/paypal/load-watcher/pkg/watcher"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// Below environment variables need to be set in order to run the unit test
// DATADOG_CLUSTER_NAME
// METRICS_PROVIDER_ADDRESS defaults to datadoghq.com
// METRICS_PROVIDER_APP_KEY
// METRICS_PROVIDER_TOKEN
// host

func TestNewDatadogClient(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken, _ = os.LookupEnv(watcher.MetricsProviderTokenKey)
	opts.ApplicationKey, _ = os.LookupEnv(watcher.MetricsProviderAppKey)

	_, err := NewDatadogClient(opts)
	assert.Nil(t, err)

	opts.Name = "invalid"
	_, err = NewDatadogClient(opts)
	assert.NotNil(t, err)
}

func TestDDFetchAllHostMetrics(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken, _ = os.LookupEnv(watcher.MetricsProviderTokenKey)
	opts.ApplicationKey, _ = os.LookupEnv(watcher.MetricsProviderAppKey)

	client, err := NewDatadogClient(opts)
	assert.Nil(t, err)

	metrics, err := client.FetchAllHostsMetrics(watcher.CurrentFifteenMinuteWindow())
	assert.Nil(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, len(metrics) > 0)
}

func TestDDFetchHostMetrics(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken, _ = os.LookupEnv(watcher.MetricsProviderTokenKey)
	opts.ApplicationKey, _ = os.LookupEnv(watcher.MetricsProviderAppKey)

	client, err := NewDatadogClient(opts)
	assert.Nil(t, err)

	host, _ := os.LookupEnv("host")
	metrics, err := client.FetchHostMetrics(host, watcher.CurrentFifteenMinuteWindow())
	assert.Nil(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, len(metrics) == 1)

	metrics, err = client.FetchHostMetrics("Invalid", watcher.CurrentFifteenMinuteWindow())
	assert.NotNil(t, err)
}

func TestDDHealth(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken, _ = os.LookupEnv(watcher.MetricsProviderTokenKey)
	opts.ApplicationKey, _ = os.LookupEnv(watcher.MetricsProviderAppKey)

	client, err := NewDatadogClient(opts)
	assert.Nil(t, err)
	ret, err := client.Health()
	assert.Nil(t, err)
	assert.Equal(t, ret, 0)
}
