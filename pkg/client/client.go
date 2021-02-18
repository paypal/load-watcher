package client

import (
	"github.com/paypal/load-watcher/pkg/watcher"
)


type Client interface {
	// Return the client name
	Name() string
	// Fetch metrics for given host
	GetRecentMetrics() (watcher.WatcherMetrics, error)
}

