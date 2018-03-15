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
				dumpedFile = c.name + ".dumped.ipv6.json"
			} else {
				dumpedFile = c.name + ".dumped.ipv4.json"
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
