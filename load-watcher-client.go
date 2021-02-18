package main

import (
	"log"

	"github.com/paypal/load-watcher/pkg/utils"
)

func main() {
	metricProviderType := "load-watcher"		// Suppported options: k8s, prometheus, signalfx, load-watcher
	metricEndpoint := "http://localhost:2020"


	cl, err := utils.NewClient(metricProviderType, metricEndpoint, "")

	if err != nil {
		log.Fatalf("unable to create new client: %v", err)
	}


	watchermetrics, err := cl.GetRecentMetrics()

	if err != nil {
		log.Fatalf("unable to fetch data from client: %v", err)
	}

	log.Printf("Timestamp: %v", watchermetrics.Timestamp)
	log.Printf("Source: %v", watchermetrics.Source)
	log.Printf("Window, start: %v", watchermetrics.Window.Start)
	log.Printf("Window, end: %v", watchermetrics.Window.End)
	log.Printf("Window, duration: %v", watchermetrics.Window.Duration)

	for k, v := range watchermetrics.Data.NodeMetricsMap {
		log.Printf("Host: %v", k)
		for _, metric := range v.Metrics {
			log.Printf("%+v\n", metric)
		}
	}
}
