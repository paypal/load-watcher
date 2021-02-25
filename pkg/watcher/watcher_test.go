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

package watcher

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/francoispqt/gojay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var w *Watcher

func TestGetLatestWatcherMetrics(t *testing.T) {
	var metrics *WatcherMetrics
	metrics, err := w.GetLatestWatcherMetrics(FifteenMinutes)
	require.Nil(t, err)
	assert.Equal(t, FifteenMinutesMetricsMap[FirstNode], metrics.Data.NodeMetricsMap[FirstNode].Metrics)
	assert.Equal(t, FifteenMinutesMetricsMap[SecondNode], metrics.Data.NodeMetricsMap[SecondNode].Metrics)

	metrics, err = w.GetLatestWatcherMetrics(TenMinutes)
	require.Nil(t, err)
	assert.Equal(t, TenMinutesMetricsMap[FirstNode], metrics.Data.NodeMetricsMap[FirstNode].Metrics)
	assert.Equal(t, TenMinutesMetricsMap[SecondNode], metrics.Data.NodeMetricsMap[SecondNode].Metrics)

	metrics, err = w.GetLatestWatcherMetrics(FiveMinutes)
	require.Nil(t, err)
	assert.Equal(t, FiveMinutesMetricsMap[FirstNode], metrics.Data.NodeMetricsMap[FirstNode].Metrics)
	assert.Equal(t, FiveMinutesMetricsMap[SecondNode], metrics.Data.NodeMetricsMap[SecondNode].Metrics)
}

func TestWatcherAPIAllHosts(t *testing.T) {
	req, err := http.NewRequest("GET", BaseUrl, nil)
	require.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(w.handler)

	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	expectedMetrics, err := w.GetLatestWatcherMetrics(FifteenMinutes)
	require.Nil(t, err)
	data := Data{NodeMetricsMap: make(map[string]NodeMetrics)}
	var watcherMetrics = &WatcherMetrics{Data: data}
	err = gojay.UnmarshalJSONObject(rr.Body.Bytes(), watcherMetrics)
	require.Nil(t, err)
	assert.Equal(t, expectedMetrics, watcherMetrics)
}

func TestWatcherAPISingleHost(t *testing.T) {
	uri, _ := url.Parse(BaseUrl)
	q := uri.Query()
	q.Set("host", FirstNode)
	uri.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", uri.String(), nil)
	require.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(w.handler)

	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	expectedMetrics, err := w.GetLatestWatcherMetrics(FifteenMinutes)
	require.Nil(t, err)
	data := Data{NodeMetricsMap: make(map[string]NodeMetrics)}
	var watcherMetrics = &WatcherMetrics{Data: data}
	err = gojay.UnmarshalJSONObject(rr.Body.Bytes(), watcherMetrics)
	require.Nil(t, err)
	assert.Equal(t, expectedMetrics.Data.NodeMetricsMap[FirstNode], watcherMetrics.Data.NodeMetricsMap[FirstNode])
	assert.Equal(t, expectedMetrics.Source, watcherMetrics.Source)
}

func TestWatcherMetricsNotFound(t *testing.T) {
	uri, _ := url.Parse(BaseUrl)
	q := uri.Query()
	q.Set("host", "deadbeef")
	uri.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", uri.String(), nil)
	require.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(w.handler)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestWatcherInternalServerError(t *testing.T) {
	client := NewTestMetricsServerClient()
	unstartedWatcher := NewWatcher(client)

	req, err := http.NewRequest("GET", BaseUrl, nil)
	require.Nil(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(unstartedWatcher.handler)

	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestMain(m *testing.M) {
	client := NewTestMetricsServerClient()
	w = NewWatcher(client)
	w.StartWatching()

	ret := m.Run()
	os.Exit(ret)
}
