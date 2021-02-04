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

package main

import (
	"log"

	"github.com/paypal/load-watcher/pkg/metricsprovider"
	"github.com/paypal/load-watcher/pkg/watcher"
)

func main() {
	// client, err := metricsprovider.NewMetricsServerClient()
	client, err := metricsprovider.NewPromClient()
	if err != nil {
		log.Fatalf("unable to create new client: %v", err)
	}
	w := watcher.NewWatcher(client)
	w.StartWatching()
	select {}
}
