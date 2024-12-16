package metricsprovider

import (
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/paypal/load-watcher/pkg/watcher"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewDatadogClient(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken = "Test"
	opts.ApplicationKey = "Test"

	_, err := NewDatadogClient(opts)
	assert.Nil(t, err)

	opts.Name = "invalid"
	_, err = NewDatadogClient(opts)
	assert.NotNil(t, err)
}

// Sample metricData for 2 hosts of cpu and memory util
func TestDDFetchAllHostMetrics(t *testing.T) {
	metricData := `{
    "data": {
        "id": "0",
        "type": "timeseries_response",
        "attributes": {
            "series": [
                {
                    "group_tags": [
                        "host:test1"
                    ],
                    "query_index": 0,
                    "unit": null
                },
                {
                    "group_tags": [
                        "host:test1"
                    ],
                    "query_index": 1,
                    "unit": null
                },
				{
                    "group_tags": [
                        "host:test2"
                    ],
                    "query_index": 0,
                    "unit": null
                },
                {
                    "group_tags": [
                        "host:test2"
                    ],
                    "query_index": 1,
                    "unit": null
                }
            ],
            "times": [
                1724967300000,
                1724967360000,
                1724967420000,
                1724967480000,
                1724967540000,
                1724967600000,
                1724967660000,
                1724967720000,
                1724967780000,
                1724967840000,
                1724967900000,
                1724967960000,
                1724968020000,
                1724968080000,
                1724968140000,
                1724968200000
            ],
            "values": [
                [
                    7.332664,
                    10.399366,
                    11.780529,
                    10.082532,
                    8.429297,
                    7.017103,
                    12.490895,
                    7.128327,
                    7.08206,
                    5.555416,
                    9.067136,
                    9.532126,
                    11.440966,
                    10.396935,
                    8.71661,
                    6.193735
                ],
                [
                    50.949683,
                    50.950399,
                    50.946066,
                    50.950076,
                    50.934947,
                    50.941883,
                    50.968234,
                    50.984515,
                    51.007935,
                    51.011688,
                    51.01305,
                    51.022667,
                    51.038384,
                    51.044218,
                    51.04624,
                    51.04946
                ],
				[
                    6.332664,
                    11.399366,
                    12.780529,
                    13.082532,
                    18.429297,
                    17.017103,
                    22.490895,
                    17.128327,
                    17.08206,
                    15.555416,
                    19.067136,
                    19.532126,
                    21.440966,
                    20.396935,
                    18.71661,
                    16.193735
                ],
                [
                    40.949683,
                    40.950399,
                    40.946066,
                    40.950076,
                    40.934947,
                    40.941883,
                    40.968234,
                    40.984515,
                    41.007935,
                    41.011688,
                    41.01305,
                    41.022667,
                    41.038384,
                    41.044218,
                    41.04624,
                    41.04946
                ]
            ]
        }
    }
}`

	bytes := []byte(metricData)
	resp := datadogV2.NewTimeseriesFormulaQueryResponseWithDefaults()
	assert.NotNil(t, resp)
	err := resp.UnmarshalJSON(bytes)
	assert.Nil(t, err)

	metrics, err1 := getMetricsFromTimeSeriesResponse(*resp)

	assert.Nil(t, err1)
	assert.NotNil(t, metrics)
	assert.Equal(t, len(metrics), 2)
	assert.NotNil(t, metrics["test1"])
	assert.NotNil(t, metrics["test2"])
}

// Sample metricData for 1 host of cpu and memory util
func TestDDFetchHostMetrics(t *testing.T) {
	metricData := `{
    "data": {
        "id": "0",
        "type": "timeseries_response",
        "attributes": {
            "series": [
                {
                    "group_tags": [
                        "host:test1"
                    ],
                    "query_index": 0,
                    "unit": null
                },
                {
                    "group_tags": [
                        "host:test1"
                    ],
                    "query_index": 1,
                    "unit": null
                }
            ],
            "times": [
                1724967300000,
                1724967360000,
                1724967420000,
                1724967480000,
                1724967540000,
                1724967600000,
                1724967660000,
                1724967720000,
                1724967780000,
                1724967840000,
                1724967900000,
                1724967960000,
                1724968020000,
                1724968080000,
                1724968140000,
                1724968200000
            ],
            "values": [
                [
                    7.332664,
                    10.399366,
                    11.780529,
                    10.082532,
                    8.429297,
                    7.017103,
                    12.490895,
                    7.128327,
                    7.08206,
                    5.555416,
                    9.067136,
                    9.532126,
                    11.440966,
                    10.396935,
                    8.71661,
                    6.193735
                ],
                [
                    50.949683,
                    50.950399,
                    50.946066,
                    50.950076,
                    50.934947,
                    50.941883,
                    50.968234,
                    50.984515,
                    51.007935,
                    51.011688,
                    51.01305,
                    51.022667,
                    51.038384,
                    51.044218,
                    51.04624,
                    51.04946
                ]
            ]
        }
    }
}`

	bytes := []byte(metricData)
	resp := datadogV2.NewTimeseriesFormulaQueryResponseWithDefaults()
	assert.NotNil(t, resp)
	err := resp.UnmarshalJSON(bytes)
	assert.Nil(t, err)

	metrics, err1 := getMetricsFromTimeSeriesResponse(*resp)

	assert.Nil(t, err1)
	assert.NotNil(t, metrics)
	assert.Equal(t, len(metrics), 1)
	assert.NotNil(t, metrics["test1"])
}

func TestDDHealth(t *testing.T) {
	opts := watcher.MetricsProviderOpts{
		Name:    watcher.DatadogClientName,
		Address: "",
	}
	opts.AuthToken = "Test"
	opts.ApplicationKey = "Test"

	client, err := NewDatadogClient(opts)
	assert.Nil(t, err)
	ret, err := client.Health()
	assert.Nil(t, err)
	assert.Equal(t, ret, 0)
}
