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

/*
	Package Watcher is responsible for watching latest metrics from metrics provider via a fetcher client.
	It exposes an HTTP REST endpoint to get these metrics, in addition to application API
	This also uses a fast json parser
*/
package watcher

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/francoispqt/gojay"
	log "github.com/sirupsen/logrus"

	statistics "github.com/grd/stat"
)

const (
	targetLoad             = "/watcher"
	variationRiskBalancing = "/variation"
	FifteenMinutes         = "15m"
	TenMinutes             = "10m"
	FiveMinutes            = "5m"
	CPU                    = "CPU"
	Memory                 = "Memory"
	AVG                    = "AVG"
	STD                    = "STD"
	minute                 = 15
	cacheSize              = 15
)

type Watcher struct {
	mutex       sync.RWMutex // For thread safe access to cache
	minuteQueue []WatcherMetrics
	cacheSize   int
	client      FetcherClient
	isStarted   bool // Indicates if the Watcher is started by calling StartWatching()
	shutdown    chan os.Signal
}

type Window struct {
	Duration string `json:"duration"`
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
}

type Metric struct {
	Name   string  `json:"name"`             // Name of metric at the provider
	Type   string  `json:"type"`             // CPU or Memory
	Rollup string  `json:"rollup,omitempty"` // Rollup used for metric calculation
	Value  float64 `json:"value"`            // Value is expected to be in %
}

type NodeMetricsMap map[string]NodeMetrics

type Data struct {
	NodeMetricsMap NodeMetricsMap
}

type WatcherMetrics struct {
	Timestamp int64  `json:"timestamp"`
	Window    Window `json:"window"`
	Source    string `json:"source"`
	Data      Data   `json:"data"`
}

type Tags struct {
}

type Metadata struct {
	DataCenter string `json:"dataCenter,omitempty"`
}

type NodeMetrics struct {
	Metrics  []Metric `json:"metrics,omitempty"`
	Tags     Tags     `json:"tags,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// Returns a new initialised Watcher
func NewWatcher(client FetcherClient) *Watcher {
	return &Watcher{
		mutex:       sync.RWMutex{},
		minuteQueue: make([]WatcherMetrics, 0, minute),
		cacheSize:   cacheSize,
		client:      client,
		shutdown:    make(chan os.Signal, 1),
	}
}

// This function needs to be called to begin actual watching
func (w *Watcher) StartWatching() {
	w.mutex.RLock()
	if w.isStarted {
		w.mutex.RUnlock()
		return
	}
	w.mutex.RUnlock()

	go func() {
		for {
			metric := &w.minuteQueue
			fetchedMetricsMap, err := w.client.FetchAllHostsMetrics()
			if err != nil {
				log.Errorf("received error while fetching metrics: %v ", err)
				continue
			}
			metricsMap := make(map[string]NodeMetrics)
			for host, fetchedMetrics := range fetchedMetricsMap {
				nodeMetric := NodeMetrics{
					Metrics: make([]Metric, len(fetchedMetrics)),
				}
				copy(nodeMetric.Metrics, fetchedMetrics)
				metricsMap[host] = nodeMetric
			}
			watcherMetrics := &WatcherMetrics{Timestamp: time.Now().Unix(),
				Data: Data{NodeMetricsMap: metricsMap},
			}
			w.appendWatcherMetrics(metric, watcherMetrics)
			time.Sleep(time.Minute) // This is assuming fetching of metrics won't exceed more than 1 minute. If it happens we need to throttle rate of fetches
		}
	}()

	http.HandleFunc(targetLoad, w.targetLoadHandler)
	http.HandleFunc(variationRiskBalancing, w.loadVariationRiskBalancingHandler)
	server := &http.Server{
		Addr:    ":2020",
		Handler: http.DefaultServeMux,
	}

	go func() {
		log.Warn(server.ListenAndServe())
	}()

	signal.Notify(w.shutdown, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-w.shutdown
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Unable to shutdown server: %v", err)
		}
	}()

	w.mutex.Lock()
	w.isStarted = true
	w.mutex.Unlock()
}

// Returns latest metrics present in load Watcher cache. StartWatching() should be called before calling this.
// It starts from 15 minute window, and falls back to 10 min, 5 min windows subsequently if metrics are not present
func (w *Watcher) GetLatestWatcherMetrics() (*WatcherMetrics, error) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	if !w.isStarted {
		return nil, errors.New("need to call StartWatching() first!")
	}
	if len(w.minuteQueue) == 0 {
		return nil, errors.New("unable to get any latest metrics")
	}
	return deepCopyWatcherMetrics(&w.minuteQueue[len(w.minuteQueue)-1]), nil
}

// GetLatestWatcherMetricsAnalysis get analysis
func (w *Watcher) GetLatestWatcherMetricsAnalysis(duration string) (*WatcherMetrics, error) {
	if !w.isStarted {
		return nil, errors.New("need to call StartWatching() first!")
	}
	window, wmData := w.getCurrentWindow(duration)
	analysisData, err := analysis(wmData)
	if err != nil {
		return nil, errors.New("analysis calculator Error")
	}

	return &WatcherMetrics{
		Timestamp: time.Now().Unix(),
		Window:    *window,
		Data:      *analysisData,
	}, nil

}

func analysis(src *[]WatcherMetrics) (*Data, error) {
	metricsMap := make(map[string]NodeMetrics)

	type resource struct {
		CPU    statistics.Float64Slice
		Memory statistics.Float64Slice
	}
	total := make(map[string]resource)

	for _, item := range *src {
		for host, fetchedMetric := range item.Data.NodeMetricsMap {
			rs, ok := total[host]
			if !ok {
				rs = resource{}
			}
			for _, mertics := range fetchedMetric.Metrics {
				switch mertics.Type {
				case CPU:
					rs.CPU = append(rs.CPU, mertics.Value)
				case Memory:
					rs.Memory = append(rs.Memory, mertics.Value)
				}
			}
			total[host] = rs
		}
	}

	for host, mertics := range total {
		cpuVariance := statistics.Variance(mertics.CPU)
		cpuMean := statistics.Mean(mertics.CPU)
		memoryVariance := statistics.Variance(mertics.Memory)
		memoryMean := statistics.Mean(mertics.Memory)
		nodeMetrics := NodeMetrics{
			Metrics: []Metric{
				{
					Type:   CPU,
					Rollup: "AVG",
					Value:  cpuMean,
					Name:   "host.cpu.utilisation",
				},
				{
					Type:   CPU,
					Rollup: "STD",
					Value:  cpuVariance,
					Name:   "host.cpu.utilisation",
				},
				{
					Type:   Memory,
					Rollup: "AVG",
					Value:  memoryMean,
					Name:   "host.memory.utilisation",
				},
				{
					Type:   Memory,
					Rollup: "STD",
					Value:  memoryVariance,
					Name:   "host.memory.utilisation",
				},
			},
		}
		metricsMap[host] = nodeMetrics
	}

	return &Data{
		NodeMetricsMap: metricsMap,
	}, nil
}

func (w *Watcher) getCurrentWindow(duration string) (*Window, *[]WatcherMetrics) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	var curWindow *Window
	var size uint
	var tmp []WatcherMetrics
	var watcherMetrics *[]WatcherMetrics

	switch {
	case duration == FifteenMinutes && len(w.minuteQueue) >= 15:
		curWindow = CurrentFifteenMinuteWindow()
		tmp = w.minuteQueue[len(w.minuteQueue)-15:]
		size = 15
	case (duration == FifteenMinutes || duration == TenMinutes) && len(w.minuteQueue) >= 10:
		curWindow = CurrentTenMinuteWindow()
		tmp = w.minuteQueue[len(w.minuteQueue)-10:]
		size = 10
	case (duration == FifteenMinutes || duration == TenMinutes || duration == FiveMinutes) && len(w.minuteQueue) >= 5:
		curWindow = CurrentFiveMinuteWindow()
		tmp = w.minuteQueue[len(w.minuteQueue)-5:]
		size = 5
	case len(w.minuteQueue) > 0:
		curWindow = CurrentFiveMinuteWindow()
		tmp = w.minuteQueue
		size = uint(len(w.minuteQueue))
	default:
		log.Error("received unexpected window duration, defaulting to 15m")
		curWindow = CurrentFifteenMinuteWindow()
		return curWindow, nil
	}
	watcherMetrics = deepCopyWatcherMetricsArray(&tmp, size)
	return curWindow, watcherMetrics
}

func (w *Watcher) appendWatcherMetrics(recentMetrics *[]WatcherMetrics, metric *WatcherMetrics) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if len(*recentMetrics) == w.cacheSize {
		*recentMetrics = (*recentMetrics)[1:]
	}
	*recentMetrics = append(*recentMetrics, *metric)
}

func deepCopyWatcherMetricsArray(src *[]WatcherMetrics, size uint) *[]WatcherMetrics {
	wMetrics := make([]WatcherMetrics, 0, size)

	for _, wm := range *src {
		wmItem := deepCopyWatcherMetrics(&wm)
		wMetrics = append(wMetrics, *wmItem)
	}

	return &wMetrics
}

func deepCopyWatcherMetrics(src *WatcherMetrics) *WatcherMetrics {
	nodeMetricsMap := make(map[string]NodeMetrics)
	for host, fetchedMetric := range src.Data.NodeMetricsMap {
		nodeMetric := NodeMetrics{
			Metrics: make([]Metric, len(fetchedMetric.Metrics)),
			Tags:    fetchedMetric.Tags,
		}
		copy(nodeMetric.Metrics, fetchedMetric.Metrics)
		nodeMetric.Metadata = fetchedMetric.Metadata
		nodeMetricsMap[host] = nodeMetric
	}
	return &WatcherMetrics{
		Timestamp: src.Timestamp,
		Window:    src.Window,
		Source:    src.Source,
		Data: Data{
			NodeMetricsMap: nodeMetricsMap,
		},
	}
}

// HTTP Handler for loadVariationRiskBalancing endpoint
func (w *Watcher) loadVariationRiskBalancingHandler(resp http.ResponseWriter, r *http.Request) {
	resp.Header().Set("Content-Type", "application/json")

	metrics, err := w.GetLatestWatcherMetricsAnalysis(FifteenMinutes)
	if metrics == nil {
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Error(err)
			return
		}
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	host := r.URL.Query().Get("host")
	var bytes []byte
	if host != "" {
		if _, ok := metrics.Data.NodeMetricsMap[host]; ok {
			hostMetricsData := make(map[string]NodeMetrics)
			hostMetricsData[host] = metrics.Data.NodeMetricsMap[host]
			hostMetrics := WatcherMetrics{Timestamp: metrics.Timestamp,
				Window: metrics.Window,
				Source: metrics.Source,
				Data:   Data{NodeMetricsMap: hostMetricsData},
			}
			bytes, err = gojay.MarshalJSONObject(&hostMetrics)
		} else {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		bytes, err = gojay.MarshalJSONObject(metrics)
	}

	if err != nil {
		log.Error(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.Error(err)
		resp.WriteHeader(http.StatusInternalServerError)
	}
}

// HTTP Handler for targetLoad endpoint
func (w *Watcher) targetLoadHandler(resp http.ResponseWriter, r *http.Request) {
	resp.Header().Set("Content-Type", "application/json")

	metrics, err := w.GetLatestWatcherMetrics()
	if metrics == nil {
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			log.Error(err)
			return
		}
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	host := r.URL.Query().Get("host")
	var bytes []byte
	if host != "" {
		if _, ok := metrics.Data.NodeMetricsMap[host]; ok {
			hostMetricsData := make(map[string]NodeMetrics)
			hostMetricsData[host] = metrics.Data.NodeMetricsMap[host]
			hostMetrics := WatcherMetrics{Timestamp: metrics.Timestamp,
				Window: metrics.Window,
				Source: metrics.Source,
				Data:   Data{NodeMetricsMap: hostMetricsData},
			}
			bytes, err = gojay.MarshalJSONObject(&hostMetrics)
		} else {
			resp.WriteHeader(http.StatusNotFound)
			return
		}
	} else {
		bytes, err = gojay.MarshalJSONObject(metrics)
	}

	if err != nil {
		log.Error(err)
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = resp.Write(bytes)
	if err != nil {
		log.Error(err)
		resp.WriteHeader(http.StatusInternalServerError)
	}
}

// Utility functions

func CurrentFifteenMinuteWindow() *Window {
	curTime := time.Now().Unix()
	return &Window{FifteenMinutes, curTime - 15*60, curTime}
}

func CurrentTenMinuteWindow() *Window {
	curTime := time.Now().Unix()
	return &Window{TenMinutes, curTime - 10*60, curTime}
}

func CurrentFiveMinuteWindow() *Window {
	curTime := time.Now().Unix()
	return &Window{FiveMinutes, curTime - 5*60, curTime}
}
