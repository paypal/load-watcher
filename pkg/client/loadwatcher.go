/*
Copyright 2020

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

package client

import (
	"crypto/tls"
	"github.com/francoispqt/gojay"
	"github.com/paypal/load-watcher/pkg/watcher"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

const (
	LoadWatcherClientName = "load-watcher"
	DefaultLoadWatcherClientEndpoint = "https://load-watcher.monitoring.svc.cluster.local:2020"
	watcherQuery = "/watcher"
	defaultRetries = 3
	loadwatcherClientTimeout = 55 * time.Second
)

var (
	loadWatcherEndpoint string
)

type loadWatcherClient struct {
	client http.Client
}

func NewLoadWatcherClient(loadwatcherurl string) (loadWatcherClient, error) {
	tlsConfig := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	loadWatcherEndpoint = loadwatcherurl

	return loadWatcherClient{client: http.Client{
		Timeout:   loadwatcherClientTimeout,
		Transport: tlsConfig}}, nil
}

func (s loadWatcherClient) Name() string {
	return LoadWatcherClientName
}


func (s loadWatcherClient) GetRecentMetrics() (watcher.WatcherMetrics, error) {
	queryURL := loadWatcherEndpoint + watcherQuery
	watchermetrics, err := s.queryMetrics(queryURL)
	return watchermetrics, err
}

func (s loadWatcherClient) queryMetrics(queryURL string) (watcher.WatcherMetrics, error) {
	metrics := watcher.WatcherMetrics{}
	req, err := http.NewRequest(http.MethodGet, queryURL, nil)
	if err != nil {
		return metrics, err
	}
	req.Header.Set("Content-Type", "application/json")

	var retries int = defaultRetries
	var resp *http.Response
	for retries > 0 {
		resp, err = s.client.Do(req)

		if err != nil {
			retries -= 1
		} else {
			break
		}
	}
	if err != nil {
		return metrics, err
	}

	defer resp.Body.Close()
	klog.V(6).Infof("received status code %v from watcher", resp.StatusCode)
	if resp.StatusCode == http.StatusOK {
		data := watcher.Data{NodeMetricsMap: make(map[string]watcher.NodeMetrics)}
		metrics = watcher.WatcherMetrics{Data: data}
		dec := gojay.BorrowDecoder(resp.Body)
		defer dec.Release()
		err = dec.Decode(&metrics)
		if err != nil {
			klog.Errorf("unable to decode watcher metrics: %v", err)
			return metrics, err
		}

		return metrics, nil
	} else {
		klog.Errorf("received status code %v from watcher", resp.StatusCode)
	}

	return metrics, nil
}
