package metricsprovider

import (
	"testing"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/consul/sdk/testutil/retry"
	"github.com/paypal/load-watcher/pkg/watcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsulFetchHostMetrics(t *testing.T) {
	// Make client config
	conf := api.DefaultConfig()

	// Create server
	var server *testutil.TestServer
	var err error
	nodeName := "test-node"
	retry.RunWith(retry.ThreeTimes(), t, func(r *retry.R) {
		server, err = testutil.NewTestServerConfigT(t, func(c *testutil.TestServerConfig) {
			c.EnableScriptChecks = true
			c.NodeName = nodeName
		})
		if err != nil {
			r.Fatalf("Failed to start server: %v", err.Error())
		}
	})
	if server.Config.Bootstrap {
		server.WaitForLeader(t)
	}

	conf.Address = server.HTTPAddr

	fetcherClient, err := NewConsulClient(conf)
	c, err := api.NewClient(conf)
	if err != nil {
		server.Stop()
		t.Fatalf("err: %v", err)
	}
	defer server.Stop()
	agent := c.Agent()
	health := c.Health()

	// Make a service with a check
	reg := &api.AgentServiceRegistration{
		Name: "uptime",
		ID:   "metrics",
		Check: &api.AgentServiceCheck{
			Args:     []string{"echo", "-n", "0.1, 0.1, 0.1\n4"},
			Interval: "1ms",
		},
	}
	if err := agent.ServiceRegister(reg); err != nil {
		t.Fatalf("err: %v", err)
	}
	// Wait for Script to execute
	time.Sleep(time.Second)

	metrics, err := fetcherClient.FetchHostMetrics(nodeName, watcher.CurrentFifteenMinuteWindow())
	assert.True(t, len(metrics) > 0)
	assert.Equal(t, metrics[0].Value, 0.025)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	check := &api.HealthCheck{
		Node:        nodeName,
		CheckID:     "metrics",
		Name:        "Service 'uptime' check",
		ServiceID:   "uptime",
		Output:      "0.1, 0.1, 0.1\n4",
		ServiceName: "uptime",
		Type:        "script",
	}

	out, meta, err := health.Checks("uptime", nil)
	if err != nil {
		t.Fatal(err)
	}
	if meta.LastIndex == 0 {
		t.Fatalf("bad: %v", meta)
	}
	require.Equal(t, check.Output, out[0].Output)
}
