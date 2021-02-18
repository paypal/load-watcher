package client

import (
	"github.com/paypal/load-watcher/pkg/watcher"
)

type Client interface {
	// Return the client name
	Name() string
	// Get the recent metrics from all types of monitoring stack
	GetRecentMetrics() (watcher.WatcherMetrics, error)
}

