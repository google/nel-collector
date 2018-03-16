// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// Basic pipeline tests

type simulatedClock struct {
	currentTime time.Time
}

func newSimulatedClock() *simulatedClock {
	return &simulatedClock{currentTime: time.Unix(0, 0)}
}

func (c simulatedClock) Now() time.Time {
	return c.currentTime
}

func TestIgnoreNonPOST(t *testing.T) {
	pipeline := NewTestPipeline(newSimulatedClock())
	request := httptest.NewRequest("GET", "https://example.com/upload/", bytes.NewReader(testdata(t, "valid-nel-report.json")))
	request.Header.Add("Content-Type", "application/report")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("ServeHTTP(GET): got %d, wanted %d", response.Code, http.StatusMethodNotAllowed)
		return
	}
}

func TestIgnoreWrongContentType(t *testing.T) {
	pipeline := NewTestPipeline(newSimulatedClock())
	request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(testdata(t, "valid-nel-report.json")))
	request.Header.Add("Content-Type", "application/json")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)
	if response.Code != http.StatusBadRequest {
		t.Errorf("ServeHTTP(GET): got %d, wanted %d", response.Code, http.StatusBadRequest)
		return
	}
}

var dumpCases = []struct {
	name    string
	useIPv6 bool
}{
	{"valid-nel-report", false},
	{"valid-nel-report", true},
	{"non-nel-report", false},
	{"non-nel-report", true},
}

func TestDumpReports(t *testing.T) {
	for _, c := range dumpCases {
		t.Run("Dump:"+c.name, func(t *testing.T) {
			jsonFile := c.name + ".json"
			var dumpedFile string
			if c.useIPv6 {
				dumpedFile = c.name + ".dumped.ipv6.log"
			} else {
				dumpedFile = c.name + ".dumped.ipv4.log"
			}

			pipeline := NewTestPipeline(newSimulatedClock())
			var buffer bytes.Buffer
			pipeline.AddProcessor(ReportDumper{&buffer})
			json := testdata(t, jsonFile)

			request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(json))
			request.Header.Add("Content-Type", "application/report")
			if c.useIPv6 {
				request.RemoteAddr = "[2001:db8::2]:1234"
			}
			var response httptest.ResponseRecorder
			pipeline.ServeHTTP(&response, request)

			if response.Code != http.StatusNoContent {
				t.Errorf("ServeHTTP(%s): got %d, wanted %d", compactJSON(json), response.Code, http.StatusNoContent)
				return
			}

			got := buffer.Bytes()
			want := goldendata(t, dumpedFile, got)
			if !cmp.Equal(got, want) {
				t.Errorf("ReportDumper(%s) == %s, wanted %s", compactJSON(json), got, want)
				return
			}
		})
	}
}

// Custom annotations

var clientCountries = map[string]string{
	"192.0.2.1": "US",
	"192.0.2.2": "CA",
}

var serverZones = map[string]string{
	"203.0.113.75": "us-east1-a",
	"203.0.113.76": "us-west1-b",
}

type geoAnnotator struct{}

func (g geoAnnotator) ProcessReports(batch *ReportBatch) {
	batch.Annotation = clientCountries[batch.ClientIP]
	for i, _ := range batch.Reports {
		batch.Reports[i].Annotation = serverZones[batch.Reports[i].ServerIP]
	}
}

type stashReports struct {
	dest *ReportBatch
}

func (s stashReports) ProcessReports(batch *ReportBatch) {
	*s.dest = *batch
}

func TestCustomAnnotation(t *testing.T) {
	var got ReportBatch
	pipeline := NewTestPipeline(newSimulatedClock())
	pipeline.AddProcessor(&geoAnnotator{})
	pipeline.AddProcessor(&stashReports{&got})
	json := testdata(t, "annotation-nel-reports.json")

	request := httptest.NewRequest("POST", "https://example.com/upload/", bytes.NewReader(json))
	request.Header.Add("Content-Type", "application/report")
	var response httptest.ResponseRecorder
	pipeline.ServeHTTP(&response, request)

	if response.Code != http.StatusNoContent {
		t.Errorf("ServeHTTP(%s): got %d, wanted %d", compactJSON(json), response.Code, http.StatusNoContent)
		return
	}

	want := ReportBatch{
		Time:       time.Unix(0, 0),
		ClientIP:   "192.0.2.1",
		Annotation: "US",
		Reports: []NelReport{
			NelReport{
				Age:              500,
				ReportType:       "network-error",
				URL:              "https://example.com/about/",
				Referrer:         "https://example.com/",
				SamplingFraction: 0.5,
				ServerIP:         "203.0.113.75",
				Protocol:         "h2",
				StatusCode:       200,
				ElapsedTime:      45,
				Type:             "ok",
				Annotation:       "us-east1-a",
			},
			NelReport{
				Age:              500,
				ReportType:       "network-error",
				URL:              "https://example.com/login/",
				Referrer:         "https://example.com/",
				SamplingFraction: 0.5,
				ServerIP:         "203.0.113.76",
				Protocol:         "h2",
				StatusCode:       200,
				ElapsedTime:      45,
				Type:             "ok",
				Annotation:       "us-west1-b",
			},
		},
	}

	if !cmp.Equal(got, want) {
		t.Errorf("ReportDumper(%s) == %v, wanted %v", compactJSON(json), got, want)
		return
	}
}
