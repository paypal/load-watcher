package main

import (
	"log"

	"github.com/paypal/load-watcher/pkg/metricsprovider"
	"github.com/paypal/load-watcher/pkg/watcher"
)

func main() {
	client, err := metricsprovider.NewMetricsServerClient()
	if err != nil {
		log.Fatalf("Unable to create new client: %v", err)
	}
	w := watcher.NewWatcher(client)
	w.StartWatching()
	select{}
}
